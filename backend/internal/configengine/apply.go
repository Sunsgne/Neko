package configengine

import (
	"context"
	"errors"
	"fmt"
)

// Applier executes a plan against a device. Implementations wrap the RouterOS
// REST/SSH client. The executor below adds commit-confirm + verification +
// rollback semantics on top of any Applier.
type Applier interface {
	// Snapshot captures the current running config so it can be restored.
	Snapshot(ctx context.Context) (State, error)
	// Apply executes the changes. When confirmTimeout > 0 the device must be
	// told to auto-revert if Confirm is not called within the window
	// (RouterOS safe-mode / commit-confirm). Implementations should arm that
	// timer before returning.
	Apply(ctx context.Context, plan Plan, confirmTimeoutSec int) error
	// Confirm cancels the pending auto-revert, making the change permanent.
	Confirm(ctx context.Context) error
	// Restore rolls the device back to a previously captured snapshot.
	Restore(ctx context.Context, snapshot State) error
}

// Verifier checks device health after an apply (connectivity, routing,
// interfaces). Returning an error triggers rollback.
type Verifier interface {
	Verify(ctx context.Context) error
}

// ApplyOptions controls executor behavior.
type ApplyOptions struct {
	// ConfirmTimeoutSec arms a device-side auto-revert; 0 disables it.
	ConfirmTimeoutSec int
	// MaxRisk blocks applies whose aggregate risk exceeds this level.
	MaxRisk Risk
	Risk    RiskOptions
}

// ApplyResult describes the outcome of an apply.
type ApplyResult struct {
	Status     string `json:"status"` // committed | rolledback | skipped | blocked
	RolledBack bool   `json:"rolled_back"`
	Reason     string `json:"reason,omitempty"`
}

// ErrRiskBlocked indicates the plan exceeded the allowed risk threshold.
var ErrRiskBlocked = errors.New("plan blocked by risk policy")

// Execute applies desired state onto a device with the full safety pipeline:
//
//	snapshot -> diff -> risk gate -> apply(commit-confirm) -> verify ->
//	   confirm (success)  |  restore+abort (failure)
//
// The management channel is protected by the device-side auto-revert: if
// verification cannot reach the device, the device reverts on its own.
func Execute(ctx context.Context, applier Applier, verifier Verifier, desired State, opts ApplyOptions) (ApplyResult, Plan, error) {
	running, err := applier.Snapshot(ctx)
	if err != nil {
		return ApplyResult{Status: "blocked", Reason: "snapshot failed"}, Plan{}, err
	}

	plan := ComputeDiff(running, desired, opts.Risk)
	if plan.Empty() {
		return ApplyResult{Status: "skipped", Reason: "no changes"}, plan, nil
	}

	if opts.MaxRisk != "" && riskRank(plan.AggregateRisk) > riskRank(opts.MaxRisk) {
		return ApplyResult{Status: "blocked", Reason: fmt.Sprintf("risk %s exceeds max %s", plan.AggregateRisk, opts.MaxRisk)},
			plan, ErrRiskBlocked
	}

	if err := applier.Apply(ctx, plan, opts.ConfirmTimeoutSec); err != nil {
		// Apply failed mid-flight; device auto-revert (if armed) protects us.
		return ApplyResult{Status: "rolledback", RolledBack: true, Reason: "apply error: " + err.Error()}, plan, err
	}

	if verifier != nil {
		if vErr := verifier.Verify(ctx); vErr != nil {
			// Verification failed: restore the snapshot explicitly.
			if rErr := applier.Restore(ctx, running); rErr != nil {
				return ApplyResult{Status: "rolledback", RolledBack: true, Reason: "verify failed, restore error: " + rErr.Error()}, plan, vErr
			}
			return ApplyResult{Status: "rolledback", RolledBack: true, Reason: "verify failed: " + vErr.Error()}, plan, nil
		}
	}

	if err := applier.Confirm(ctx); err != nil {
		return ApplyResult{Status: "rolledback", RolledBack: true, Reason: "confirm failed: " + err.Error()}, plan, err
	}
	return ApplyResult{Status: "committed"}, plan, nil
}
