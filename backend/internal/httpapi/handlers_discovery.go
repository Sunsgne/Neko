package httpapi

import (
	"net/http"

	"github.com/neko/sdwan/backend/internal/discovery"
	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/store"
)

type discoverRequest struct {
	CIDR     string `json:"cidr"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleDiscover scans an IP range for reachable RouterOS devices.
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	var req discoverRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	cands, err := discovery.Scan(r.Context(), discovery.Options{
		CIDR: req.CIDR, Port: req.Port, Username: req.Username, Password: req.Password, MaxHosts: 1024,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "scan_failed", err.Error())
		return
	}
	respondList(w, cands, Meta{Page: 1, PageSize: len(cands), Total: len(cands)})
}

type batchItem struct {
	Name        string           `json:"name"`
	MgmtAddress string           `json:"mgmt_address"`
	Role        store.DeviceRole `json:"role"`
	Region      string           `json:"region"`
}

type batchRequest struct {
	Devices []batchItem `json:"devices"`
	// When Username is set, each created device is also enrolled immediately.
	Username string `json:"username"`
	Password string `json:"password"`
}

type batchResultItem struct {
	Name     string `json:"name"`
	DeviceID string `json:"device_id,omitempty"`
	Enrolled bool   `json:"enrolled"`
	Error    string `json:"error,omitempty"`
}

// handleBatchOnboard creates (and optionally enrolls) multiple devices.
func (s *Server) handleBatchOnboard(w http.ResponseWriter, r *http.Request) {
	var req batchRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if len(req.Devices) == 0 {
		respondError(w, http.StatusBadRequest, "empty", "devices list is empty")
		return
	}
	tenant := tenantFrom(r.Context())
	results := make([]batchResultItem, 0, len(req.Devices))
	created, enrolled := 0, 0
	for _, it := range req.Devices {
		res := batchResultItem{Name: it.Name}
		d, err := s.inventory.Register(r.Context(), tenant, inventory.RegisterInput{
			Name: it.Name, MgmtAddress: it.MgmtAddress, Role: it.Role, Region: it.Region,
		})
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}
		res.DeviceID = d.ID
		created++
		if req.Username != "" {
			if _, eerr := s.inventory.Enroll(r.Context(), tenant, d.ID, req.Username, req.Password); eerr != nil {
				res.Error = "enroll: " + eerr.Error()
			} else {
				res.Enrolled = true
				enrolled++
			}
		}
		results = append(results, res)
	}
	s.record(r.Context(), "batch_onboard", "device", "", map[string]string{"created": itoa(created), "enrolled": itoa(enrolled)})
	respondData(w, http.StatusOK, map[string]any{"created": created, "enrolled": enrolled, "results": results})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
