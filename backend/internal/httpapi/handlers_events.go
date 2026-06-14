package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleEvents is a Server-Sent Events stream emitting a periodic summary
// (firing alerts, device counts) so the console can update live without
// polling each resource. Clients reconnect automatically per the SSE spec.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "no_stream", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	tenant := tenantFrom(r.Context())
	ctx := r.Context()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	emit := func() bool {
		sum := s.summary(ctx, tenant)
		b, _ := json.Marshal(sum)
		if _, err := fmt.Fprintf(w, "event: summary\ndata: %s\n\n", b); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !emit() {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !emit() {
				return
			}
		}
	}
}

type liveSummary struct {
	FiringAlerts   int   `json:"firing_alerts"`
	CriticalAlerts int   `json:"critical_alerts"`
	Devices        int   `json:"devices"`
	OnlineDevices  int   `json:"online_devices"`
	TS             int64 `json:"ts"`
}

func (s *Server) summary(ctx context.Context, tenant string) liveSummary {
	out := liveSummary{TS: time.Now().Unix()}
	if s.alerts != nil {
		if alerts, err := s.alerts.List(ctx, tenant, 500); err == nil {
			for _, a := range alerts {
				if a.State == "firing" {
					out.FiringAlerts++
					if a.Severity == "critical" {
						out.CriticalAlerts++
					}
				}
			}
		}
	}
	if devs, err := s.inventory.ListByRole(ctx, tenant, ""); err == nil {
		out.Devices = len(devs)
		for _, d := range devs {
			if d.Status != nil && d.Status.Online {
				out.OnlineDevices++
			}
		}
	}
	return out
}
