package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/linkqos"
	"github.com/neko/sdwan/backend/internal/routing"
)

// These stateless "tools" endpoints expose the platform's planning/validation
// engines so the UI can preview results before applying anything.

func (s *Server) handleConfigDiff(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Running configengine.State       `json:"running"`
		Desired configengine.State       `json:"desired"`
		Risk    configengine.RiskOptions `json:"risk"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	plan := configengine.ComputeDiff(body.Running, body.Desired, body.Risk)
	respondData(w, http.StatusOK, plan)
}

func (s *Server) handleRoutingValidate(w http.ResponseWriter, r *http.Request) {
	var intent routing.Intent
	if err := decodeJSON(r, &intent); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	issues := routing.Validate(intent)
	respondData(w, http.StatusOK, map[string]any{
		"issues":     issues,
		"has_errors": routing.HasErrors(issues),
		"plan":       routing.BuildState(intent),
	})
}

func (s *Server) handleLinkScore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Metrics linkqos.Metrics      `json:"metrics"`
		Config  *linkqos.ScoreConfig `json:"config,omitempty"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	cfg := linkqos.DefaultScoreConfig()
	if body.Config != nil {
		cfg = *body.Config
	}
	respondData(w, http.StatusOK, map[string]any{"score": linkqos.Score(body.Metrics, cfg)})
}
