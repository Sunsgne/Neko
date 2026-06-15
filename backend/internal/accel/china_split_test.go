package accel

import (
	"strings"
	"testing"
)

func TestBuildChinaSplitScript(t *testing.T) {
	prefixes := []string{"1.0.1.0/24", "1.0.2.0/23", "1.0.1.0/24", "bogus", "2001:db8::/32"}
	script, count, err := BuildChinaSplitScript(prefixes, ChinaSplitParams{
		WANGateway:      "192.168.1.1",
		OverseasGateway: "100.64.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	// 2 overseas halves + 2 unique valid IPv4 China prefixes (dupe + bogus + v6 dropped).
	if count != 4 {
		t.Fatalf("want 4 routes, got %d", count)
	}
	for _, half := range OverseasHalves {
		if !strings.Contains(script, "dst-address="+half+" gateway=100.64.0.1") {
			t.Errorf("missing overseas route for %s", half)
		}
	}
	if !strings.Contains(script, "dst-address=1.0.1.0/24 gateway=192.168.1.1") {
		t.Error("missing domestic route via WAN gateway")
	}
	if strings.Contains(script, "2001:db8") {
		t.Error("IPv6 prefix must be dropped")
	}
	// Idempotent cleanup present.
	if !strings.Contains(script, CommentOverseas) || !strings.Contains(script, CommentDomestic) {
		t.Error("script must tag routes with neko comments")
	}
	if !strings.Contains(script, "remove $r") {
		t.Error("script must remove prior neko routes (idempotent)")
	}
}

func TestBuildChinaSplitRoutingTable(t *testing.T) {
	script, _, err := BuildChinaSplitScript([]string{"1.0.1.0/24"}, ChinaSplitParams{
		WANGateway:      "ether1",
		OverseasGateway: "wg-hk",
		RoutingTable:    "accel",
		Distance:        5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "routing-table=accel") {
		t.Error("routing-table must be scoped when provided")
	}
	if !strings.Contains(script, "distance=5") {
		t.Error("custom distance must be honored")
	}
}

func TestBuildChinaSplitRequiresGateways(t *testing.T) {
	if _, _, err := BuildChinaSplitScript(nil, ChinaSplitParams{OverseasGateway: "x"}); err == nil {
		t.Error("missing wan_gateway should error")
	}
	if _, _, err := BuildChinaSplitScript(nil, ChinaSplitParams{WANGateway: "x"}); err == nil {
		t.Error("missing overseas_gateway should error")
	}
}
