package httpapi

import "net/http"

func (s *Server) handleListLinks(w http.ResponseWriter, r *http.Request) {
	if s.catalog == nil {
		respondList(w, []any{}, Meta{Page: 1, PageSize: 20, Total: 0})
		return
	}
	items := s.catalog.Links(r.Context(), tenantFrom(r.Context()))
	respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
}

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	// Prefer persisted, monitoring-generated alerts; fall back to catalog
	// (demo) when no alert store is configured.
	if s.alerts != nil {
		items, err := s.alerts.List(r.Context(), tenantFrom(r.Context()), 200)
		if err != nil {
			respondServiceError(w, err)
			return
		}
		respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
		return
	}
	if s.catalog == nil {
		respondList(w, []any{}, Meta{Page: 1, PageSize: 20, Total: 0})
		return
	}
	items := s.catalog.Alerts(r.Context(), tenantFrom(r.Context()))
	respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
}

func (s *Server) handleListDNSServers(w http.ResponseWriter, r *http.Request) {
	items, err := s.dns.List(r.Context(), tenantFrom(r.Context()))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
}
