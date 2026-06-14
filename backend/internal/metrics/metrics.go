// Package metrics provides a tiny, dependency-free metrics registry that
// exposes counters and gauges in Prometheus text exposition format. This is
// the metrics pillar of observability (T9.1); VictoriaMetrics scrapes the
// /metrics endpoint. Logs carry request IDs (structured slog) and OTLP tracing
// can be layered on later behind the same registry.
package metrics

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds named counters and gauges.
type Registry struct {
	mu       sync.Mutex
	counters map[string]float64
	gauges   map[string]float64
	help     map[string]string
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		counters: map[string]float64{},
		gauges:   map[string]float64{},
		help:     map[string]string{},
	}
}

func key(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := name + "{"
	for i, k := range keys {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%s=%q", k, labels[k])
	}
	return s + "}"
}

// IncCounter increments a counter by 1.
func (r *Registry) IncCounter(name string, labels map[string]string) {
	r.AddCounter(name, labels, 1)
}

// AddCounter adds delta to a counter.
func (r *Registry) AddCounter(name string, labels map[string]string, delta float64) {
	r.mu.Lock()
	r.counters[key(name, labels)] += delta
	r.mu.Unlock()
}

// SetGauge sets a gauge value.
func (r *Registry) SetGauge(name string, labels map[string]string, v float64) {
	r.mu.Lock()
	r.gauges[key(name, labels)] = v
	r.mu.Unlock()
}

// Expose renders all metrics in Prometheus text format (deterministic order).
func (r *Registry) Expose() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	lines := make([]string, 0, len(r.counters)+len(r.gauges))
	for k, v := range r.counters {
		lines = append(lines, fmt.Sprintf("%s %g", k, v))
	}
	for k, v := range r.gauges {
		lines = append(lines, fmt.Sprintf("%s %g", k, v))
	}
	sort.Strings(lines)
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}
