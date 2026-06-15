package httpapi

import (
	"net/http"
	"strings"

	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/qos"
	"github.com/neko/sdwan/backend/internal/store"
)

type createQoSPolicyRequest struct {
	Name       string `json:"name"`
	Target     string `json:"target"`
	MaxLimit   string `json:"max_limit"`
	LimitAt    string `json:"limit_at,omitempty"`
	BurstLimit string `json:"burst_limit,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

func (s *Server) handleListQoSPolicies(w http.ResponseWriter, r *http.Request) {
	items, err := s.qos.List(r.Context(), tenantFrom(r.Context()))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
}

func (s *Server) handleCreateQoSPolicy(w http.ResponseWriter, r *http.Request) {
	var req createQoSPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	rule := qos.Rule{
		Name:       strings.TrimSpace(req.Name),
		Target:     strings.TrimSpace(req.Target),
		MaxLimit:   strings.TrimSpace(req.MaxLimit),
		LimitAt:    strings.TrimSpace(req.LimitAt),
		BurstLimit: strings.TrimSpace(req.BurstLimit),
		Priority:   req.Priority,
		Comment:    strings.TrimSpace(req.Comment),
	}
	if err := qos.ValidateRule(rule); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	p := store.QoSPolicy{
		ID:         s.idgen("qos"),
		TenantID:   tenantFrom(r.Context()),
		Name:       rule.Name,
		Target:     rule.Target,
		MaxLimit:   qos.NormalizeRate(rule.MaxLimit),
		LimitAt:    rule.LimitAt,
		BurstLimit: rule.BurstLimit,
		Priority:   rule.Priority,
		Comment:    rule.Comment,
	}
	if err := s.qos.Create(r.Context(), p); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "create", "qos_policy", p.ID, map[string]string{"name": p.Name})
	respondData(w, http.StatusCreated, p)
}

func (s *Server) handleDeleteQoSPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.qos.Delete(r.Context(), tenantFrom(r.Context()), id); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "delete", "qos_policy", id, nil)
	respondData(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

type qosApplyRequest struct {
	PolicyIDs []string  `json:"policy_ids"`
	Rules     []qos.Rule `json:"rules,omitempty"`
	DryRun    bool      `json:"dry_run"`
}

func (s *Server) handleQoSApply(w http.ResponseWriter, r *http.Request) {
	dev, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	var req qosApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}

	var rules []qos.Rule
	if len(req.PolicyIDs) > 0 {
		pool, err := s.qos.List(r.Context(), tenantFrom(r.Context()))
		if err != nil {
			respondServiceError(w, err)
			return
		}
		byID := map[string]*store.QoSPolicy{}
		for _, p := range pool {
			byID[p.ID] = p
		}
		for _, id := range req.PolicyIDs {
			p, ok := byID[id]
			if !ok {
				continue
			}
			rules = append(rules, qos.Rule{
				Name: p.Name, Target: p.Target, MaxLimit: p.MaxLimit,
				LimitAt: p.LimitAt, BurstLimit: p.BurstLimit, Priority: p.Priority, Comment: p.Comment,
			})
		}
	}
	rules = append(rules, req.Rules...)
	if len(rules) == 0 {
		respondError(w, http.StatusBadRequest, "empty", "请至少选择一条限速策略")
		return
	}

	desired, err := qos.BuildSimpleQueues(rules)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}

	if req.DryRun {
		plan := configengine.ComputeDiff(configengine.State{}, desired, configengine.RiskOptions{})
		respondData(w, http.StatusOK, map[string]any{"dry_run": true, "desired": desired, "plan": plan, "rule_count": len(rules)})
		return
	}

	res, plan, applyErr := s.inventory.ApplyDesiredConfig(r.Context(), tenantFrom(r.Context()), dev.ID, desired, configengine.ApplyOptions{})
	if applyErr != nil {
		respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "error": applyErr.Error()})
		return
	}
	s.record(r.Context(), "qos_apply", "device", dev.ID, map[string]string{"count": itoa(len(rules))})
	respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "rule_count": len(rules)})
}

// handleQoSPreview builds queue config without a device (policy validation).
func (s *Server) handleQoSPreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rules []qos.Rule `json:"rules"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	desired, err := qos.BuildSimpleQueues(req.Rules)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}
	plan := configengine.ComputeDiff(configengine.State{}, desired, configengine.RiskOptions{})
	respondData(w, http.StatusOK, map[string]any{"desired": desired, "plan": plan})
}
