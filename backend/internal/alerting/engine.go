// Package alerting evaluates threshold rules over metric streams with
// for-duration support and de-duplication, producing firing/resolved
// transitions. It is the core of the SNMP/monitoring alert engine (Epic 5).
package alerting

import "time"

// Comparison is a threshold operator.
type Comparison string

const (
	GT  Comparison = ">"
	GTE Comparison = ">="
	LT  Comparison = "<"
	LTE Comparison = "<="
)

// Severity classifies an alert's importance.
type Severity string

const (
	SevInfo     Severity = "info"
	SevWarning  Severity = "warning"
	SevCritical Severity = "critical"
)

// Rule defines a threshold condition that must hold for For duration before
// firing. Metric is informational; the caller routes the right value in.
type Rule struct {
	ID        string
	Metric    string
	Op        Comparison
	Threshold float64
	For       time.Duration
	Severity  Severity
}

func (r Rule) breached(v float64) bool {
	switch r.Op {
	case GT:
		return v > r.Threshold
	case GTE:
		return v >= r.Threshold
	case LT:
		return v < r.Threshold
	case LTE:
		return v <= r.Threshold
	}
	return false
}

// AlertState is the current state of an alert series.
type AlertState string

const (
	StateOK     AlertState = "ok"
	StateFiring AlertState = "firing"
)

// Event is emitted when an alert transitions between states.
type Event struct {
	RuleID   string
	Series   string
	State    AlertState
	Severity Severity
	Value    float64
	At       time.Time
}

type seriesState struct {
	state        AlertState
	pendingSince time.Time
	pending      bool
}

// Evaluator holds per-(rule,series) state across evaluations. Not safe for
// concurrent use; serialize calls per evaluator.
type Evaluator struct {
	states map[string]*seriesState
}

// NewEvaluator builds an empty evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{states: map[string]*seriesState{}}
}

// Evaluate feeds one observation for (rule, series). It returns an Event and
// true only when the alert transitions (OK->firing or firing->OK); otherwise
// it returns false (de-duplication: no repeated events while steady).
func (e *Evaluator) Evaluate(now time.Time, rule Rule, series string, value float64) (Event, bool) {
	key := rule.ID + "\x00" + series
	st, ok := e.states[key]
	if !ok {
		st = &seriesState{state: StateOK}
		e.states[key] = st
	}

	if rule.breached(value) {
		if !st.pending {
			st.pending = true
			st.pendingSince = now
		}
		if st.state != StateFiring && now.Sub(st.pendingSince) >= rule.For {
			st.state = StateFiring
			return Event{RuleID: rule.ID, Series: series, State: StateFiring, Severity: rule.Severity, Value: value, At: now}, true
		}
		return Event{}, false
	}

	// Condition cleared.
	st.pending = false
	if st.state == StateFiring {
		st.state = StateOK
		return Event{RuleID: rule.ID, Series: series, State: StateOK, Severity: rule.Severity, Value: value, At: now}, true
	}
	return Event{}, false
}
