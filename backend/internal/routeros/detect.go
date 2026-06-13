package routeros

import (
	"strconv"
	"strings"

	"github.com/neko/sdwan/backend/internal/store"
)

// Detection is the normalized result of analyzing DeviceFacts.
type Detection struct {
	Platform     store.DevicePlatform
	Model        string
	Serial       string
	Capabilities store.CapabilityMatrix
}

// Detect analyzes raw facts and produces platform classification, model,
// serial, and a normalized capability matrix.
func Detect(f DeviceFacts) Detection {
	platform := classifyPlatform(f)
	model, serial := modelAndSerial(f, platform)

	major := majorVersion(f.Resource.Version)
	pkgs := packageNames(f.Packages)
	hasPkg := func(name string) bool {
		for _, p := range pkgs {
			if p == name {
				return true
			}
		}
		return false
	}

	// RouterOS 7 has routing (OSPF/BGP) and WireGuard built into the system
	// package; RouterOS 6 ships them via the "routing" package.
	routingV6 := hasPkg("routing")
	supportsRouting := major >= 7 || routingV6

	cap := store.CapabilityMatrix{
		RouterOSVersion:   f.Resource.Version,
		Architecture:      f.Resource.Architecture,
		BoardName:         f.Resource.BoardName,
		Packages:          pkgs,
		LicenseLevel:      licenseLevel(f),
		DeviceMode:        f.DeviceMode.Mode,
		Interfaces:        normalizeInterfaces(f.Interfaces),
		SupportsBGP:       supportsRouting,
		SupportsOSPF:      supportsRouting,
		SupportsWireGuard: major >= 7,
		SupportsContainer: hasPkg("container"),
	}

	return Detection{Platform: platform, Model: model, Serial: serial, Capabilities: cap}
}

func classifyPlatform(f DeviceFacts) store.DevicePlatform {
	if f.Routerboard.Routerboard {
		return store.PlatformRouterBOARD
	}
	board := strings.ToUpper(strings.TrimSpace(f.Resource.BoardName))
	switch {
	case strings.Contains(board, "CHR"):
		return store.PlatformCHR
	case board == "X86" || strings.Contains(strings.ToLower(f.Resource.Architecture), "x86"):
		// Non-routerboard, non-CHR x86_64 install.
		return store.PlatformX86
	case board == "":
		return store.PlatformUnknown
	default:
		// Non-routerboard with a board name we don't recognize: treat as x86
		// install if architecture is x86_64, otherwise unknown.
		if strings.Contains(strings.ToLower(f.Resource.Architecture), "x86") {
			return store.PlatformX86
		}
		return store.PlatformUnknown
	}
}

func modelAndSerial(f DeviceFacts, platform store.DevicePlatform) (string, string) {
	if platform == store.PlatformRouterBOARD && f.Routerboard.Model != "" {
		return f.Routerboard.Model, f.Routerboard.SerialNumber
	}
	if f.Resource.BoardName != "" {
		return f.Resource.BoardName, f.Routerboard.SerialNumber
	}
	return string(platform), f.Routerboard.SerialNumber
}

func licenseLevel(f DeviceFacts) int {
	if f.License.Level > 0 {
		return f.License.Level
	}
	if n, err := strconv.Atoi(strings.TrimSpace(f.License.Level6)); err == nil {
		return n
	}
	return 0
}

func majorVersion(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	// Versions look like "7.14.3" or "6.49.10 (stable)".
	if i := strings.IndexByte(v, ' '); i >= 0 {
		v = v[:i]
	}
	parts := strings.SplitN(v, ".", 2)
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}

func packageNames(pkgs []Package) []string {
	out := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		if p.Disabled {
			continue
		}
		out = append(out, p.Name)
	}
	return out
}

func normalizeInterfaces(ifaces []Interface) []store.InterfaceCapability {
	out := make([]store.InterfaceCapability, 0, len(ifaces))
	for _, i := range ifaces {
		out = append(out, store.InterfaceCapability{
			Name:      i.Name,
			Type:      i.Type,
			SpeedMbps: speedForType(i.Type, i.Name),
			Features:  featuresForInterface(i),
		})
	}
	return out
}

func speedForType(typ, name string) int {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "sfp28") || strings.Contains(n, "qsfp"):
		return 25000
	case strings.Contains(n, "sfp-plus") || strings.Contains(n, "sfpplus") || strings.Contains(n, "sfp+"):
		return 10000
	case strings.Contains(n, "sfp"):
		return 1000
	}
	switch strings.ToLower(typ) {
	case "ether":
		return 1000
	default:
		return 0
	}
}

func featuresForInterface(i Interface) []string {
	feats := []string{}
	n := strings.ToLower(i.Name)
	t := strings.ToLower(i.Type)
	switch {
	case strings.Contains(n, "sfp28") || strings.Contains(n, "qsfp"):
		feats = append(feats, "sfp28")
	case strings.Contains(n, "sfp-plus") || strings.Contains(n, "sfpplus") || strings.Contains(n, "sfp+"):
		feats = append(feats, "sfp+")
	case strings.Contains(n, "sfp"):
		feats = append(feats, "sfp")
	}
	if t == "wlan" || strings.HasPrefix(n, "wlan") || strings.HasPrefix(n, "wifi") {
		feats = append(feats, "wireless")
	}
	if t == "wg" || strings.HasPrefix(n, "wg") {
		feats = append(feats, "wireguard")
	}
	if t == "ether" {
		feats = append(feats, "l2", "l3")
	}
	return feats
}
