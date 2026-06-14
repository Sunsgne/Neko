package linkpolicy

import "testing"

func TestFailoverDistances(t *testing.T) {
	st, err := BuildConfig(Policy{
		Strategy: Failover,
		Uplinks: []Uplink{
			{Name: "联通", Gateway: "192.168.2.1", Interface: "ether2", Priority: 2},
			{Name: "电信", Gateway: "192.168.1.1", Interface: "ether1", Priority: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Collect default routes with their distance, keyed by gateway.
	dist := map[string]string{}
	for _, s := range st.Statements {
		if s.Path == "/ip/route" {
			dist[s.Attributes["gateway"]] = s.Attributes["distance"]
			if s.Attributes["check-gateway"] != "ping" {
				t.Errorf("route %s missing check-gateway", s.Attributes["gateway"])
			}
		}
	}
	if dist["192.168.1.1"] != "1" {
		t.Errorf("preferred (电信) should have distance 1, got %q", dist["192.168.1.1"])
	}
	if dist["192.168.2.1"] != "2" {
		t.Errorf("backup (联通) should have distance 2, got %q", dist["192.168.2.1"])
	}
}

func TestLoadBalanceECMPWeights(t *testing.T) {
	st, err := BuildConfig(Policy{
		Strategy: LoadBalance,
		Uplinks: []Uplink{
			{Name: "a", Gateway: "10.0.0.1", Weight: 2},
			{Name: "b", Gateway: "10.0.1.1", Weight: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var ecmp string
	for _, s := range st.Statements {
		if s.Path == "/ip/route" {
			ecmp = s.Attributes["gateway"]
		}
	}
	// weight 2 + 1 => gateway list "10.0.0.1,10.0.0.1,10.0.1.1"
	if ecmp != "10.0.0.1,10.0.0.1,10.0.1.1" {
		t.Errorf("ECMP gateway list = %q", ecmp)
	}
}

func TestValidation(t *testing.T) {
	if _, err := BuildConfig(Policy{Strategy: Failover}); err == nil {
		t.Error("empty uplinks should error")
	}
	if _, err := BuildConfig(Policy{Strategy: Failover, Uplinks: []Uplink{{Name: "x"}}}); err == nil {
		t.Error("missing gateway should error")
	}
	if _, err := BuildConfig(Policy{Strategy: "bogus", Uplinks: []Uplink{{Name: "x", Gateway: "1.1.1.1"}}}); err == nil {
		t.Error("unknown strategy should error")
	}
}
