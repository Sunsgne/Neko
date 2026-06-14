package configengine

import (
	"context"
	"errors"
	"testing"
)

type fakeApplier struct {
	running    State
	applied    bool
	confirmed  bool
	restored   bool
	applyErr   error
	confirmErr error
}

func (f *fakeApplier) Snapshot(context.Context) (State, error) { return f.running, nil }
func (f *fakeApplier) Apply(_ context.Context, _ Plan, _ int) error {
	if f.applyErr != nil {
		return f.applyErr
	}
	f.applied = true
	return nil
}
func (f *fakeApplier) Confirm(context.Context) error        { f.confirmed = true; return f.confirmErr }
func (f *fakeApplier) Restore(context.Context, State) error { f.restored = true; return nil }

type fakeVerifier struct{ err error }

func (v fakeVerifier) Verify(context.Context) error { return v.err }

func desiredWith(path, key string) State {
	return State{Statements: []Statement{{Path: path, Key: key, Attributes: map[string]string{"x": "1"}}}}
}

func TestExecuteCommitsOnSuccess(t *testing.T) {
	a := &fakeApplier{}
	res, _, err := Execute(context.Background(), a, fakeVerifier{}, desiredWith("/ip/dns", "s"), ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "committed" || !a.confirmed || a.restored {
		t.Fatalf("expected commit, got %+v applier=%+v", res, a)
	}
}

func TestExecuteRollsBackOnVerifyFailure(t *testing.T) {
	a := &fakeApplier{}
	res, _, _ := Execute(context.Background(), a, fakeVerifier{err: errors.New("unreachable")}, desiredWith("/ip/dns", "s"), ApplyOptions{})
	if !res.RolledBack || !a.restored || a.confirmed {
		t.Fatalf("expected rollback, got %+v applier=%+v", res, a)
	}
}

func TestExecuteSkipsWhenNoChange(t *testing.T) {
	s := desiredWith("/ip/dns", "s")
	a := &fakeApplier{running: s}
	res, _, _ := Execute(context.Background(), a, fakeVerifier{}, s, ApplyOptions{})
	if res.Status != "skipped" {
		t.Fatalf("expected skipped, got %+v", res)
	}
}

func TestExecuteBlocksHighRisk(t *testing.T) {
	a := &fakeApplier{}
	_, _, err := Execute(context.Background(), a, nil, desiredWith("/ip/firewall/filter", "drop"), ApplyOptions{MaxRisk: RiskMedium})
	if !errors.Is(err, ErrRiskBlocked) {
		t.Fatalf("expected risk block, got %v", err)
	}
	if a.applied {
		t.Error("must not apply when blocked")
	}
}

func TestPlanCanaryBatches(t *testing.T) {
	devices := make([]string, 100)
	for i := range devices {
		devices[i] = string(rune('a')) + string(rune(i))
	}
	batches := PlanCanaryBatches(devices, DefaultCanaryStages)
	if len(batches) == 0 {
		t.Fatal("expected batches")
	}
	if len(batches[0]) != 1 {
		t.Errorf("first batch should be 1 (canary), got %d", len(batches[0]))
	}
	// All devices covered exactly once.
	total := 0
	for _, b := range batches {
		total += len(b)
	}
	if total != 100 {
		t.Errorf("coverage = %d, want 100", total)
	}
}

func TestPlanCanaryBatchesSmallFleet(t *testing.T) {
	batches := PlanCanaryBatches([]string{"a", "b"}, DefaultCanaryStages)
	total := 0
	for _, b := range batches {
		total += len(b)
	}
	if total != 2 {
		t.Errorf("coverage = %d, want 2", total)
	}
}
