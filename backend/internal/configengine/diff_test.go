package configengine

import "testing"

func stmt(path, key string, attrs map[string]string) Statement {
	return Statement{Path: path, Key: key, Attributes: attrs}
}

func TestComputeDiffAddUpdateRemove(t *testing.T) {
	running := State{Statements: []Statement{
		stmt("/ip/dns", "servers", map[string]string{"servers": "8.8.8.8"}),
		stmt("/system/ntp", "client", map[string]string{"enabled": "yes"}),
	}}
	desired := State{Statements: []Statement{
		stmt("/ip/dns", "servers", map[string]string{"servers": "1.1.1.1"}), // update
		stmt("/snmp", "community", map[string]string{"name": "neko"}),       // add
	}}

	plan := ComputeDiff(running, desired, RiskOptions{})

	var add, upd, rem int
	for _, c := range plan.Changes {
		switch c.Type {
		case ChangeAdd:
			add++
		case ChangeUpdate:
			upd++
		case ChangeRemove:
			rem++
		}
	}
	if add != 1 || upd != 1 || rem != 1 {
		t.Fatalf("add=%d upd=%d rem=%d, want 1/1/1 (%+v)", add, upd, rem, plan.Changes)
	}
}

func TestComputeDiffNoChange(t *testing.T) {
	s := State{Statements: []Statement{stmt("/ip/dns", "servers", map[string]string{"servers": "1.1.1.1"})}}
	plan := ComputeDiff(s, s, RiskOptions{})
	if !plan.Empty() {
		t.Errorf("expected empty plan, got %+v", plan.Changes)
	}
}

func TestRiskAggregation(t *testing.T) {
	running := State{}
	desired := State{Statements: []Statement{
		stmt("/ip/dns", "servers", map[string]string{"servers": "1.1.1.1"}),          // low
		stmt("/ip/firewall/filter", "drop-all", map[string]string{"action": "drop"}), // high
	}}
	plan := ComputeDiff(running, desired, RiskOptions{})
	if plan.AggregateRisk != RiskHigh {
		t.Errorf("aggregate = %q, want high", plan.AggregateRisk)
	}
}

func TestManagementChannelCritical(t *testing.T) {
	running := State{Statements: []Statement{
		stmt("/ip/address", "10.0.0.1/24", map[string]string{"interface": "ether1"}),
	}}
	desired := State{} // removing the management address
	plan := ComputeDiff(running, desired, RiskOptions{ManagementAddresses: []string{"10.0.0.1/24"}})
	if plan.AggregateRisk != RiskCritical {
		t.Fatalf("aggregate = %q, want critical", plan.AggregateRisk)
	}
}

func TestRemovalBumpsRisk(t *testing.T) {
	running := State{Statements: []Statement{
		stmt("/ip/dns", "static-1", map[string]string{"name": "a.example"}),
	}}
	desired := State{}
	plan := ComputeDiff(running, desired, RiskOptions{})
	// base /ip/dns is low; removal bumps to medium.
	if plan.Changes[0].Risk != RiskMedium {
		t.Errorf("risk = %q, want medium", plan.Changes[0].Risk)
	}
}
