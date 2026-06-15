package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/tenant"
)

func (s *Server) requireOperator(w http.ResponseWriter, r *http.Request) bool {
	p, ok := principalFrom(r.Context())
	if !ok || !p.IsOperator {
		respondError(w, http.StatusForbidden, "forbidden", "仅运营账号可管理租户")
		return false
	}
	return true
}

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperator(w, r) {
		return
	}
	page := pageFrom(r)
	if r.URL.Query().Get("include") == "stats" {
		items, total, err := s.tenants.ListViews(r.Context(), page)
		if err != nil {
			respondServiceError(w, err)
			return
		}
		respondList(w, items, Meta{Page: page.Normalize().Number, PageSize: page.Normalize().Size, Total: total})
		return
	}
	items, total, err := s.tenants.List(r.Context(), page)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondList(w, items, Meta{Page: page.Normalize().Number, PageSize: page.Normalize().Size, Total: total})
}

func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperator(w, r) {
		return
	}
	var in tenant.CreateInput
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	t, err := s.tenants.Create(r.Context(), in)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "create", "tenant", t.ID, map[string]string{"name": t.Name, "slug": t.Slug})
	respondData(w, http.StatusCreated, t)
}

func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperator(w, r) {
		return
	}
	if r.URL.Query().Get("include") == "stats" {
		v, err := s.tenants.GetView(r.Context(), r.PathValue("id"))
		if err != nil {
			respondServiceError(w, err)
			return
		}
		respondData(w, http.StatusOK, v)
		return
	}
	t, err := s.tenants.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, t)
}

func (s *Server) handleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperator(w, r) {
		return
	}
	var in tenant.UpdateInput
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	t, err := s.tenants.Update(r.Context(), r.PathValue("id"), in)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "update", "tenant", t.ID, map[string]string{"name": t.Name, "slug": t.Slug, "status": string(t.Status)})
	respondData(w, http.StatusOK, t)
}

func (s *Server) handleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperator(w, r) {
		return
	}
	var in tenant.DeleteInput
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	id := r.PathValue("id")
	if err := s.tenants.Delete(r.Context(), id, in); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "delete", "tenant", id, map[string]string{"confirm_slug": in.ConfirmSlug})
	respondData(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}
