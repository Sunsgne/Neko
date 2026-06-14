// Package configengine implements the Desired State configuration engine:
// it models device configuration declaratively, computes a minimal diff
// between running and desired state, and classifies the risk of applying it.
//
// Per requirement #5, the platform never pushes fixed per-model scripts; it
// converges devices toward a declared desired state with diff + risk grading +
// safe rollback (commit-confirm) + canary rollout.
package configengine

import "sort"

// Statement is a single declarative configuration resource, identified by its
// RouterOS Path (e.g. "/ip/address") plus a natural Key within that path
// (e.g. an interface name or address). Attributes hold the desired values.
type Statement struct {
	Path       string            `json:"path"`
	Key        string            `json:"key"`
	Attributes map[string]string `json:"attributes"`
}

// State is a collection of statements representing a full or partial config.
type State struct {
	Statements []Statement `json:"statements"`
}

// Merge combines multiple states into one. Later statements override earlier
// ones with the same (path,key), so callers can layer policies (e.g. link
// failover + acceleration) deterministically.
func Merge(states ...State) State {
	idx := map[string]Statement{}
	var order []string
	for _, st := range states {
		for _, s := range st.Statements {
			k := s.Path + "\x00" + s.Key
			if _, seen := idx[k]; !seen {
				order = append(order, k)
			}
			idx[k] = s
		}
	}
	out := State{Statements: make([]Statement, 0, len(order))}
	for _, k := range order {
		out.Statements = append(out.Statements, idx[k])
	}
	return out
}

// index maps "path\x00key" -> statement for O(1) lookups.
func (s State) index() map[string]Statement {
	m := make(map[string]Statement, len(s.Statements))
	for _, st := range s.Statements {
		m[st.Path+"\x00"+st.Key] = st
	}
	return m
}

// ChangeType enumerates the kinds of changes in a plan.
type ChangeType string

const (
	ChangeAdd    ChangeType = "add"
	ChangeUpdate ChangeType = "update"
	ChangeRemove ChangeType = "remove"
)

// AttrChange records a single attribute's before/after values.
type AttrChange struct {
	Attr string `json:"attr"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

// Change is one unit of work in a plan.
type Change struct {
	Type   ChangeType   `json:"type"`
	Path   string       `json:"path"`
	Key    string       `json:"key"`
	Attrs  []AttrChange `json:"attrs,omitempty"`
	Risk   Risk         `json:"risk"`
	Reason string       `json:"reason,omitempty"`
}

// Plan is the ordered set of changes plus the aggregate risk.
type Plan struct {
	Changes       []Change `json:"changes"`
	AggregateRisk Risk     `json:"aggregate_risk"`
}

// Empty reports whether the plan has no changes.
func (p Plan) Empty() bool { return len(p.Changes) == 0 }

func sortedAttrChanges(m []AttrChange) []AttrChange {
	sort.Slice(m, func(i, j int) bool { return m[i].Attr < m[j].Attr })
	return m
}
