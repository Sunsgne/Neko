package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/linkpolicy"
	"github.com/neko/sdwan/backend/internal/routeros"
)

// orchestrateRequest is the unified "站点编排 + 一键下发" payload: link
// selection + (optional) acceleration mode, previewed or pushed to a device.
type orchestrateRequest struct {
	LinkPolicy *linkpolicy.Policy `json:"link_policy,omitempty"`
	Accel      *accel.Profile     `json:"accel,omitempty"`
	// DryRun=true returns the generated config + plan without touching the
	// device (preview). Otherwise the config is pushed over RouterOS REST.
	DryRun         bool   `json:"dry_run"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	ConfirmTimeout int    `json:"confirm_timeout_sec"`
	MaxRisk        string `json:"max_risk"`
}

// handleOrchestrate composes the requested intents into a single desired
// configuration and either previews it (dry_run) or delivers it to the device
// via the config engine (snapshot→diff→apply→verify→confirm/rollback).
func (s *Server) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	dev, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	var req orchestrateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}

	var states []configengine.State
	if req.LinkPolicy != nil {
		st, err := linkpolicy.BuildConfig(*req.LinkPolicy)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_link_policy", err.Error())
			return
		}
		states = append(states, st)
	}
	if req.Accel != nil {
		st, err := accel.BuildConfig(*req.Accel)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_accel", err.Error())
			return
		}
		states = append(states, st)
	}
	if len(states) == 0 {
		respondError(w, http.StatusBadRequest, "empty_intent", "需要 link_policy 或 accel 至少一项")
		return
	}
	desired := configengine.Merge(states...)

	// Preview: diff against an empty baseline (the device's running config is
	// only fetched when actually delivering).
	if req.DryRun {
		plan := configengine.ComputeDiff(configengine.State{}, desired, configengine.RiskOptions{})
		respondData(w, http.StatusOK, map[string]any{"dry_run": true, "desired": desired, "plan": plan})
		return
	}

	applier := routeros.NewApplier(routeros.Target{
		Address:  dev.MgmtAddress,
		Username: req.Username,
		Secret:   req.Password,
	}, nil)
	opts := configengine.ApplyOptions{ConfirmTimeoutSec: req.ConfirmTimeout}
	if req.MaxRisk != "" {
		opts.MaxRisk = configengine.Risk(req.MaxRisk)
	}

	res, plan, err := configengine.Execute(r.Context(), applier, nil, desired, opts)
	if err != nil {
		respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "error": err.Error()})
		return
	}
	s.record(r.Context(), "orchestrate", "device", dev.ID, map[string]string{"status": res.Status})
	respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan})
}
