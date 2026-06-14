package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/neko/sdwan/backend/internal/vmetrics"
)

// handleDeviceMetrics returns recent CPU and memory time series for a device
// from VictoriaMetrics, powering the device detail charts.
func (s *Server) handleDeviceMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Ensure the device is visible in the caller's tenant scope.
	if _, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), id); err != nil {
		respondServiceError(w, err)
		return
	}
	if s.vm == nil || !s.vm.Enabled() {
		respondData(w, http.StatusOK, map[string]any{"enabled": false, "series": []vmetrics.Series{}})
		return
	}

	end := time.Now()
	start := end.Add(-1 * time.Hour)
	step := 60 * time.Second

	cpu, _ := s.vm.QueryRange(r.Context(), "cpu", fmt.Sprintf(`neko_device_cpu{device=%q}`, id), start, end, step)
	mem, _ := s.vm.QueryRange(r.Context(), "mem", fmt.Sprintf(`neko_device_mem_ratio{device=%q}*100`, id), start, end, step)
	online, _ := s.vm.QueryRange(r.Context(), "online", fmt.Sprintf(`neko_device_online{device=%q}`, id), start, end, step)

	respondData(w, http.StatusOK, map[string]any{
		"enabled": true,
		"series":  []vmetrics.Series{cpu, mem, online},
	})
}
