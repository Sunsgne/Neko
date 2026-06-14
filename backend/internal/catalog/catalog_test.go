package catalog

import (
	"context"
	"testing"
)

func TestLinksScopedAndSorted(t *testing.T) {
	c := New()
	c.ReplaceLinks([]Link{
		{ID: "l1", TenantID: "a", Score: 60},
		{ID: "l2", TenantID: "a", Score: 95},
		{ID: "l3", TenantID: "b", Score: 80},
	})
	got := c.Links(context.Background(), "a")
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "l2" {
		t.Errorf("expected highest score first, got %s", got[0].ID)
	}
}

func TestAlertsFiringFirst(t *testing.T) {
	c := New()
	c.ReplaceAlerts([]Alert{
		{ID: "a1", TenantID: "a", State: "resolved", FiredAt: "2026-01-01"},
		{ID: "a2", TenantID: "a", State: "firing", FiredAt: "2026-01-02"},
	})
	got := c.Alerts(context.Background(), "a")
	if got[0].ID != "a2" {
		t.Errorf("expected firing first, got %s", got[0].ID)
	}
}

func TestDNSSharedVisibleToTenant(t *testing.T) {
	c := New()
	c.ReplaceDNS([]DNSServer{
		{ID: "shared", TenantID: "", LatencyMs: 5},
		{ID: "ten-a", TenantID: "a", LatencyMs: 8},
		{ID: "ten-b", TenantID: "b", LatencyMs: 3},
	})
	got := c.DNSServers(context.Background(), "a")
	if len(got) != 2 { // shared + ten-a
		t.Fatalf("len = %d, want 2 (%+v)", len(got), got)
	}
	if got[0].ID != "shared" {
		t.Errorf("expected lowest latency first, got %s", got[0].ID)
	}
}
