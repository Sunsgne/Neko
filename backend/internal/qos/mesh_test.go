package qos

import (
	"testing"

	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/routing"
	"github.com/neko/sdwan/backend/internal/store"
)

func TestRulesForSite(t *testing.T) {
	rules, err := RulesForSite("site-a", []string{"10.0.0.0/24", "10.0.1.0/24"}, "10M", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(rules))
	}
	if rules[0].Target != "10.0.0.0/24" || rules[0].MaxLimit != "10M" {
		t.Fatalf("unexpected rule: %+v", rules[0])
	}

	rules, err = RulesForSite("site-a", nil, "5M", "192.168.1.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Target != "192.168.1.0/24" {
		t.Fatalf("unexpected rules: %+v", rules)
	}

	if _, err := RulesForSite("site-a", nil, "10M", ""); err == nil {
		t.Fatal("missing target/prefixes should fail")
	}
	if _, err := RulesForSite("site-a", []string{"10.0.0.0/24"}, "", ""); err != nil {
		t.Fatal("empty rate should return nil")
	}
}

func TestApplyToMeshPlan(t *testing.T) {
	cpeID := "cpe-1"
	plan := routing.MeshPlan{
		Nodes: []routing.MeshNodePlan{{
			DeviceID: cpeID,
			Desired:  configengine.State{Statements: []configengine.Statement{{Path: "/interface/wireguard", Key: "wg0"}}},
		}},
	}
	sites := []routing.MeshSiteInput{{
		DeviceID: cpeID, Prefixes: []string{"10.0.0.0/24"}, RateLimit: "20M",
	}}
	devices := map[string]*store.Device{cpeID: {ID: cpeID, Name: "branch-a"}}

	if err := ApplyToMeshPlan(&plan, sites, devices); err != nil {
		t.Fatal(err)
	}
	if len(plan.Nodes[0].Desired.Statements) < 2 {
		t.Fatalf("expected queue merged, got %d stmts", len(plan.Nodes[0].Desired.Statements))
	}
	found := false
	for _, st := range plan.Nodes[0].Desired.Statements {
		if st.Path == "/queue/simple" {
			found = true
			if st.Attributes["max-limit"] != "20M/20M" {
				t.Fatalf("max-limit: %s", st.Attributes["max-limit"])
			}
		}
	}
	if !found {
		t.Fatal("no simple queue in desired")
	}
}
