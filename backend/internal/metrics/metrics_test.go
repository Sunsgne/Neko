package metrics

import (
	"strings"
	"testing"
)

func TestCountersAndGauges(t *testing.T) {
	r := NewRegistry()
	r.IncCounter("neko_http_requests_total", map[string]string{"path": "/healthz"})
	r.IncCounter("neko_http_requests_total", map[string]string{"path": "/healthz"})
	r.SetGauge("neko_devices", nil, 5)

	out := r.Expose()
	if !strings.Contains(out, `neko_http_requests_total{path="/healthz"} 2`) {
		t.Errorf("counter not aggregated: %s", out)
	}
	if !strings.Contains(out, "neko_devices 5") {
		t.Errorf("gauge missing: %s", out)
	}
}

func TestExposeDeterministic(t *testing.T) {
	r := NewRegistry()
	r.SetGauge("b", nil, 1)
	r.SetGauge("a", nil, 2)
	out := r.Expose()
	if strings.Index(out, "a ") > strings.Index(out, "b ") {
		t.Errorf("expected sorted output: %s", out)
	}
}
