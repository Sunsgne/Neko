package probe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/neko/sdwan/backend/internal/linkqos"
)

func TestTCPProbeSuccess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	res := Run(context.Background(), Spec{Kind: KindTCP, Target: ln.Addr().String(), Count: 3, Timeout: time.Second})
	if res.Received != 3 {
		t.Errorf("received = %d, want 3", res.Received)
	}
	if res.Loss != 0 {
		t.Errorf("loss = %g, want 0", res.Loss)
	}
}

func TestTCPProbeLoss(t *testing.T) {
	// Closed port → all attempts fail → 100% loss.
	res := Run(context.Background(), Spec{Kind: KindTCP, Target: "127.0.0.1:1", Count: 2, Timeout: 300 * time.Millisecond})
	if res.Loss != 1 {
		t.Errorf("loss = %g, want 1", res.Loss)
	}
}

func TestHTTPProbe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	res := Run(context.Background(), Spec{Kind: KindHTTP, Target: srv.URL, Count: 2, Timeout: time.Second})
	if res.Received != 2 {
		t.Errorf("received = %d, want 2", res.Received)
	}
}

func TestScoreAndLineProtocol(t *testing.T) {
	cfg := linkqos.DefaultScoreConfig()
	r := Result{Kind: KindTCP, LatencyMs: 10, JitterMs: 2, Loss: 0}
	score := ScoreResult(r, cfg)
	if score < 90 {
		t.Errorf("good link should score high, got %g", score)
	}
	lp := LineProtocol([]Sample{{TenantID: "t1", LinkID: "l1", Result: r, Score: score}}, time.Unix(0, 1))
	if !strings.Contains(lp, "neko_link,tenant=t1,link=l1,kind=tcp") {
		t.Errorf("unexpected line protocol: %s", lp)
	}
	if !strings.Contains(lp, "score=") {
		t.Error("missing score field")
	}
}

func TestVMReporter(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		r.Body.Read(b)
		got = string(b)
	}))
	defer srv.Close()
	rep := NewVMReporter(srv.URL)
	err := rep.Report(context.Background(), []Sample{{TenantID: "t", LinkID: "l", Result: Result{Kind: KindICMP, LatencyMs: 5}, Score: 99}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "neko_link") {
		t.Errorf("VM did not receive metric, got %q", got)
	}
}
