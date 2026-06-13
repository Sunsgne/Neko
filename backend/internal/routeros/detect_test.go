package routeros

import (
	"testing"

	"github.com/neko/sdwan/backend/internal/store"
)

func TestDetectRouterBOARD(t *testing.T) {
	f := DeviceFacts{
		Resource: SystemResource{
			BoardName:    "RB5009UG+S+IN",
			Architecture: "arm64",
			Version:      "7.14.3 (stable)",
		},
		Routerboard: RouterboardInfo{
			Routerboard:     true,
			Model:           "RB5009UG+S+IN",
			SerialNumber:    "HEX123",
			CurrentFirmware: "7.14.3",
			UpgradeFirmware: "7.14.3",
		},
		License:    License{Level6: "6"},
		DeviceMode: DeviceMode{Mode: "enterprise"},
		Interfaces: []Interface{
			{Name: "ether1", Type: "ether"},
			{Name: "sfp-sfpplus1", Type: "ether"},
		},
	}
	d := Detect(f)
	if d.Platform != store.PlatformRouterBOARD {
		t.Errorf("platform = %q, want routerboard", d.Platform)
	}
	if d.Model != "RB5009UG+S+IN" {
		t.Errorf("model = %q", d.Model)
	}
	if d.Serial != "HEX123" {
		t.Errorf("serial = %q", d.Serial)
	}
	if !d.Capabilities.SupportsBGP || !d.Capabilities.SupportsOSPF {
		t.Error("v7 should support BGP/OSPF")
	}
	if !d.Capabilities.SupportsWireGuard {
		t.Error("v7 should support WireGuard")
	}
	if d.Capabilities.LicenseLevel != 6 {
		t.Errorf("license = %d, want 6", d.Capabilities.LicenseLevel)
	}
	// sfp-sfpplus1 should be detected as a 10G SFP+ port.
	var sfp *store.InterfaceCapability
	for i := range d.Capabilities.Interfaces {
		if d.Capabilities.Interfaces[i].Name == "sfp-sfpplus1" {
			sfp = &d.Capabilities.Interfaces[i]
		}
	}
	if sfp == nil || sfp.SpeedMbps != 10000 {
		t.Errorf("sfp+ speed detection failed: %+v", sfp)
	}
}

func TestDetectCHR(t *testing.T) {
	f := DeviceFacts{
		Resource:    SystemResource{BoardName: "CHR", Architecture: "x86_64", Version: "7.13"},
		Routerboard: RouterboardInfo{Routerboard: false},
		License:     License{Level: 4},
	}
	d := Detect(f)
	if d.Platform != store.PlatformCHR {
		t.Errorf("platform = %q, want chr", d.Platform)
	}
	if d.Capabilities.LicenseLevel != 4 {
		t.Errorf("license = %d, want 4", d.Capabilities.LicenseLevel)
	}
}

func TestDetectX86(t *testing.T) {
	f := DeviceFacts{
		Resource:    SystemResource{BoardName: "x86", Architecture: "x86_64", Version: "7.11"},
		Routerboard: RouterboardInfo{Routerboard: false},
	}
	d := Detect(f)
	if d.Platform != store.PlatformX86 {
		t.Errorf("platform = %q, want x86", d.Platform)
	}
}

func TestDetectV6RoutingPackage(t *testing.T) {
	f := DeviceFacts{
		Resource:    SystemResource{BoardName: "RB750Gr3", Architecture: "mmips", Version: "6.49.10 (stable)"},
		Routerboard: RouterboardInfo{Routerboard: true, Model: "RB750Gr3"},
		Packages: []Package{
			{Name: "system", Version: "6.49.10"},
			{Name: "routing", Version: "6.49.10"},
		},
	}
	d := Detect(f)
	if !d.Capabilities.SupportsBGP {
		t.Error("v6 with routing package should support BGP")
	}
	if d.Capabilities.SupportsWireGuard {
		t.Error("v6 should NOT support WireGuard")
	}
}

func TestDetectContainerPackage(t *testing.T) {
	f := DeviceFacts{
		Resource:    SystemResource{BoardName: "CCR2004", Architecture: "arm64", Version: "7.14"},
		Routerboard: RouterboardInfo{Routerboard: true, Model: "CCR2004-1G-12S+2XS"},
		Packages:    []Package{{Name: "container", Version: "7.14"}},
	}
	d := Detect(f)
	if !d.Capabilities.SupportsContainer {
		t.Error("expected container support")
	}
}
