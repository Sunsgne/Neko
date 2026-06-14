package alerting

import (
	"context"
	"sync"
	"time"
)

// Notifier delivers alert events to an external channel (webhook/email/钉钉/
// 企业微信). Implementations should be non-blocking or fast.
type Notifier interface {
	Notify(ctx context.Context, ev Event) error
}

// NotifierFunc adapts a function to the Notifier interface.
type NotifierFunc func(ctx context.Context, ev Event) error

// Notify implements Notifier.
func (f NotifierFunc) Notify(ctx context.Context, ev Event) error { return f(ctx, ev) }

// Manager ties the Evaluator to notification, with de-duplication (handled by
// the evaluator), inhibition (suppress a series while a higher-severity alert
// is active for the same device), and escalation (re-notify if still firing
// after an interval).
type Manager struct {
	mu        sync.Mutex
	eval      *Evaluator
	notifiers []Notifier
	now       func() time.Time

	// firing tracks active alerts by series for inhibition/escalation.
	firing map[string]firingInfo
	// escalateAfter re-notifies if an alert is still firing after this long.
	escalateAfter time.Duration
}

type firingInfo struct {
	severity   Severity
	firedAt    time.Time
	lastNotify time.Time
	device     string
}

// NewManager builds a manager.
func NewManager(escalateAfter time.Duration, notifiers ...Notifier) *Manager {
	return &Manager{
		eval:          NewEvaluator(),
		notifiers:     notifiers,
		now:           time.Now,
		firing:        map[string]firingInfo{},
		escalateAfter: escalateAfter,
	}
}

// sevRank orders severities for inhibition.
func sevRank(s Severity) int {
	switch s {
	case SevCritical:
		return 3
	case SevWarning:
		return 2
	case SevInfo:
		return 1
	}
	return 0
}

// Observe evaluates one sample and notifies on transitions, applying
// inhibition. device groups series for inhibition (a critical alert on a
// device suppresses warnings/info on the same device).
func (m *Manager) Observe(ctx context.Context, rule Rule, device, series string, value float64) {
	now := m.now()
	ev, changed := m.eval.Evaluate(now, rule, series, value)
	if !changed {
		m.maybeEscalate(ctx)
		return
	}

	m.mu.Lock()
	if ev.State == StateFiring {
		m.firing[series] = firingInfo{severity: ev.Severity, firedAt: now, lastNotify: now, device: device}
	} else {
		delete(m.firing, series)
	}
	inhibited := ev.State == StateFiring && m.inhibitedLocked(device, ev.Severity)
	m.mu.Unlock()

	if inhibited {
		return // a higher-severity alert on this device is already active
	}
	m.dispatch(ctx, ev)
}

// inhibitedLocked reports whether a higher-severity alert is active on the
// same device. Caller holds the lock.
func (m *Manager) inhibitedLocked(device string, sev Severity) bool {
	for _, f := range m.firing {
		if f.device == device && sevRank(f.severity) > sevRank(sev) {
			return true
		}
	}
	return false
}

func (m *Manager) maybeEscalate(ctx context.Context) {
	if m.escalateAfter <= 0 {
		return
	}
	now := m.now()
	m.mu.Lock()
	var toEscalate []Event
	for series, f := range m.firing {
		if now.Sub(f.lastNotify) >= m.escalateAfter {
			f.lastNotify = now
			m.firing[series] = f
			toEscalate = append(toEscalate, Event{
				Series: series, State: StateFiring, Severity: f.severity, At: now,
			})
		}
	}
	m.mu.Unlock()
	for _, ev := range toEscalate {
		m.dispatch(ctx, ev)
	}
}

func (m *Manager) dispatch(ctx context.Context, ev Event) {
	for _, n := range m.notifiers {
		_ = n.Notify(ctx, ev)
	}
}
