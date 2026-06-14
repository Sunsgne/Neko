package accel

import (
	"strings"
	"testing"
)

func TestOverseasDirectNoSplit(t *testing.T) {
	st, err := BuildConfig(Profile{
		Mode:            ModeOverseasDirect,
		TunnelInterface: "wg-hk",
		OverseasGateway: "100.64.0.1",
		OverseasExitIP:  "203.0.113.9",
		OverseasDNS:     []string{"8.8.8.8", "1.1.1.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var hasDefault, hasNAT, hasMangle, hasDNS bool
	for _, s := range st.Statements {
		switch s.Path {
		case "/ip/route":
			if s.Attributes["dst-address"] == "0.0.0.0/0" && s.Attributes["gateway"] == "100.64.0.1" {
				hasDefault = true
			}
		case "/ip/firewall/nat":
			hasNAT = true
		case "/ip/firewall/mangle":
			hasMangle = true
		case "/ip/dns":
			hasDNS = true
		}
	}
	if !hasDefault {
		t.Error("overseas_direct must set default route to overseas gateway")
	}
	if !hasNAT {
		t.Error("overseas_direct must masquerade out the tunnel")
	}
	if hasMangle {
		t.Error("overseas_direct must NOT create split/mangle rules (不分流)")
	}
	if !hasDNS {
		t.Error("overseas_direct should set overseas DNS directly")
	}
}

func TestSmartSplitMarksOverseas(t *testing.T) {
	st, err := BuildConfig(Profile{
		Mode:            ModeSmartSplit,
		TunnelInterface: "wg-hk",
		OverseasGateway: "100.64.0.1",
		LocalWANGateway: "192.168.1.1",
		DomesticDNS:     []string{"223.5.5.5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var hasMangle, hasMarkedRoute bool
	for _, s := range st.Statements {
		if s.Path == "/ip/firewall/mangle" && s.Attributes["new-routing-mark"] == overseasMark {
			hasMangle = true
		}
		if s.Path == "/ip/route" && s.Attributes["routing-mark"] == overseasMark {
			hasMarkedRoute = true
		}
	}
	if !hasMangle || !hasMarkedRoute {
		t.Errorf("smart_split must mark + route overseas traffic via tunnel (mangle=%v route=%v)", hasMangle, hasMarkedRoute)
	}
}

func TestInvalidProfiles(t *testing.T) {
	if _, err := BuildConfig(Profile{Mode: ModeOverseasDirect}); err == nil {
		t.Error("overseas_direct without tunnel/gateway should error")
	}
	if _, err := BuildConfig(Profile{Mode: "bogus"}); err == nil {
		t.Error("unknown mode should error")
	}
}

func TestDescribe(t *testing.T) {
	if !strings.Contains(Describe(ModeOverseasDirect), "海外") {
		t.Error("describe should mention 海外")
	}
}
