package httpapi

import "net/http"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.users == nil || s.sessions == nil {
		respondError(w, http.StatusNotImplemented, "auth_disabled", "authentication is not enabled")
		return
	}
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	u, err := s.users.Verify(r.Context(), req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid_credentials", "邮箱或密码错误")
		return
	}
	sess := s.sessions.Create(u.ID, u.Email, u.TenantID, u.IsOperator)
	respondData(w, http.StatusOK, map[string]any{
		"token":      sess.Token,
		"expires_at": sess.ExpiresAt,
		"user":       u.Public(),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if s.sessions != nil {
		s.sessions.Delete(bearerToken(r))
	}
	respondData(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}
	respondData(w, http.StatusOK, map[string]any{
		"id":          p.TokenID,
		"email":       p.Email,
		"tenant_id":   p.TenantID,
		"is_operator": p.IsOperator,
	})
}
