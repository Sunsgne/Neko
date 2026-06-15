package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/linkpolicy"
	"github.com/neko/sdwan/backend/internal/qos"
	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/routing"
)

// orchestrateRequest is the SD-WAN "站点编排 + 一键下发" payload. The core
// business is connecting a CPE/site into the fabric via an overlay tunnel to a
// backbone POP, then layering networking routes and/or acceleration on top.
type orchestrateRequest struct {
	// Tunnel builds the site↔POP overlay (e.g. WireGuard) — the SD-WAN fabric edge.
	Tunnel *routing.Tunnel `json:"tunnel,omitempty"`
	VRF    string          `json:"vrf,omitempty"`
	// OverlayRoutes are destination CIDRs reachable through the tunnel (组网).
	OverlayRoutes []string `json:"overlay_routes,omitempty"`
	// Accel applies an acceleration mode (海外直连 / 智能分流) over the tunnel.
	Accel *accel.Profile `json:"accel,omitempty"`
	// LinkPolicy is the optional local multi-WAN selection (advanced).
	LinkPolicy *linkpolicy.Policy `json:"link_policy,omitempty"`
	QoSRules   []qos.Rule         `json:"qos_rules,omitempty"`
	RateLimit  string             `json:"rate_limit,omitempty"`
	RateTarget string             `json:"rate_target,omitempty"`
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
	// 1) Overlay tunnel to the backbone POP (fabric edge).
	if req.Tunnel != nil {
		states = append(states, routing.BuildTunnelState(req.VRF, *req.Tunnel))
	}
	// 2) Networking: routes reachable through the tunnel.
	if len(req.OverlayRoutes) > 0 && req.Tunnel != nil {
		states = append(states, overlayRouteState(req.VRF, req.Tunnel.Name, req.OverlayRoutes))
	}
	// 3) Acceleration over the tunnel.
	if req.Accel != nil {
		st, err := accel.BuildConfig(*req.Accel)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_accel", err.Error())
			return
		}
		states = append(states, st)
	}
	// 4) Optional local multi-WAN selection (advanced).
	if req.LinkPolicy != nil {
		st, err := linkpolicy.BuildConfig(*req.LinkPolicy)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_link_policy", err.Error())
			return
		}
		states = append(states, st)
	}
	if len(states) == 0 {
		respondError(w, http.StatusBadRequest, "empty_intent", "需要 tunnel / accel / link_policy 至少一项")
		return
	}
	desired := configengine.Merge(states...)

	qosRules := req.QoSRules
	if req.RateLimit != "" {
		auto, err := qos.RulesForSite(dev.Name, req.OverlayRoutes, req.RateLimit, req.RateTarget)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_qos", err.Error())
			return
		}
		qosRules = append(qosRules, auto...)
	}
	if len(qosRules) > 0 {
		var err error
		desired, err = qos.MergeState(desired, qosRules)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_qos", err.Error())
			return
		}
	}

	// Preview: diff against an empty baseline (the device's running config is
	// only fetched when actually delivering).
	if req.DryRun {
		plan := configengine.ComputeDiff(configengine.State{}, desired, configengine.RiskOptions{})
		respondData(w, http.StatusOK, map[string]any{"dry_run": true, "desired": desired, "plan": plan})
		return
	}

	opts := configengine.ApplyOptions{ConfirmTimeoutSec: req.ConfirmTimeout}
	if req.MaxRisk != "" {
		opts.MaxRisk = configengine.Risk(req.MaxRisk)
	}

	if req.Username == "" {
		res, plan, err := s.inventory.ApplyDesiredConfig(r.Context(), tenantFrom(r.Context()), dev.ID, desired, opts)
		if err != nil {
			respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "error": err.Error()})
			return
		}
		s.record(r.Context(), "orchestrate", "device", dev.ID, map[string]string{"status": res.Status})
		respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan})
		return
	}

	applier := routeros.NewApplier(routeros.Target{
		Address:  dev.MgmtAddress,
		Username: req.Username,
		Secret:   req.Password,
	}, nil)

	res, plan, err := configengine.Execute(r.Context(), applier, nil, desired, opts)
	if err != nil {
		respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "error": err.Error()})
		return
	}
	s.record(r.Context(), "orchestrate", "device", dev.ID, map[string]string{"status": res.Status})
	respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan})
}

// overlayRouteState routes the given CIDRs through the overlay tunnel interface.
func overlayRouteState(vrf, iface string, cidrs []string) configengine.State {
	var sts []configengine.Statement
	for _, c := range cidrs {
		attrs := map[string]string{
			"dst-address": c,
			"gateway":     iface,
			"distance":    "1",
			"comment":     "neko-fabric: route via " + iface,
		}
		if vrf != "" {
			attrs["routing-table"] = vrf
		}
		sts = append(sts, configengine.Statement{Path: "/ip/route", Key: c + "@" + iface, Attributes: attrs})
	}
	return configengine.State{Statements: sts}
}
