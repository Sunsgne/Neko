package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neko/sdwan/backend/internal/store"
)

func newTestService() *Service {
	st := store.NewMemory()
	n := 0
	id := func() string { n++; return "dev_" + string(rune('a'+n)) }
	now := func() time.Time { return time.Unix(0, 0).UTC() }
	return NewService(st.Devices(), nil, id, now)
}

func TestRegisterDevice(t *testing.T) {
	svc := newTestService()
	d, err := svc.Register(context.Background(), "ten_1", RegisterInput{Name: "edge-01", MgmtAddress: "10.0.0.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.TrustState != store.TrustDiscovered {
		t.Errorf("trust = %q, want discovered", d.TrustState)
	}
	if d.Platform != store.PlatformUnknown {
		t.Errorf("platform = %q, want unknown", d.Platform)
	}
	if d.TenantID != "ten_1" {
		t.Errorf("tenant = %q, want ten_1", d.TenantID)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc := newTestService()
	cases := []RegisterInput{
		{Name: "", MgmtAddress: "10.0.0.1"},
		{Name: "x", MgmtAddress: ""},
		{Name: "x", MgmtAddress: "not a host!"},
	}
	for i, c := range cases {
		if _, err := svc.Register(context.Background(), "ten_1", c); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("case %d: expected ErrInvalidInput, got %v", i, err)
		}
	}
}

func TestTenantScoping(t *testing.T) {
	svc := newTestService()
	d, err := svc.Register(context.Background(), "ten_1", RegisterInput{Name: "edge", MgmtAddress: "10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	// Another tenant must not see it.
	if _, err := svc.Get(context.Background(), "ten_2", d.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-tenant access, got %v", err)
	}
	// Owning tenant can.
	if _, err := svc.Get(context.Background(), "ten_1", d.ID); err != nil {
		t.Errorf("owning tenant get failed: %v", err)
	}
}
