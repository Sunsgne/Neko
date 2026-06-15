package httpapi

import (
	"net/http"
	"strings"

	"github.com/neko/sdwan/backend/internal/routeros"
)

// handleConfigCatalog returns the grouped catalog of RouterOS sections so the
// console can render the full WebFig-style menu tree for remote configuration.
func (s *Server) handleConfigCatalog(w http.ResponseWriter, _ *http.Request) {
	respondData(w, http.StatusOK, routeros.Catalog)
}

// restWriteRequest is the body for create/update generic config calls.
type restWriteRequest struct {
	Path       string            `json:"path"`
	ItemID     string            `json:"item_id,omitempty"`
	Attributes map[string]string `json:"attributes"`
	// Singleton marks a settings resource (e.g. /ip/dns) edited via "set".
	Singleton bool `json:"singleton,omitempty"`
	// Username/Password are optional; when omitted the device's stored
	// (enrolled) credentials are used (远程配置无需登录设备).
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p != "" && !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// handleRESTList lists items at an arbitrary RouterOS path on a device, using
// the device's stored credentials (or ?username/&password when provided).
func (s *Server) handleRESTList(w http.ResponseWriter, r *http.Request) {
	path := cleanPath(r.URL.Query().Get("path"))
	if !routeros.ValidPath(path) {
		respondError(w, http.StatusBadRequest, "invalid_path", "path must be a RouterOS resource path like /ip/address")
		return
	}
	items, err := s.inventory.RESTList(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), path,
		r.URL.Query().Get("username"), r.URL.Query().Get("password"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, map[string]any{"path": path, "items": items})
}

// handleRESTCreate adds an item at an arbitrary RouterOS path.
func (s *Server) handleRESTCreate(w http.ResponseWriter, r *http.Request) {
	var req restWriteRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	path := cleanPath(req.Path)
	if !routeros.ValidPath(path) {
		respondError(w, http.StatusBadRequest, "invalid_path", "path must be a RouterOS resource path like /ip/address")
		return
	}
	if req.Singleton {
		if err := s.inventory.RESTSet(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), path, req.Attributes, req.Username, req.Password); err != nil {
			respondServiceError(w, err)
			return
		}
		s.record(r.Context(), "config_set", "device", r.PathValue("id"), map[string]string{"path": path})
		respondData(w, http.StatusOK, map[string]any{"status": "updated", "path": path})
		return
	}
	if err := s.inventory.RESTCreate(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), path, req.Attributes, req.Username, req.Password); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "config_create", "device", r.PathValue("id"), map[string]string{"path": path})
	respondData(w, http.StatusCreated, map[string]any{"status": "created", "path": path})
}

// handleRESTUpdate modifies an item (by RouterOS .id) at an arbitrary path.
func (s *Server) handleRESTUpdate(w http.ResponseWriter, r *http.Request) {
	var req restWriteRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	path := cleanPath(req.Path)
	if !routeros.ValidPath(path) || req.ItemID == "" {
		respondError(w, http.StatusBadRequest, "invalid_path", "valid path and item_id are required")
		return
	}
	if err := s.inventory.RESTUpdate(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), path, req.ItemID, req.Attributes, req.Username, req.Password); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "config_update", "device", r.PathValue("id"), map[string]string{"path": path, "item": req.ItemID})
	respondData(w, http.StatusOK, map[string]any{"status": "updated", "path": path, "item_id": req.ItemID})
}

// handleRESTDelete removes an item (by RouterOS .id) at an arbitrary path.
func (s *Server) handleRESTDelete(w http.ResponseWriter, r *http.Request) {
	path := cleanPath(r.URL.Query().Get("path"))
	itemID := r.URL.Query().Get("item_id")
	if !routeros.ValidPath(path) || itemID == "" {
		respondError(w, http.StatusBadRequest, "invalid_path", "valid path and item_id are required")
		return
	}
	if err := s.inventory.RESTDelete(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), path, itemID,
		r.URL.Query().Get("username"), r.URL.Query().Get("password")); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "config_delete", "device", r.PathValue("id"), map[string]string{"path": path, "item": itemID})
	respondData(w, http.StatusOK, map[string]any{"status": "deleted", "path": path, "item_id": itemID})
}
