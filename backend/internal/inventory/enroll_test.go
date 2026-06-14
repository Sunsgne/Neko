package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/secret"
	"github.com/neko/sdwan/backend/internal/store"
)

type fakeProbe struct {
	status store.DeviceStatus
	err    error
}

func (f fakeProbe) Status(context.Context, routeros.Target) (store.DeviceStatus, error) {
	return f.status, f.err
}

func TestEnrollStoresCredsAndManages(t *testing.T) {
	st := store.NewMemory()
	sealer, _ := secret.New("k")
	facts := &routeros.DeviceFacts{
		Resource:    routeros.SystemResource{BoardName: "CHR", Architecture: "x86_64", Version: "7.14"},
		Routerboard: routeros.RouterboardInfo{Routerboard: false},
	}
	svc := NewService(Deps{
		Devices:     st.Devices(),
		Credentials: st.Credentials(),
		Collector:   routeros.StaticCollector{Facts: facts},
		Probe:       fakeProbe{status: store.DeviceStatus{Online: true, Version: "7.14", CPULoadPercent: 5}},
		Sealer:      sealer,
		ID:          func() string { return "dev_1" },
		Now:         func() time.Time { return time.Unix(0, 0).UTC() },
	})

	d, err := svc.Register(context.Background(), "", RegisterInput{Name: "chr-1", MgmtAddress: "10.0.0.1", Role: store.RoleBackbone})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Enroll(context.Background(), "", d.ID, "admin", "secret")
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if !got.Enrolled || got.TrustState != store.TrustManaged {
		t.Errorf("device should be enrolled+managed, got enrolled=%v trust=%s", got.Enrolled, got.TrustState)
	}
	if got.Platform != store.PlatformCHR {
		t.Errorf("platform = %s", got.Platform)
	}
	if got.Status == nil || !got.Status.Online {
		t.Error("status should be populated and online after enroll poll")
	}

	// Credentials stored encrypted (not plaintext).
	cred, err := st.Credentials().Get(context.Background(), d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Sealed == "" || contains(cred.Sealed, "secret") {
		t.Error("credential must be stored encrypted, not plaintext")
	}
}

func TestPollRequiresEnrollment(t *testing.T) {
	st := store.NewMemory()
	sealer, _ := secret.New("k")
	svc := NewService(Deps{
		Devices: st.Devices(), Credentials: st.Credentials(), Probe: fakeProbe{}, Sealer: sealer,
		ID: func() string { return "dev_x" }, Now: func() time.Time { return time.Unix(0, 0).UTC() },
	})
	d, _ := svc.Register(context.Background(), "", RegisterInput{Name: "x", MgmtAddress: "10.0.0.1"})
	if _, err := svc.Poll(context.Background(), "", d.ID); err != ErrNotEnrolled {
		t.Errorf("expected ErrNotEnrolled, got %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
