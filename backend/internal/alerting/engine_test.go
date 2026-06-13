package alerting

import (
	"testing"
	"time"
)

func TestFiresAfterForDuration(t *testing.T) {
	e := NewEvaluator()
	rule := Rule{ID: "cpu", Metric: "cpu", Op: GT, Threshold: 80, For: 30 * time.Second, Severity: SevWarning}
	base := time.Unix(0, 0)

	// First breach: pending, not firing yet.
	if _, fired := e.Evaluate(base, rule, "dev1", 95); fired {
		t.Fatal("should not fire immediately")
	}
	// Still within For window.
	if _, fired := e.Evaluate(base.Add(20*time.Second), rule, "dev1", 95); fired {
		t.Fatal("should not fire before For elapses")
	}
	// After For window.
	ev, fired := e.Evaluate(base.Add(31*time.Second), rule, "dev1", 95)
	if !fired || ev.State != StateFiring {
		t.Fatalf("expected firing, got %+v fired=%v", ev, fired)
	}
}

func TestDeduplicatesWhileFiring(t *testing.T) {
	e := NewEvaluator()
	rule := Rule{ID: "cpu", Op: GT, Threshold: 80, For: 0, Severity: SevWarning}
	base := time.Unix(0, 0)

	if _, fired := e.Evaluate(base, rule, "dev1", 95); !fired {
		t.Fatal("expected initial fire")
	}
	// Repeated breaches should NOT re-fire.
	if _, fired := e.Evaluate(base.Add(time.Second), rule, "dev1", 96); fired {
		t.Fatal("should not re-fire while already firing")
	}
}

func TestResolves(t *testing.T) {
	e := NewEvaluator()
	rule := Rule{ID: "cpu", Op: GT, Threshold: 80, For: 0, Severity: SevWarning}
	base := time.Unix(0, 0)

	e.Evaluate(base, rule, "dev1", 95) // fire
	ev, changed := e.Evaluate(base.Add(time.Second), rule, "dev1", 10)
	if !changed || ev.State != StateOK {
		t.Fatalf("expected resolve, got %+v changed=%v", ev, changed)
	}
}

func TestPendingResetOnClear(t *testing.T) {
	e := NewEvaluator()
	rule := Rule{ID: "cpu", Op: GT, Threshold: 80, For: 30 * time.Second, Severity: SevWarning}
	base := time.Unix(0, 0)

	e.Evaluate(base, rule, "dev1", 95)                     // pending since +0s
	e.Evaluate(base.Add(10*time.Second), rule, "dev1", 10) // clears pending
	// New breach restarts the For timer at +15s; firing requires +45s.
	e.Evaluate(base.Add(15*time.Second), rule, "dev1", 95)
	if _, fired := e.Evaluate(base.Add(40*time.Second), rule, "dev1", 95); fired {
		t.Fatal("must NOT fire at +40s (only 25s since restart)")
	}
	if _, fired := e.Evaluate(base.Add(46*time.Second), rule, "dev1", 95); !fired {
		t.Fatal("must fire at +46s (31s since restart)")
	}
}

func TestSeriesIsolation(t *testing.T) {
	e := NewEvaluator()
	rule := Rule{ID: "cpu", Op: GT, Threshold: 80, For: 0, Severity: SevWarning}
	base := time.Unix(0, 0)
	if _, fired := e.Evaluate(base, rule, "dev1", 95); !fired {
		t.Fatal("dev1 should fire")
	}
	// dev2 is independent and healthy.
	if _, fired := e.Evaluate(base, rule, "dev2", 10); fired {
		t.Fatal("dev2 should not fire")
	}
}
