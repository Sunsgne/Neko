package dns

import (
	"context"
	"strings"
	"testing"
)

func TestBuildConfigPrimaryAndSplit(t *testing.T) {
	primary := []Server{
		{ID: "a", Address: "223.5.5.5"},
		{ID: "b", Address: "114.114.114.114"},
	}
	splits := []SplitRule{
		{MatchSuffix: ".cn", Servers: []string{"202.96.209.133"}},
	}
	st := BuildConfig(primary, splits)

	var settings, fwd, static bool
	for _, s := range st.Statements {
		switch s.Path {
		case "/ip/dns":
			settings = true
			if !strings.Contains(s.Attributes["servers"], "223.5.5.5") {
				t.Error("primary servers missing")
			}
		case "/ip/dns/forwarders":
			fwd = true
		case "/ip/dns/static":
			static = true
			if s.Attributes["match-subdomain"] != "yes" {
				t.Error("split should match subdomains")
			}
		}
	}
	if !settings || !fwd || !static {
		t.Errorf("missing statements settings=%v fwd=%v static=%v", settings, fwd, static)
	}
}

func TestBuildConfigDoH(t *testing.T) {
	st := BuildConfig([]Server{
		{Kind: "udp", Address: "223.5.5.5"},
		{Kind: "doh", Address: "https://dns.alidns.com/dns-query"},
	}, nil)
	var settings map[string]string
	for _, s := range st.Statements {
		if s.Path == "/ip/dns" {
			settings = s.Attributes
		}
	}
	if settings["servers"] != "223.5.5.5" {
		t.Errorf("plain server should be in servers, got %q", settings["servers"])
	}
	if settings["use-doh-server"] != "https://dns.alidns.com/dns-query" {
		t.Errorf("DoH should set use-doh-server, got %q", settings["use-doh-server"])
	}
	if settings["verify-doh-cert"] != "yes" {
		t.Error("DoH should verify cert")
	}
}

func TestCheckerHandlesUnreachable(t *testing.T) {
	c := NewChecker()
	c.Timeout = 500_000_000 // 500ms
	// 127.0.0.1:1 is a closed local port: UDP writes draw an ICMP
	// port-unreachable, so the lookup fails deterministically.
	res := c.Check(context.Background(), Server{ID: "x", Address: "127.0.0.1:1"})
	if res.Healthy {
		t.Error("unreachable server should be unhealthy")
	}
	if res.ServerID != "x" {
		t.Errorf("server id = %q", res.ServerID)
	}
}
