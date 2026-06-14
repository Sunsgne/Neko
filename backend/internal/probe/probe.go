// Package probe implements active link-quality probing (ICMP/TCP/HTTP/HTTPS/
// DNS) producing latency/loss/jitter samples that feed link scoring (Epic 8).
package probe

import (
	"context"
	"crypto/tls"
	"math"
	"net"
	"net/http"
	"time"
)

// Kind selects the probe type.
type Kind string

const (
	KindICMP  Kind = "icmp"
	KindTCP   Kind = "tcp"
	KindHTTP  Kind = "http"
	KindHTTPS Kind = "https"
	KindDNS   Kind = "dns"
)

// Spec describes a probe target.
type Spec struct {
	Kind    Kind
	Target  string // host, host:port, or URL depending on Kind
	Count   int    // samples to collect (for loss/jitter); default 5
	Timeout time.Duration
}

// Result aggregates a probe run.
type Result struct {
	Kind      Kind    `json:"kind"`
	Target    string  `json:"target"`
	Sent      int     `json:"sent"`
	Received  int     `json:"received"`
	Loss      float64 `json:"loss"`       // 0..1
	LatencyMs float64 `json:"latency_ms"` // mean of successes
	JitterMs  float64 `json:"jitter_ms"`  // mean abs successive diff
}

// Prober runs a single attempt for a kind, returning the RTT or an error.
type attemptFunc func(ctx context.Context, target string, timeout time.Duration) (time.Duration, error)

// Run executes Count attempts and aggregates latency/loss/jitter.
func Run(ctx context.Context, spec Spec) Result {
	count := spec.Count
	if count <= 0 {
		count = 5
	}
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	attempt := attemptFor(spec.Kind)

	res := Result{Kind: spec.Kind, Target: spec.Target, Sent: count}
	var rtts []float64
	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			break
		}
		d, err := attempt(ctx, spec.Target, timeout)
		if err != nil {
			continue
		}
		res.Received++
		rtts = append(rtts, float64(d.Microseconds())/1000.0)
	}
	res.Loss = float64(res.Sent-res.Received) / float64(res.Sent)
	res.LatencyMs = mean(rtts)
	res.JitterMs = jitter(rtts)
	return res
}

func attemptFor(k Kind) attemptFunc {
	switch k {
	case KindICMP:
		return attemptICMP
	case KindHTTP, KindHTTPS:
		return attemptHTTP
	case KindDNS:
		return attemptDNS
	default:
		return attemptTCP
	}
}

func attemptTCP(ctx context.Context, target string, timeout time.Duration) (time.Duration, error) {
	if _, _, err := net.SplitHostPort(target); err != nil {
		target = net.JoinHostPort(target, "80")
	}
	d := net.Dialer{Timeout: timeout}
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		return 0, err
	}
	_ = conn.Close()
	return time.Since(start), nil
}

func attemptHTTP(ctx context.Context, target string, timeout time.Duration) (time.Duration, error) {
	url := target
	if len(url) < 4 || (url[:4] != "http") {
		url = "https://" + target
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // probe only
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return time.Since(start), nil
}

func attemptDNS(ctx context.Context, target string, timeout time.Duration) (time.Duration, error) {
	r := &net.Resolver{}
	start := time.Now()
	_, err := r.LookupHost(ctx, target)
	if err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// jitter is the mean absolute difference between successive RTTs.
func jitter(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var s float64
	for i := 1; i < len(xs); i++ {
		s += math.Abs(xs[i] - xs[i-1])
	}
	return s / float64(len(xs)-1)
}
