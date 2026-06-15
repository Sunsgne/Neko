package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/routing"
)

type fabricDeployRequest struct {
	CpeDeviceID     string   `json:"cpe_device_id"`
	PopDeviceID     string   `json:"pop_device_id"`
	Mode            string   `json:"mode"`
	LocalWANGateway string   `json:"local_wan_gateway,omitempty"`
	CpeOverlay      string   `json:"cpe_overlay,omitempty"`
	PopPublicKey    string   `json:"pop_public_key,omitempty"`
	CpePublicKey    string   `json:"cpe_public_key,omitempty"`
	OverlayRoutes   []string `json:"overlay_routes,omitempty"`
	DryRun          bool     `json:"dry_run"`
	ConfirmTimeout  int      `json:"confirm_timeout_sec"`
	MaxRisk         string   `json:"max_risk"`
}

// handleFabricDeploy generates a bilateral CPE↔POP WireGuard + acceleration
// plan and either previews it (dry_run) or delivers to POP then CPE using
// enrolled credentials (no manual login).
func (s *Server) handleFabricDeploy(w http.ResponseWriter, r *http.Request) {
	var req fabricDeployRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if req.CpeDeviceID == "" || req.PopDeviceID == "" {
		respondError(w, http.StatusBadRequest, "invalid_input", "cpe_device_id and pop_device_id required")
		return
	}
	tenant := tenantFrom(r.Context())
	cpe, err := s.inventory.Get(r.Context(), tenant, req.CpeDeviceID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	pop, err := s.inventory.Get(r.Context(), tenant, req.PopDeviceID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	mode := accel.Mode(req.Mode)
	if req.Mode == "mesh" {
		mode = ""
	} else if req.Mode == "china_split" {
		mode = accel.ModeChinaSplit
	} else if mode == "" {
		mode = accel.ModeOverseasDirect
	}
	popPubHint := req.PopPublicKey
	if popPubHint == "" && pop.Enrolled {
		if items, err := s.inventory.RESTList(r.Context(), tenant, pop.ID, "/interface/wireguard", "", ""); err == nil {
			for _, it := range items {
				if pk, ok := it["public-key"].(string); ok && pk != "" {
					popPubHint = pk
					break
				}
			}
		}
	}
	plan, err := routing.BuildFabricPlan(cpe, pop, mode, req.LocalWANGateway, req.CpeOverlay, popPubHint, req.CpePublicKey, req.OverlayRoutes)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}
	proposal := routing.FabricToProposal(plan)

	if req.DryRun {
		respondData(w, http.StatusOK, map[string]any{
			"dry_run":     true,
			"fabric":      plan,
			"proposal":    proposal,
			"cpe_desired": plan.CPEDesired,
			"pop_desired": plan.POPDesired,
			"cpe_plan":    plan.CPEPlan,
			"pop_plan":    plan.POPPlan,
		})
		return
	}

	opts := configengine.ApplyOptions{ConfirmTimeoutSec: req.ConfirmTimeout}
	if req.MaxRisk != "" {
		opts.MaxRisk = configengine.Risk(req.MaxRisk)
	}
	result, err := s.inventory.FabricDeploy(r.Context(), tenant, cpe.ID, pop.ID, plan.CPEDesired, plan.POPDesired, opts)
	resp := map[string]any{
		"fabric":      plan,
		"proposal":    proposal,
		"cpe_desired": plan.CPEDesired,
		"pop_desired": plan.POPDesired,
		"deploy":      result,
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	s.record(r.Context(), "fabric_deploy", "device", cpe.ID, map[string]string{
		"pop_id": pop.ID,
		"status": result.CPEResult.Status,
	})
	respondData(w, http.StatusOK, resp)
}
