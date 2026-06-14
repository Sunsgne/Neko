package alerting

import (
	"context"
	"sync"
	"testing"
	"time"
)

type capture struct {
	mu sync.Mutex
	ev []Event
}

func (c *capture) Notify(_ context.Context, ev Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ev = append(c.ev, ev)
	return nil
}

func (c *capture) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.ev)
}

func TestManagerNotifiesOnFire(t *testing.T) {
	cap := &capture{}
	m := NewManager(0, cap)
	rule := Rule{ID: "cpu", Op: GT, Threshold: 80, For: 0, Severity: SevWarning}
	m.Observe(context.Background(), rule, "dev1", "dev1.cpu", 95)
	if cap.count() != 1 {
		t.Fatalf("expected 1 notification, got %d", cap.count())
	}
}

func TestManagerInhibitsLowerSeverity(t *testing.T) {
	cap := &capture{}
	m := NewManager(0, cap)
	crit := Rule{ID: "link-down", Op: LT, Threshold: 40, For: 0, Severity: SevCritical}
	warn := Rule{ID: "cpu", Op: GT, Threshold: 80, For: 0, Severity: SevWarning}

	// Critical fires first for dev1.
	m.Observe(context.Background(), crit, "dev1", "dev1.link", 10)
	// Warning on the same device should be inhibited.
	m.Observe(context.Background(), warn, "dev1", "dev1.cpu", 95)

	if cap.count() != 1 {
		t.Fatalf("warning should be inhibited; got %d notifications", cap.count())
	}
}

func TestManagerEscalates(t *testing.T) {
	cap := &capture{}
	m := NewManager(30*time.Second, cap)
	base := time.Unix(0, 0)
	m.now = func() time.Time { return base }
	rule := Rule{ID: "cpu", Op: GT, Threshold: 80, For: 0, Severity: SevWarning}
	m.Observe(context.Background(), rule, "dev1", "dev1.cpu", 95) // fire (1)

	// Still firing after escalateAfter → re-notify.
	m.now = func() time.Time { return base.Add(31 * time.Second) }
	m.Observe(context.Background(), rule, "dev1", "dev1.cpu", 95)
	if cap.count() != 2 {
		t.Fatalf("expected escalation re-notify (2), got %d", cap.count())
	}
}
