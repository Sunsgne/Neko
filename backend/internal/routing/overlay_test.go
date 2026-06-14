package routing

import (
	"testing"

	"github.com/neko/sdwan/backend/internal/store"
)

func TestSelectTunnelType(t *testing.T) {
	wg := &store.CapabilityMatrix{SupportsWireGuard: true}
	if SelectTunnelType(wg, false) != TunnelWireGuard {
		t.Error("v7 device should prefer WireGuard")
	}
	old := &store.CapabilityMatrix{SupportsWireGuard: false}
	if SelectTunnelType(old, true) != TunnelEoIP {
		t.Error("L2 fallback should be EoIP")
	}
	if SelectTunnelType(old, false) != TunnelGRE {
		t.Error("L3 fallback should be GRE")
	}
}

func TestBuildTunnelStateWireGuard(t *testing.T) {
	st := BuildTunnelState("vrf-a", Tunnel{
		Name: "wg-shbj", Type: TunnelWireGuard, RemoteIP: "203.0.113.1",
		TunnelAddr: "100.64.0.1/30", PublicKey: "abc=", ListenPort: 13231,
	})
	var hasIface, hasPeer, hasAddr bool
	for _, s := range st.Statements {
		switch s.Path {
		case "/interface/wireguard":
			hasIface = true
		case "/interface/wireguard/peers":
			hasPeer = true
			if s.Attributes["public-key"] != "abc=" {
				t.Error("peer public-key missing")
			}
		case "/ip/address":
			hasAddr = true
			if s.Attributes["routing-table"] != "vrf-a" {
				t.Error("tunnel addr should be in VRF")
			}
		}
	}
	if !hasIface || !hasPeer || !hasAddr {
		t.Errorf("missing statements iface=%v peer=%v addr=%v", hasIface, hasPeer, hasAddr)
	}
}

func TestBuildTunnelStateGRE(t *testing.T) {
	st := BuildTunnelState("", Tunnel{Name: "gre1", Type: TunnelGRE, LocalIP: "10.0.0.1", RemoteIP: "10.0.0.2"})
	found := false
	for _, s := range st.Statements {
		if s.Path == "/interface/gre" {
			found = true
		}
	}
	if !found {
		t.Error("expected /interface/gre statement")
	}
}
