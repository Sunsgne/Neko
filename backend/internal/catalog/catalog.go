// Package catalog provides read models and an in-memory store for monitoring
// entities surfaced in the console (links, alerts, DNS servers). These are
// populated by the demo seeder and, in production, by the SNMP/probe workers
// and DNS health checker.
package catalog

import (
	"context"
	"sort"
	"sync"
)

// Link is a WAN/overlay link with its latest quality snapshot.
type Link struct {
	ID        string  `json:"id"`
	TenantID  string  `json:"tenant_id"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"` // wan | overlay
	ISP       string  `json:"isp"`
	Role      string  `json:"role"` // primary | backup
	Status    string  `json:"status"`
	LatencyMs float64 `json:"latency_ms"`
	JitterMs  float64 `json:"jitter_ms"`
	Loss      float64 `json:"loss"`
	Score     float64 `json:"score"`
}

// Alert is an active or resolved alert.
type Alert struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	DeviceID string `json:"device_id"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	State    string `json:"state"`
	FiredAt  string `json:"fired_at"`
}

// DNSServer is a pool entry shown in the DNS scheduling view.
type DNSServer struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Address     string `json:"address"`
	Region      string `json:"region"`
	ISP         string `json:"isp"`
	SupportsECS bool   `json:"supports_ecs"`
	Healthy     bool   `json:"healthy"`
	LatencyMs   int    `json:"latency_ms"`
}

// Catalog is a thread-safe in-memory holder of read models.
type Catalog struct {
	mu     sync.RWMutex
	links  []Link
	alerts []Alert
	dns    []DNSServer
}

// New returns an empty catalog.
func New() *Catalog { return &Catalog{} }

// ReplaceLinks sets the full link list (used by seeder/workers).
func (c *Catalog) ReplaceLinks(l []Link) { c.mu.Lock(); c.links = l; c.mu.Unlock() }

// ReplaceAlerts sets the full alert list.
func (c *Catalog) ReplaceAlerts(a []Alert) { c.mu.Lock(); c.alerts = a; c.mu.Unlock() }

// ReplaceDNS sets the full DNS server list.
func (c *Catalog) ReplaceDNS(d []DNSServer) { c.mu.Lock(); c.dns = d; c.mu.Unlock() }

// Links returns links scoped to tenantID ("" = all), best-first by score.
func (c *Catalog) Links(_ context.Context, tenantID string) []Link {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Link, 0, len(c.links))
	for _, l := range c.links {
		if tenantID == "" || l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// Alerts returns alerts scoped to tenantID, firing first then most recent.
func (c *Catalog) Alerts(_ context.Context, tenantID string) []Alert {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Alert, 0, len(c.alerts))
	for _, a := range c.alerts {
		if tenantID == "" || a.TenantID == tenantID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].State != out[j].State {
			return out[i].State == "firing"
		}
		return out[i].FiredAt > out[j].FiredAt
	})
	return out
}

// DNSServers returns DNS servers scoped to tenantID ("" = all + shared).
func (c *Catalog) DNSServers(_ context.Context, tenantID string) []DNSServer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]DNSServer, 0, len(c.dns))
	for _, d := range c.dns {
		if tenantID == "" || d.TenantID == "" || d.TenantID == tenantID {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LatencyMs < out[j].LatencyMs })
	return out
}
