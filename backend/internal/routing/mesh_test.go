package routing

import (
	"testing"

	"github.com/neko/sdwan/backend/internal/store"
)

func testDevices() map[string]*store.Device {
	return map[string]*store.Device{
		"cpe-sh": {ID: "cpe-sh", Name: "edge-sh-01", Role: "cpe", MgmtAddress: "10.10.1.1:8728"},
		"cpe-bj": {ID: "cpe-bj", Name: "edge-bj-01", Role: "cpe", MgmtAddress: "10.20.1.1:8728"},
		"pop-hk": {ID: "pop-hk", Name: "pop-hk", Role: "backbone", MgmtAddress: "107.1.1.1:55888"},
		"pop-sg": {ID: "pop-sg", Name: "pop-sg", Role: "backbone", MgmtAddress: "107.2.2.2:55888"},
	}
}

func TestBuildMeshPlanTransit(t *testing.T) {
	devs := testDevices()
	plan, err := BuildMeshPlan(devs, MeshPlanInput{
		Topology: MeshTransit,
		LocalAS:  65001,
		Sites: []MeshSiteInput{
			{DeviceID: "cpe-sh", PopDeviceID: "pop-hk", Prefixes: []string{"10.1.0.0/16"}},
			{DeviceID: "cpe-bj", PopDeviceID: "pop-sg", Prefixes: []string{"10.2.0.0/16"}},
		},
		BackbonePath: []string{"pop-hk", "pop-sg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Links) < 3 {
		t.Fatalf("want >=3 links (2 cpe-pop + 1 pop-pop), got %d", len(plan.Links))
	}
	if len(plan.Nodes) != 4 {
		t.Fatalf("want 4 nodes, got %d", len(plan.Nodes))
	}
	hasPopPop := false
	for _, l := range plan.Links {
		if l.Kind == "pop_pop" {
			hasPopPop = true
		}
	}
	if !hasPopPop {
		t.Fatal("transit plan missing pop_pop link")
	}
}

func TestBuildMeshPlanFullMesh(t *testing.T) {
	devs := testDevices()
	plan, err := BuildMeshPlan(devs, MeshPlanInput{
		Topology: MeshFullMesh,
		Sites: []MeshSiteInput{
			{DeviceID: "cpe-sh", PopDeviceID: "pop-hk", Prefixes: []string{"10.1.0.0/16"}},
			{DeviceID: "cpe-bj", PopDeviceID: "pop-sg", Prefixes: []string{"10.2.0.0/16"}},
		},
		BackbonePath: []string{"pop-hk", "pop-sg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Topology != MeshFullMesh {
		t.Fatalf("topology %s", plan.Topology)
	}
}

func TestBuildMeshPlanRequiresTwoSites(t *testing.T) {
	_, err := BuildMeshPlan(testDevices(), MeshPlanInput{
		Sites: []MeshSiteInput{
			{DeviceID: "cpe-sh", PopDeviceID: "pop-hk", Prefixes: []string{"10.0.0.0/8"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for single site")
	}
}

func TestMeshDeployOrder(t *testing.T) {
	nodes := []MeshNodePlan{
		{DeviceID: "c1", Role: "cpe"},
		{DeviceID: "p1", Role: "backbone"},
		{DeviceID: "c2", Role: "cpe"},
	}
	order := MeshDeployOrder(nodes)
	if order[0] != "p1" {
		t.Fatalf("backbone first, got %v", order)
	}
}
