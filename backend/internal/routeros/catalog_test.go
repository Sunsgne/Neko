package routeros

import "testing"

func TestValidPath(t *testing.T) {
	ok := []string{"/ip/address", "/interface/wireguard/peers", "/routing/bgp/connection", "/system/identity", "/ipv6/firewall/nat"}
	for _, p := range ok {
		if !ValidPath(p) {
			t.Errorf("expected %q valid", p)
		}
	}
	bad := []string{"", "ip/address", "/ip/../etc", "/ip address", "/IP/Address", "/ip/address?x=1", "//ip", "/ip/"}
	for _, p := range bad {
		if ValidPath(p) {
			t.Errorf("expected %q invalid", p)
		}
	}
}

func TestCatalogNonEmpty(t *testing.T) {
	if len(Catalog) < 10 {
		t.Fatalf("catalog should cover all menus, got %d groups", len(Catalog))
	}
	for _, g := range Catalog {
		if g.Menu == "" || len(g.Sections) == 0 {
			t.Errorf("group %q has no sections", g.Menu)
		}
		for _, sec := range g.Sections {
			if !ValidPath(sec.Path) {
				t.Errorf("catalog path %q (%s) is not a valid path", sec.Path, sec.Label)
			}
		}
	}
}
