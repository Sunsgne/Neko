package httpapi

import (
	"net/http"
	"strconv"

	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/store"
)

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	respondData(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	respondData(w, http.StatusOK, map[string]string{"status": "ready", "store": s.storeKind})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(s.metrics.Expose()))
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	// Optional role filter (?role=backbone|cpe|gateway) for e.g. the backbone
	// node management view.
	if role := r.URL.Query().Get("role"); role != "" {
		items, err := s.inventory.ListByRole(r.Context(), tenantFrom(r.Context()), store.DeviceRole(role))
		if err != nil {
			respondServiceError(w, err)
			return
		}
		respondList(w, items, Meta{Page: 1, PageSize: len(items), Total: len(items)})
		return
	}
	page := pageFrom(r)
	items, total, err := s.inventory.List(r.Context(), tenantFrom(r.Context()), page)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondList(w, items, Meta{Page: page.Normalize().Number, PageSize: page.Normalize().Size, Total: total})
}

func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var in inventory.RegisterInput
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	d, err := s.inventory.Register(r.Context(), tenantFrom(r.Context()), in)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "create", "device", d.ID, map[string]string{"name": d.Name, "mgmt_address": d.MgmtAddress, "role": string(d.Role)})
	respondData(w, http.StatusCreated, d)
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	d, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, d)
}

func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.inventory.Delete(r.Context(), tenantFrom(r.Context()), id); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "delete", "device", id, nil)
	respondData(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (s *Server) handleDetectDevice(w http.ResponseWriter, r *http.Request) {
	d, err := s.inventory.Detect(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, d)
}

func (s *Server) handleSetDeviceTrust(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TrustState store.TrustState `json:"trust_state"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	d, err := s.inventory.SetTrustState(r.Context(), tenantFrom(r.Context()), r.PathValue("id"), body.TrustState)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "trust_change", "device", d.ID, map[string]string{"trust_state": string(d.TrustState)})
	respondData(w, http.StatusOK, d)
}

func pageFrom(r *http.Request) store.Page {
	q := r.URL.Query()
	num, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("page_size"))
	return store.Page{Number: num, Size: size}
}
