package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/routing"
	"github.com/neko/sdwan/backend/internal/store"
)

// handleAccelModes lists the available acceleration business modes.
func (s *Server) handleAccelModes(w http.ResponseWriter, _ *http.Request) {
	modes := []map[string]string{
		{"mode": string(accel.ModeSmartSplit), "desc": accel.Describe(accel.ModeSmartSplit)},
		{"mode": string(accel.ModeChinaSplit), "desc": accel.Describe(accel.ModeChinaSplit)},
		{"mode": string(accel.ModeOverseasDirect), "desc": accel.Describe(accel.ModeOverseasDirect)},
		{"mode": string(accel.ModeDomesticDirect), "desc": accel.Describe(accel.ModeDomesticDirect)},
	}
	respondData(w, http.StatusOK, modes)
}

// handleAccelPreview builds the RouterOS config for an acceleration profile and
// returns it plus a diff/risk plan (against an empty baseline) for preview.
func (s *Server) handleAccelPreview(w http.ResponseWriter, r *http.Request) {
	var p accel.Profile
	if err := decodeJSON(r, &p); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	state, err := accel.BuildConfig(p)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}
	plan := configengine.ComputeDiff(configengine.State{}, state, configengine.RiskOptions{})
	respondData(w, http.StatusOK, map[string]any{
		"mode":  p.Mode,
		"desc":  accel.Describe(p.Mode),
		"state": state,
		"plan":  plan,
	})
}

type accelProposeRequest struct {
	CpeDeviceID     string `json:"cpe_device_id"`
	PopDeviceID     string `json:"pop_device_id"`
	Mode            string `json:"mode"`
	LocalWANGateway string `json:"local_wan_gateway,omitempty"`
	CpeOverlay      string `json:"cpe_overlay,omitempty"`
	PopPublicKey    string `json:"pop_public_key,omitempty"`
}

// handleAccelPropose generates WireGuard tunnel negotiation parameters (keys,
// overlay addressing, POP endpoint) plus the acceleration profile for CPE→POP.
func (s *Server) handleAccelPropose(w http.ResponseWriter, r *http.Request) {
	var req accelProposeRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if req.PopDeviceID == "" {
		respondError(w, http.StatusBadRequest, "invalid_input", "pop_device_id required")
		return
	}
	tenant := tenantFrom(r.Context())
	pop, err := s.inventory.Get(r.Context(), tenant, req.PopDeviceID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	var cpe *store.Device
	if req.CpeDeviceID != "" {
		cpe, err = s.inventory.Get(r.Context(), tenant, req.CpeDeviceID)
		if err != nil {
			respondServiceError(w, err)
			return
		}
	}
	mode := accel.Mode(req.Mode)
	if mode == "" {
		mode = accel.ModeOverseasDirect
	}
	proposal, err := routing.ProposeAccelToPOP(cpe, pop, mode, req.LocalWANGateway, req.CpeOverlay)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}
	if req.PopPublicKey != "" {
		proposal.Tunnel.PublicKey = req.PopPublicKey
	} else if pop.Enrolled {
		if items, err := s.inventory.RESTList(r.Context(), tenant, pop.ID, "/interface/wireguard", "", ""); err == nil {
			for _, it := range items {
				if pk, ok := it["public-key"].(string); ok && pk != "" {
					proposal.PopPublicKeyHint = pk
					proposal.Tunnel.PublicKey = pk
					break
				}
			}
		}
	}

	var fabric routing.FabricPlan
	if mode != accel.ModeDomesticDirect && cpe != nil {
		fabric, err = routing.BuildFabricPlan(cpe, pop, mode, req.LocalWANGateway, req.CpeOverlay, proposal.Tunnel.PublicKey, "", nil)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
			return
		}
		proposal = routing.FabricToProposal(fabric)
	}

	if mode == accel.ModeDomesticDirect {
		desired := mustAccelState(proposal.Accel)
		plan := configengine.ComputeDiff(configengine.State{}, desired, configengine.RiskOptions{})
		respondData(w, http.StatusOK, map[string]any{
			"proposal":    proposal,
			"desired":     desired,
			"plan":        plan,
			"cpe_desired": desired,
			"cpe_plan":    plan,
		})
		return
	}

	respondData(w, http.StatusOK, map[string]any{
		"proposal":    proposal,
		"fabric":      fabric,
		"desired":     fabric.CPEDesired,
		"plan":        fabric.CPEPlan,
		"cpe_desired": fabric.CPEDesired,
		"pop_desired": fabric.POPDesired,
		"cpe_plan":    fabric.CPEPlan,
		"pop_plan":    fabric.POPPlan,
	})
}

func mustAccelState(p accel.Profile) configengine.State {
	st, err := accel.BuildConfig(p)
	if err != nil {
		return configengine.State{}
	}
	return st
}

// handleConfigSections returns the catalog of fully-managed RouterOS sections.
func (s *Server) handleConfigSections(w http.ResponseWriter, _ *http.Request) {
	respondData(w, http.StatusOK, routeros.ManagedSections)
}

// configPushRequest pushes full configuration to a device WITHOUT logging in.
type configPushRequest struct {
	Username       string             `json:"username"`
	Password       string             `json:"password"`
	Desired        configengine.State `json:"desired"`
	Sections       []string           `json:"sections,omitempty"`
	ConfirmTimeout int                `json:"confirm_timeout_sec,omitempty"`
	MaxRisk        string             `json:"max_risk,omitempty"`
}

// handlePushConfig applies a desired configuration to a device over the
// RouterOS REST API via the config engine (snapshot→diff→apply→verify→
// confirm/rollback). No SSH/console login required.
func (s *Server) handlePushConfig(w http.ResponseWriter, r *http.Request) {
	dev, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	var req configPushRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	applier := routeros.NewApplier(routeros.Target{
		Address:  dev.MgmtAddress,
		Username: req.Username,
		Secret:   req.Password,
	}, req.Sections)

	opts := configengine.ApplyOptions{ConfirmTimeoutSec: req.ConfirmTimeout}
	if req.MaxRisk != "" {
		opts.MaxRisk = configengine.Risk(req.MaxRisk)
	}

	res, plan, err := configengine.Execute(r.Context(), applier, nil, req.Desired, opts)
	if err != nil {
		respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "error": err.Error()})
		return
	}
	s.record(r.Context(), "config_push", "device", dev.ID, map[string]string{"status": res.Status})
	respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan})
}

// handleSnapshotConfig reads the live configuration sections from a device.
func (s *Server) handleSnapshotConfig(w http.ResponseWriter, r *http.Request) {
	dev, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	user := r.URL.Query().Get("username")
	pass := r.URL.Query().Get("password")
	applier := routeros.NewApplier(routeros.Target{Address: dev.MgmtAddress, Username: user, Secret: pass}, nil)
	state, err := applier.Snapshot(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, "device_unreachable", err.Error())
		return
	}
	respondData(w, http.StatusOK, state)
}
