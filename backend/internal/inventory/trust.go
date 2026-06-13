package inventory

import "github.com/neko/sdwan/backend/internal/store"

// trustOrder defines the linear progression of the device trust lifecycle.
//
//	untrusted -> discovered -> authenticated -> enrolled -> managed
var trustOrder = map[store.TrustState]int{
	store.TrustUntrusted:     0,
	store.TrustDiscovered:    1,
	store.TrustAuthenticated: 2,
	store.TrustEnrolled:      3,
	store.TrustManaged:       4,
}

// CanTransition reports whether moving from -> to is allowed. Forward steps of
// exactly one stage are allowed; any backward move (e.g. quarantine) is always
// allowed so operators can demote a misbehaving device.
func CanTransition(from, to store.TrustState) bool {
	f, ok1 := trustOrder[from]
	t, ok2 := trustOrder[to]
	if !ok1 || !ok2 {
		return false
	}
	if t <= f {
		return true // backward / same: demotion or idempotent
	}
	return t == f+1 // forward: one step at a time
}
