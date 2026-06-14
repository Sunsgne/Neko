package httpapi

import "net/http"

// handleSnapshotSave captures and stores a config backup snapshot of a device.
func (s *Server) handleSnapshotSave(w http.ResponseWriter, r *http.Request) {
	snap, _, err := s.inventory.SnapshotConfig(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), "manual")
	if err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "config_snapshot", "device", r.PathValue("id"), map[string]string{"snapshot": snap.ID})
	respondData(w, http.StatusOK, snap)
}

// handleSnapshotList lists a device's config backup history.
func (s *Server) handleSnapshotList(w http.ResponseWriter, r *http.Request) {
	items, err := s.inventory.ListSnapshots(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), 50)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
}

// handleDrift reports config drift between the two most recent snapshots.
func (s *Server) handleDrift(w http.ResponseWriter, r *http.Request) {
	res, err := s.inventory.Drift(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, res)
}
