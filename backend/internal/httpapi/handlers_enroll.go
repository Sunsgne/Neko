package httpapi

import "net/http"

type enrollRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleEnroll stores device credentials (encrypted), connects to pull the
// device's facts/capabilities, and transitions it to managed (托管).
func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	var req enrollRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if req.Username == "" {
		respondError(w, http.StatusBadRequest, "invalid_input", "username is required")
		return
	}
	d, err := s.inventory.Enroll(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), req.Username, req.Password)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "enroll", "device", d.ID, map[string]string{"platform": string(d.Platform), "model": d.Model})
	respondData(w, http.StatusOK, d)
}

// handlePoll refreshes a managed device's live status using stored credentials.
func (s *Server) handlePoll(w http.ResponseWriter, r *http.Request) {
	d, err := s.inventory.Poll(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, d)
}
