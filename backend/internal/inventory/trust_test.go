package inventory

import (
	"testing"

	"github.com/neko/sdwan/backend/internal/store"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to store.TrustState
		want     bool
	}{
		{store.TrustDiscovered, store.TrustAuthenticated, true}, // forward one
		{store.TrustAuthenticated, store.TrustEnrolled, true},   // forward one
		{store.TrustEnrolled, store.TrustManaged, true},         // forward one
		{store.TrustDiscovered, store.TrustManaged, false},      // skip stages
		{store.TrustManaged, store.TrustDiscovered, true},       // demotion allowed
		{store.TrustManaged, store.TrustManaged, true},          // idempotent
		{store.TrustUntrusted, store.TrustEnrolled, false},      // skip stages
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}
