package routing

import "testing"

func TestBGPKind(t *testing.T) {
	if (BGPNeighbor{LocalAS: 65000, PeerAS: 65000}).Kind() != "ibgp" {
		t.Error("same AS should be iBGP")
	}
	if (BGPNeighbor{LocalAS: 65000, PeerAS: 65001}).Kind() != "ebgp" {
		t.Error("different AS should be eBGP")
	}
}

func validIntent() Intent {
	return Intent{
		TenantID:        "ten1",
		DeviceID:        "dev1",
		VRF:             "vrf-ten1",
		TenantCommunity: "65000:1001",
		Static:          []StaticRoute{{DstPrefix: "0.0.0.0/0", Gateway: "10.0.0.1", Distance: 1}},
		BGP: []BGPNeighbor{
			{Name: "pop-a", PeerAddress: "100.64.0.1", LocalAS: 65000, PeerAS: 64500, BFD: true, ImportFilter: "in-pop", ExportFilter: "out-pop"},
		},
	}
}

func TestValidatePasses(t *testing.T) {
	if issues := Validate(validIntent()); HasErrors(issues) {
		t.Fatalf("expected no errors, got %+v", issues)
	}
}

func TestValidateRequiresVRFAndCommunity(t *testing.T) {
	in := validIntent()
	in.VRF = ""
	in.TenantCommunity = ""
	issues := Validate(in)
	if !HasErrors(issues) {
		t.Fatal("expected errors for missing VRF/community")
	}
	codes := map[string]bool{}
	for _, i := range issues {
		codes[i.Code] = true
	}
	if !codes["missing_vrf"] || !codes["missing_community"] {
		t.Errorf("expected leak-prevention errors, got %+v", issues)
	}
}

func TestValidateEBGPRequiresFilters(t *testing.T) {
	in := validIntent()
	in.BGP[0].ImportFilter = ""
	issues := Validate(in)
	if !HasErrors(issues) {
		t.Fatal("eBGP without filter must be an error (leak prevention)")
	}
}

func TestValidateRedistributionRequiresFilter(t *testing.T) {
	in := validIntent()
	in.Redistributions = []Redistribution{{Source: "connected", Into: "bgp"}}
	if !HasErrors(Validate(in)) {
		t.Fatal("unfiltered redistribution must be an error")
	}
}

func TestValidateIBGPFullMeshWarning(t *testing.T) {
	in := validIntent()
	in.BGP = []BGPNeighbor{
		{Name: "i1", LocalAS: 65000, PeerAS: 65000},
		{Name: "i2", LocalAS: 65000, PeerAS: 65000},
		{Name: "i3", LocalAS: 65000, PeerAS: 65000},
	}
	issues := Validate(in)
	found := false
	for _, i := range issues {
		if i.Code == "ibgp_full_mesh" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ibgp_full_mesh warning, got %+v", issues)
	}
}

func TestBuildStateGeneratesStatements(t *testing.T) {
	st := BuildState(validIntent())
	var hasRoute, hasBGP bool
	for _, s := range st.Statements {
		if s.Path == "/ip/route" {
			hasRoute = true
			if s.Attributes["routing-table"] != "vrf-ten1" {
				t.Error("static route should be in tenant VRF")
			}
		}
		if s.Path == "/routing/bgp/connection" {
			hasBGP = true
			if s.Attributes["use-bfd"] != "yes" {
				t.Error("BFD should be enabled")
			}
			if s.Attributes["kind"] != "ebgp" {
				t.Error("expected ebgp kind")
			}
		}
	}
	if !hasRoute || !hasBGP {
		t.Errorf("missing statements: route=%v bgp=%v", hasRoute, hasBGP)
	}
}
