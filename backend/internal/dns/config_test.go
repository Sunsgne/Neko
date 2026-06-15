package dns

import (
	"context"
	"strings"
	"testing"

	"github.com/neko/sdwan/backend/internal/configengine"
)

func TestBuildConfigPrimaryAndSplit(t *testing.T) {
	primary := []Server{
		{ID: "a", Address: "223.5.5.5"},
		{ID: "b", Address: "114.114.114.114"},
	}
	splits := []SplitRule{
		{MatchSuffix: ".cn", Servers: []string{"202.96.209.133"}},
	}
	st := BuildConfig(primary, splits, Options{})

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

func TestBuildConfigDoHHostname(t *testing.T) {
	st := BuildConfig([]Server{
		{Kind: "udp", Address: "223.5.5.5"},
		{Kind: "doh", Address: "https://dns.alidns.com/dns-query"},
	}, nil, Options{})
	settings := dnsSettings(st)
	if settings["servers"] != "223.5.5.5" {
		t.Errorf("plain server should be in servers, got %q", settings["servers"])
	}
	if settings["use-doh-server"] != "https://dns.alidns.com/dns-query" {
		t.Errorf("DoH should set use-doh-server, got %q", settings["use-doh-server"])
	}
	// Hostname endpoint -> verification auto-enabled.
	if settings["verify-doh-cert"] != "yes" {
		t.Errorf("hostname DoH should verify cert, got %q", settings["verify-doh-cert"])
	}
}

func TestBuildConfigDoHIPDisablesVerify(t *testing.T) {
	// IP-based DoH endpoint (with port) cannot present a verifiable cert.
	st := BuildConfig([]Server{
		{Kind: "doh", Address: "https://202.101.51.194:9291/dns-query"},
	}, nil, Options{})
	settings := dnsSettings(st)
	if settings["use-doh-server"] != "https://202.101.51.194:9291/dns-query" {
		t.Errorf("DoH URL with port should be preserved, got %q", settings["use-doh-server"])
	}
	if settings["verify-doh-cert"] != "no" {
		t.Errorf("IP-literal DoH should auto-disable cert verify, got %q", settings["verify-doh-cert"])
	}
}

func TestBuildConfigDoHExplicitOverride(t *testing.T) {
	on := true
	st := BuildConfig([]Server{
		{Kind: "doh", Address: "https://202.101.51.194:9291/dns-query"},
	}, nil, Options{VerifyDoHCert: &on})
	if dnsSettings(st)["verify-doh-cert"] != "yes" {
		t.Error("explicit override should force verify-doh-cert=yes")
	}
}

func dnsSettings(st configengine.State) map[string]string {
	for _, s := range st.Statements {
		if s.Path == "/ip/dns" {
			return s.Attributes
		}
	}
	return map[string]string{}
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
