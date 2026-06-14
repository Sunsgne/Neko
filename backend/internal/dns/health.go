package dns

import (
	"context"
	"net"
	"time"
)

// HealthResult is the outcome of probing a DNS server.
type HealthResult struct {
	ServerID  string `json:"server_id"`
	Healthy   bool   `json:"healthy"`
	LatencyMs int    `json:"latency_ms"`
	Err       string `json:"err,omitempty"`
}

// Checker probes DNS servers for reachability, correctness and latency by
// resolving a known probe domain against each server directly.
type Checker struct {
	Probe   string // domain to resolve, e.g. "www.qq.com"
	Timeout time.Duration
}

// NewChecker builds a checker with defaults.
func NewChecker() *Checker {
	return &Checker{Probe: "www.qq.com", Timeout: 2 * time.Second}
}

// Check resolves the probe domain against a single server (address may be
// "ip" or "ip:port"; port 53 assumed). It measures latency and verifies that
// at least one A/AAAA record is returned.
func (c *Checker) Check(ctx context.Context, server Server) HealthResult {
	addr := server.Address
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "53")
	}
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: c.Timeout}
			return d.DialContext(ctx, "udp", addr)
		},
	}
	cctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	start := time.Now()
	ips, err := r.LookupHost(cctx, c.Probe)
	latency := int(time.Since(start).Milliseconds())

	res := HealthResult{ServerID: server.ID, LatencyMs: latency}
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Healthy = len(ips) > 0
	if !res.Healthy {
		res.Err = "no records"
	}
	return res
}

// CheckAll probes every server concurrently and returns results keyed by id.
func (c *Checker) CheckAll(ctx context.Context, servers []Server) map[string]HealthResult {
	out := make(map[string]HealthResult, len(servers))
	type pair struct {
		id  string
		res HealthResult
	}
	ch := make(chan pair, len(servers))
	for _, s := range servers {
		go func(s Server) { ch <- pair{s.ID, c.Check(ctx, s)} }(s)
	}
	for range servers {
		p := <-ch
		out[p.id] = p.res
	}
	return out
}
