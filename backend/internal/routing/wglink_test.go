package routing

import (
	"strings"
	"testing"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/store"
)

func TestAllocateOverlayPair(t *testing.T) {
	pop, cpe := AllocateOverlayPair("cpe-a", "pop-b")
	if pop == cpe {
		t.Fatalf("pop and cpe overlays must differ: %s", pop)
	}
	popHost := strings.Split(pop, "/")[0]
	if PopPeerOf(cpe) != popHost {
		t.Fatalf("PopPeerOf(%s) = %s, want %s", cpe, PopPeerOf(cpe), popHost)
	}
}

func TestPopPeerOfAlignedSubnet(t *testing.T) {
	if got := PopPeerOf("100.64.88.118/30"); got != "100.64.88.117" {
		t.Fatalf("got %s", got)
	}
}

func TestBuildFabricPlanBothSides(t *testing.T) {
	cpe := &store.Device{ID: "c1", Name: "edge-sh-01", MgmtAddress: "10.10.1.1:8728"}
	pop := &store.Device{ID: "p1", Name: "pop-hk", MgmtAddress: "107.155.12.197:55888"}
	plan, err := BuildFabricPlan(cpe, pop, accel.ModeOverseasDirect, "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.CPEDesired.Statements) == 0 || len(plan.POPDesired.Statements) == 0 {
		t.Fatal("expected non-empty CPE and POP desired states")
	}
	if plan.Link.CPEInterface == "" || plan.Link.POPInterface == "" {
		t.Fatal("expected tunnel interface names")
	}
	if plan.Link.POPGateway == "" {
		t.Fatal("expected POP gateway")
	}
}
