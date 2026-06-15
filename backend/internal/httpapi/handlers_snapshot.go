package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/configengine"
)

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

// handleSnapshotGet returns one snapshot's metadata plus its captured config
// state (the actual RouterOS statements) so it can be viewed/diffed.
func (s *Server) handleSnapshotGet(w http.ResponseWriter, r *http.Request) {
	snap, state, err := s.inventory.GetSnapshot(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), r.PathValue("snapshotId"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, map[string]any{
		"id":              snap.ID,
		"device_id":       snap.DeviceID,
		"source":          snap.Source,
		"statement_count": snap.StatementCount,
		"taken_at":        snap.TakenAt,
		"state":           state,
	})
}

// handleSnapshotRestore re-converges a device to a stored snapshot.
func (s *Server) handleSnapshotRestore(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	snapshotID := r.PathValue("snapshotId")
	res, plan, err := s.inventory.RestoreSnapshot(r.Context(), tenantFrom(r.Context()), deviceID, snapshotID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "config_restore", "device", deviceID, map[string]string{"snapshot": snapshotID, "status": res.Status})
	respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan})
}

type snapshotApplyRequest struct {
	State   configengine.State `json:"state"`
	MaxRisk string             `json:"max_risk,omitempty"`
}

// handleSnapshotApply pushes an edited configuration state to a device.
func (s *Server) handleSnapshotApply(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	snapshotID := r.PathValue("snapshotId")
	var req snapshotApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	opts := configengine.ApplyOptions{}
	if req.MaxRisk != "" {
		opts.MaxRisk = configengine.Risk(req.MaxRisk)
	}
	res, plan, err := s.inventory.ApplyDesiredConfig(r.Context(), tenantFrom(r.Context()), deviceID, req.State, opts)
	if err != nil {
		respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "error": err.Error()})
		return
	}
	s.record(r.Context(), "config_apply", "device", deviceID, map[string]string{"snapshot": snapshotID, "status": res.Status})
	respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan})
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
