package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/store"
)

func TestDetectEnrichesDevice(t *testing.T) {
	st := store.NewMemory()
	n := 0
	id := func() string { n++; return "dev_" + string(rune('a'+n)) }
	now := func() time.Time { return time.Unix(0, 0).UTC() }

	facts := &routeros.DeviceFacts{
		Resource:    routeros.SystemResource{BoardName: "RB5009UG+S+IN", Architecture: "arm64", Version: "7.14.3"},
		Routerboard: routeros.RouterboardInfo{Routerboard: true, Model: "RB5009UG+S+IN", SerialNumber: "SN1"},
		Interfaces:  []routeros.Interface{{Name: "ether1", Type: "ether"}},
	}
	svc := NewService(Deps{Devices: st.Devices(), Collector: routeros.StaticCollector{Facts: facts}, ID: id, Now: now})

	d, err := svc.Register(context.Background(), "ten_1", RegisterInput{Name: "edge", MgmtAddress: "10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	if d.TrustState != store.TrustDiscovered {
		t.Fatalf("initial trust = %q", d.TrustState)
	}

	got, err := svc.Detect(context.Background(), "ten_1", d.ID)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if got.Platform != store.PlatformRouterBOARD {
		t.Errorf("platform = %q", got.Platform)
	}
	if got.Model != "RB5009UG+S+IN" || got.Serial != "SN1" {
		t.Errorf("model/serial = %q/%q", got.Model, got.Serial)
	}
	if got.TrustState != store.TrustAuthenticated {
		t.Errorf("trust = %q, want authenticated", got.TrustState)
	}
	if got.Capabilities == nil || !got.Capabilities.SupportsBGP {
		t.Error("capabilities not populated")
	}
	if got.LastSeenAt == nil {
		t.Error("last_seen_at not set")
	}
}

func TestSetTrustStateEnforcesMachine(t *testing.T) {
	st := store.NewMemory()
	id := func() string { return "dev_x" }
	now := func() time.Time { return time.Unix(0, 0).UTC() }
	svc := NewService(Deps{Devices: st.Devices(), ID: id, Now: now})

	d, _ := svc.Register(context.Background(), "ten_1", RegisterInput{Name: "edge", MgmtAddress: "10.0.0.1"})

	// discovered -> managed is an illegal skip.
	if _, err := svc.SetTrustState(context.Background(), "ten_1", d.ID, store.TrustManaged); err == nil {
		t.Error("expected error skipping stages")
	}
	// discovered -> authenticated is allowed.
	if _, err := svc.SetTrustState(context.Background(), "ten_1", d.ID, store.TrustAuthenticated); err != nil {
		t.Errorf("unexpected: %v", err)
	}
}
