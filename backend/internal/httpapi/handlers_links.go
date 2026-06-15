package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/neko/sdwan/backend/internal/store"
)

// handleListLinks returns persisted, actively-measured links for the tenant.
func (s *Server) handleListLinks(w http.ResponseWriter, r *http.Request) {
	if s.links == nil {
		respondList(w, []any{}, Meta{Page: 1, PageSize: 0, Total: 0})
		return
	}
	items, err := s.links.List(r.Context(), tenantFrom(r.Context()))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
}

type createLinkRequest struct {
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`   // wan | overlay
	ISP      string `json:"isp"`    // telecom | unicom | mobile | edu | overlay
	Role     string `json:"role"`   // primary | backup
	Target   string `json:"target"` // host/IP probed from the device
}

// handleCreateLink registers a monitored link on a device.
func (s *Server) handleCreateLink(w http.ResponseWriter, r *http.Request) {
	if s.links == nil {
		respondError(w, http.StatusServiceUnavailable, "unavailable", "link store not configured")
		return
	}
	var req createLinkRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	tenant := tenantFrom(r.Context())
	dev, err := s.inventory.Get(r.Context(), tenant, strings.TrimSpace(req.DeviceID))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_device", "device not found")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Target) == "" {
		respondError(w, http.StatusBadRequest, "invalid_input", "name 与 target 必填")
		return
	}
	kind := req.Kind
	if kind != "overlay" {
		kind = "wan"
	}
	role := req.Role
	if role != "backup" {
		role = "primary"
	}
	link := store.Link{
		ID:       s.idgen("link"),
		TenantID: dev.TenantID,
		DeviceID: dev.ID,
		Name:     strings.TrimSpace(req.Name),
		Kind:     kind,
		ISP:      strings.TrimSpace(req.ISP),
		Role:     role,
		Target:   strings.TrimSpace(req.Target),
		Status:   "unknown",
	}
	if err := s.links.Create(r.Context(), link); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "create", "link", link.ID, map[string]string{"name": link.Name, "target": link.Target})
	respondData(w, http.StatusCreated, link)
}

// handleDeleteLink removes a monitored link.
func (s *Server) handleDeleteLink(w http.ResponseWriter, r *http.Request) {
	if s.links == nil {
		respondError(w, http.StatusServiceUnavailable, "unavailable", "link store not configured")
		return
	}
	id := r.PathValue("id")
	if err := s.links.Delete(r.Context(), tenantFrom(r.Context()), id); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "delete", "link", id, nil)
	respondData(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// handleProbeLink measures a link on-demand (pings target from the device),
// persists the result, and returns the updated link.
func (s *Server) handleProbeLink(w http.ResponseWriter, r *http.Request) {
	if s.links == nil {
		respondError(w, http.StatusServiceUnavailable, "unavailable", "link store not configured")
		return
	}
	tenant := tenantFrom(r.Context())
	link, err := s.links.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	if tenant != "" && link.TenantID != "" && link.TenantID != tenant {
		respondError(w, http.StatusNotFound, "not_found", "link not found")
		return
	}
	now := time.Now().UTC()
	m, err := s.inventory.MeasureLink(r.Context(), tenant, link.DeviceID, link.Target, "", 5)
	if err != nil {
		// Unreachable: record as down so the console reflects reality.
		_ = s.links.UpdateMeasurement(r.Context(), link.ID, "down", 0, 0, 1, 0, now)
		link.Status = "down"
		link.Loss = 1
		link.Score = 0
		link.MeasuredAt = &now
		respondData(w, http.StatusOK, map[string]any{"link": link, "error": err.Error()})
		return
	}
	if err := s.links.UpdateMeasurement(r.Context(), link.ID, m.Status, m.Metrics.LatencyMs, m.Metrics.JitterMs, m.Metrics.Loss, m.Score, now); err != nil {
		respondServiceError(w, err)
		return
	}
	link.Status = m.Status
	link.LatencyMs = m.Metrics.LatencyMs
	link.JitterMs = m.Metrics.JitterMs
	link.Loss = m.Metrics.Loss
	link.Score = m.Score
	link.MeasuredAt = &now
	s.record(r.Context(), "probe", "link", link.ID, map[string]string{"status": m.Status})
	respondData(w, http.StatusOK, map[string]any{"link": link})
}
