package probe

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/neko/sdwan/backend/internal/linkqos"
)

// Sample is a scored link-quality datapoint ready for reporting.
type Sample struct {
	TenantID string
	LinkID   string
	Result   Result
	Score    float64
}

// ScoreResult converts a probe Result into a link score using the given config.
func ScoreResult(r Result, cfg linkqos.ScoreConfig) float64 {
	return linkqos.Score(linkqos.Metrics{
		LatencyMs: r.LatencyMs,
		JitterMs:  r.JitterMs,
		Loss:      r.Loss,
	}, cfg)
}

// Reporter ships samples to a metrics backend.
type Reporter interface {
	Report(ctx context.Context, samples []Sample) error
}

// VictoriaMetricsReporter writes samples to VictoriaMetrics via the Influx
// line-protocol ingestion endpoint (/write), which VM accepts natively.
type VictoriaMetricsReporter struct {
	BaseURL string // e.g. http://localhost:8428
	client  *http.Client
}

// NewVMReporter builds a VictoriaMetrics reporter.
func NewVMReporter(baseURL string) *VictoriaMetricsReporter {
	return &VictoriaMetricsReporter{BaseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 5 * time.Second}}
}

// Report sends samples as line-protocol metrics.
func (r *VictoriaMetricsReporter) Report(ctx context.Context, samples []Sample) error {
	if len(samples) == 0 {
		return nil
	}
	body := LineProtocol(samples, time.Now())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+"/write", bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("vm write failed: %d", resp.StatusCode)
	}
	return nil
}

// LineProtocol renders samples in Influx line protocol. Exposed for testing.
func LineProtocol(samples []Sample, ts time.Time) string {
	var sb strings.Builder
	tsNano := ts.UnixNano()
	// Deterministic ordering for stable output/tests.
	sorted := append([]Sample(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].LinkID < sorted[j].LinkID })
	for _, s := range sorted {
		tags := fmt.Sprintf("tenant=%s,link=%s,kind=%s", escape(s.TenantID), escape(s.LinkID), s.Result.Kind)
		fmt.Fprintf(&sb, "neko_link,%s latency_ms=%g,jitter_ms=%g,loss=%g,score=%g %d\n",
			tags, s.Result.LatencyMs, s.Result.JitterMs, s.Result.Loss, s.Score, tsNano)
	}
	return sb.String()
}

func escape(s string) string {
	if s == "" {
		return "none"
	}
	r := strings.NewReplacer(" ", "\\ ", ",", "\\,", "=", "\\=")
	return r.Replace(s)
}
