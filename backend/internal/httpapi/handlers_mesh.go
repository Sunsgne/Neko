package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/qos"
	"github.com/neko/sdwan/backend/internal/routing"
	"github.com/neko/sdwan/backend/internal/store"
)

type meshDeployRequest struct {
	Topology       string                `json:"topology"`
	LocalAS        uint32                `json:"local_as,omitempty"`
	Sites          []routing.MeshSiteInput `json:"sites"`
	BackbonePath   []string              `json:"backbone_path,omitempty"`
	RRDeviceID     string                `json:"rr_device_id,omitempty"`
	DryRun         bool                  `json:"dry_run"`
	ConfirmTimeout int                   `json:"confirm_timeout_sec"`
	MaxRisk        string                `json:"max_risk"`
}

// handleMeshDeploy plans (or delivers) multi-site SD-WAN mesh: hub-spoke,
// transit (D→B→B→D), or backbone full-mesh with BGP on overlay.
func (s *Server) handleMeshDeploy(w http.ResponseWriter, r *http.Request) {
	var req meshDeployRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if len(req.Sites) < 2 {
		respondError(w, http.StatusBadRequest, "invalid_input", "至少需要 2 个站点")
		return
	}

	tenant := tenantFrom(r.Context())
	devices := make(map[string]*store.Device)
	collect := func(id string) error {
		if id == "" || devices[id] != nil {
			return nil
		}
		d, err := s.inventory.Get(r.Context(), tenant, id)
		if err != nil {
			return err
		}
		devices[id] = d
		return nil
	}
	for _, site := range req.Sites {
		if err := collect(site.DeviceID); err != nil {
			respondServiceError(w, err)
			return
		}
		if err := collect(site.PopDeviceID); err != nil {
			respondServiceError(w, err)
			return
		}
	}
	for _, id := range req.BackbonePath {
		if err := collect(id); err != nil {
			respondServiceError(w, err)
			return
		}
	}
	if req.RRDeviceID != "" {
		if err := collect(req.RRDeviceID); err != nil {
			respondServiceError(w, err)
			return
		}
	}

	plan, err := routing.BuildMeshPlan(devices, routing.MeshPlanInput{
		Topology:     routing.MeshTopology(req.Topology),
		LocalAS:      req.LocalAS,
		Sites:        req.Sites,
		BackbonePath: req.BackbonePath,
		RRDeviceID:   req.RRDeviceID,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}
	if err := qos.ApplyToMeshPlan(&plan, req.Sites, devices); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_qos", err.Error())
		return
	}

	if req.DryRun {
		respondData(w, http.StatusOK, map[string]any{
			"dry_run": true,
			"mesh":    plan,
		})
		return
	}

	opts := configengine.ApplyOptions{ConfirmTimeoutSec: req.ConfirmTimeout}
	if req.MaxRisk != "" {
		opts.MaxRisk = configengine.Risk(req.MaxRisk)
	}
	result, err := s.inventory.MeshDeploy(r.Context(), tenant, plan.Nodes, opts)
	resp := map[string]any{
		"mesh":   plan,
		"deploy": result,
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	s.record(r.Context(), "mesh_deploy", "mesh", req.Topology, map[string]string{
		"sites":  itoa(len(req.Sites)),
		"failed": itoa(result.Failed),
	})
	respondData(w, http.StatusOK, resp)
}
