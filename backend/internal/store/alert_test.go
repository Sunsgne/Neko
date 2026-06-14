package store

import (
	"context"
	"testing"
	"time"
)

func TestMemoryAlertFireDedupResolve(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	a := Alert{TenantID: "t1", DeviceID: "d1", Code: "device_offline", Severity: "critical", Title: "down"}

	_, created, err := s.Alerts().Fire(ctx, a)
	if err != nil || !created {
		t.Fatalf("first fire should create: created=%v err=%v", created, err)
	}
	// Firing again while open must NOT create a duplicate.
	_, created2, _ := s.Alerts().Fire(ctx, a)
	if created2 {
		t.Error("duplicate fire while firing should not create")
	}

	list, _ := s.Alerts().List(ctx, "t1", 10)
	if len(list) != 1 || list[0].State != "firing" {
		t.Fatalf("expected 1 firing alert, got %+v", list)
	}

	resolved, _ := s.Alerts().Resolve(ctx, "d1", "device_offline", time.Now())
	if !resolved {
		t.Error("resolve should return true")
	}
	// After resolve, a new fire creates again (new incident).
	_, created3, _ := s.Alerts().Fire(ctx, a)
	if !created3 {
		t.Error("fire after resolve should create a new alert")
	}

	// Tenant scoping.
	other, _ := s.Alerts().List(ctx, "t2", 10)
	if len(other) != 0 {
		t.Errorf("tenant t2 should see no alerts, got %d", len(other))
	}
}
