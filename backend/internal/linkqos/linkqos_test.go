package linkqos

import (
	"testing"
	"time"
)

func TestScoreBounds(t *testing.T) {
	cfg := DefaultScoreConfig()
	perfect := Score(Metrics{LatencyMs: 0, JitterMs: 0, Loss: 0}, cfg)
	if perfect < 99.9 {
		t.Errorf("perfect score = %.2f, want ~100", perfect)
	}
	awful := Score(Metrics{LatencyMs: 1000, JitterMs: 500, Loss: 1}, cfg)
	if awful != 0 {
		t.Errorf("awful score = %.2f, want 0", awful)
	}
	mid := Score(Metrics{LatencyMs: 150, JitterMs: 50, Loss: 0.05}, cfg)
	if mid <= 0 || mid >= 100 {
		t.Errorf("mid score = %.2f, want strictly between 0 and 100", mid)
	}
}

func TestInitSelectsPrimary(t *testing.T) {
	c := NewController(DefaultThresholds())
	now := time.Unix(0, 0)
	d := c.Observe(now, []LinkInput{
		{ID: "wan1", Role: RolePrimary, Score: 95},
		{ID: "wan2", Role: RoleBackup, Score: 90},
	})
	if d.Active != "wan1" {
		t.Errorf("init active = %q, want wan1", d.Active)
	}
}

func TestFailoverAfterMinDownAndDwell(t *testing.T) {
	thr := Thresholds{DownScore: 40, UpScore: 70, MinDown: 15 * time.Second, MinUp: 0, MinDwell: 10 * time.Second}
	c := NewController(thr)
	base := time.Unix(1000, 0)

	// Init on primary.
	c.Observe(base, []LinkInput{
		{ID: "wan1", Role: RolePrimary, Score: 95},
		{ID: "wan2", Role: RoleBackup, Score: 90},
	})

	// Primary goes bad; backup stays good. Advance past dwell + min-down.
	now := base.Add(20 * time.Second)
	in := []LinkInput{
		{ID: "wan1", Role: RolePrimary, Score: 10},
		{ID: "wan2", Role: RoleBackup, Score: 90},
	}
	c.Observe(now, in) // marks wan1 bad, wan2 good
	now = now.Add(16 * time.Second)
	d := c.Observe(now, in)
	if !d.Switched || d.Active != "wan2" {
		t.Fatalf("expected failover to wan2, got %+v", d)
	}
}

func TestDwellSuppressesFlapping(t *testing.T) {
	thr := Thresholds{DownScore: 40, UpScore: 70, MinDown: 1 * time.Second, MinUp: 0, MinDwell: 60 * time.Second}
	c := NewController(thr)
	base := time.Unix(0, 0)
	c.Observe(base, []LinkInput{{ID: "wan1", Role: RolePrimary, Score: 95}, {ID: "wan2", Role: RoleBackup, Score: 90}})

	// Immediately try to flip: within dwell window, must be suppressed.
	d := c.Observe(base.Add(5*time.Second), []LinkInput{
		{ID: "wan1", Role: RolePrimary, Score: 5},
		{ID: "wan2", Role: RoleBackup, Score: 95},
	})
	if d.Switched {
		t.Errorf("switch within dwell window should be suppressed, got %+v", d)
	}
}

func TestAutomaticFailback(t *testing.T) {
	thr := Thresholds{DownScore: 40, UpScore: 70, MinDown: 1 * time.Second, MinUp: 5 * time.Second, MinDwell: 1 * time.Second}
	c := NewController(thr)
	base := time.Unix(0, 0)

	// Start, fail over to backup.
	c.Observe(base, []LinkInput{{ID: "wan1", Role: RolePrimary, Score: 10}, {ID: "wan2", Role: RoleBackup, Score: 90}})
	now := base.Add(10 * time.Second)
	bad := []LinkInput{{ID: "wan1", Role: RolePrimary, Score: 10}, {ID: "wan2", Role: RoleBackup, Score: 90}}
	c.Observe(now, bad)
	now = now.Add(2 * time.Second)
	d := c.Observe(now, bad)
	if d.Active != "wan2" {
		t.Fatalf("expected to be on wan2, got %+v", d)
	}

	// Primary recovers; after MinUp it should fail back automatically.
	good := []LinkInput{{ID: "wan1", Role: RolePrimary, Score: 95}, {ID: "wan2", Role: RoleBackup, Score: 90}}
	now = now.Add(2 * time.Second)
	c.Observe(now, good) // marks wan1 good
	now = now.Add(6 * time.Second)
	d = c.Observe(now, good)
	if !d.Switched || d.Active != "wan1" || d.Reason != "failback" {
		t.Fatalf("expected failback to wan1, got %+v", d)
	}
}
