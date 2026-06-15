package routing

import (
	"fmt"
	"hash/fnv"
	"net"
	"strconv"
	"strings"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/store"
)

// WGLink is a negotiated WireGuard site↔POP link with keys and overlay /30
// addressing for both ends.
type WGLink struct {
	ListenPort int `json:"listen_port"`

	CPEInterface string `json:"cpe_interface"`
	POPInterface string `json:"pop_interface"`

	CPEOverlay string `json:"cpe_overlay"` // e.g. 100.64.1.2/30
	POPOverlay string `json:"pop_overlay"` // e.g. 100.64.1.1/30
	POPGateway string `json:"pop_gateway"` // POP overlay IP without prefix (CPE next-hop)

	CPEPrivateKey string `json:"cpe_private_key"`
	CPEPublicKey  string `json:"cpe_public_key"`
	POPPrivateKey string `json:"pop_private_key"`
	POPPublicKey  string `json:"pop_public_key"`

	CPEEndpointHost string `json:"cpe_endpoint_host"`
	POPEndpointHost string `json:"pop_endpoint_host"`

	CPEPublicKeyHint string `json:"cpe_public_key_hint,omitempty"`
	POPPublicKeyHint string `json:"pop_public_key_hint,omitempty"`
}

// FabricPlan is the full bidirectional deploy preview for CPE + POP.
type FabricPlan struct {
	Link       WGLink                `json:"link"`
	Accel      accel.Profile         `json:"accel"`
	CPEDesired configengine.State    `json:"cpe_desired"`
	POPDesired configengine.State    `json:"pop_desired"`
	CPEPlan    configengine.Plan     `json:"cpe_plan"`
	POPPlan    configengine.Plan     `json:"pop_plan"`
}

// BuildWGLink negotiates a production-grade WireGuard link between CPE and POP.
func BuildWGLink(cpe, pop *store.Device, popPublicKeyHint, cpePublicKeyHint string) (WGLink, error) {
	if cpe == nil || pop == nil {
		return WGLink{}, fmt.Errorf("cpe and pop devices required")
	}
	popOverlay, cpeOverlay := AllocateOverlayPair(cpe.ID, pop.ID)
	popGW, _ := splitHostPrefix(popOverlay)

	cpePriv, cpePub, err := GenerateWGKeyPair()
	if err != nil {
		return WGLink{}, err
	}
	popPriv, popPub, err := GenerateWGKeyPair()
	if err != nil {
		return WGLink{}, err
	}
	if popPublicKeyHint != "" {
		popPub = popPublicKeyHint
	}
	if cpePublicKeyHint != "" {
		cpePub = cpePublicKeyHint
	}

	port := 13231
	return WGLink{
		ListenPort:      port,
		CPEInterface:    TunnelNameForPOP(pop.Name),
		POPInterface:    tunnelNameForCPE(cpe.Name),
		CPEOverlay:      cpeOverlay,
		POPOverlay:      popOverlay,
		POPGateway:      popGW,
		CPEPrivateKey:   cpePriv,
		CPEPublicKey:    cpePub,
		POPPrivateKey:   popPriv,
		POPPublicKey:    popPub,
		CPEEndpointHost: HostFromMgmt(cpe.MgmtAddress),
		POPEndpointHost: HostFromMgmt(pop.MgmtAddress),
	}, nil
}

// BuildFabricPlan composes tunnel + acceleration for both sides.
// When mode is empty, only the WireGuard link is configured (mesh onboarding).
func BuildFabricPlan(cpe, pop *store.Device, mode accel.Mode, localWAN string, cpeOverlay string, popPubHint, cpePubHint string, overlayRoutes []string) (FabricPlan, error) {
	link, err := BuildWGLink(cpe, pop, popPubHint, cpePubHint)
	if err != nil {
		return FabricPlan{}, err
	}
	if cpeOverlay != "" {
		if err := applyCPEOverlay(&link, cpeOverlay); err != nil {
			return FabricPlan{}, err
		}
	}

	var profile accel.Profile
	cpeState := link.CPEState()
	popDesired := link.POPState(mode)
	if mode == accel.ModeChinaSplit {
		// POP needs egress NAT; routes come from chnroutes script on CPE.
		popDesired = link.POPState(accel.ModeOverseasDirect)
	}

	switch mode {
	case "", accel.ModeChinaSplit:
		if mode == accel.ModeChinaSplit && localWAN == "" {
			return FabricPlan{}, fmt.Errorf("china_split 需要 local_wan_gateway")
		}
		if mode == accel.ModeChinaSplit {
			profile = accel.Profile{
				Mode:            mode,
				TunnelInterface: link.CPEInterface,
				OverseasGateway: link.POPGateway,
				LocalWANGateway: localWAN,
			}
		}
		// Tunnel-only on CPE (mesh / china_split).
	case accel.ModeOverseasDirect:
		profile = accel.Profile{
			Mode:            mode,
			TunnelInterface: link.CPEInterface,
			OverseasGateway: link.POPGateway,
			LocalWANGateway: localWAN,
			OverseasDNS:     []string{"8.8.8.8", "1.1.1.1"},
		}
	case accel.ModeSmartSplit:
		if localWAN == "" {
			return FabricPlan{}, fmt.Errorf("smart_split 需要 local_wan_gateway")
		}
		profile = accel.Profile{
			Mode:            mode,
			TunnelInterface: link.CPEInterface,
			OverseasGateway: link.POPGateway,
			LocalWANGateway: localWAN,
			DomesticDNS:     []string{"223.5.5.5", "114.114.114.114"},
		}
	case accel.ModeDomesticDirect:
		return FabricPlan{}, fmt.Errorf("domestic_direct 不需要 POP 隧道，请仅下发 CPE 加速配置")
	default:
		return FabricPlan{}, fmt.Errorf("unknown accel mode %s", mode)
	}

	cpeDesired := cpeState
	if mode != "" && mode != accel.ModeChinaSplit {
		accelState, err := accel.BuildConfig(profile)
		if err != nil {
			return FabricPlan{}, err
		}
		cpeDesired = configengine.Merge(cpeState, accelState)
	}
	if len(overlayRoutes) > 0 {
		cpeDesired = configengine.Merge(cpeDesired, meshOverlayRoutes(link.CPEInterface, overlayRoutes))
	}

	return FabricPlan{
		Link:       link,
		Accel:      profile,
		CPEDesired: cpeDesired,
		POPDesired: popDesired,
		CPEPlan:    configengine.ComputeDiff(configengine.State{}, cpeDesired, configengine.RiskOptions{}),
		POPPlan:    configengine.ComputeDiff(configengine.State{}, popDesired, configengine.RiskOptions{}),
	}, nil
}

func meshOverlayRoutes(iface string, cidrs []string) configengine.State {
	return meshReachabilityRoutes(cidrs, iface, "")
}

// meshReachabilityRoutes installs static routes via interface name and/or IP gateway.
func meshReachabilityRoutes(cidrs []string, iface, gateway string) configengine.State {
	var sts []configengine.Statement
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		attrs := map[string]string{
			"dst-address": c,
			"distance":    "1",
			"comment":     "neko-mesh: remote site",
		}
		if gateway != "" {
			attrs["gateway"] = gateway
		} else if iface != "" {
			attrs["gateway"] = iface
		}
		key := c + "@" + gateway + iface
		sts = append(sts, configengine.Statement{Path: "/ip/route", Key: key, Attributes: attrs})
	}
	return configengine.State{Statements: sts}
}

// CPEState returns RouterOS statements for the CPE WireGuard client side.
func (l WGLink) CPEState() configengine.State {
	return buildWGEndpoint(wgEndpoint{
		iface:       l.CPEInterface,
		privateKey:  l.CPEPrivateKey,
		listenPort:  l.ListenPort,
		overlayAddr: l.CPEOverlay,
		peerKey:     l.POPPublicKey,
		peerHost:    l.POPEndpointHost,
		peerPort:    l.ListenPort,
		peerName:    l.CPEInterface + "-peer",
	})
}

// POPState returns RouterOS statements for the POP WireGuard hub side.
func (l WGLink) POPState(mode accel.Mode) configengine.State {
	st := buildWGEndpoint(wgEndpoint{
		iface:       l.POPInterface,
		privateKey:  l.POPPrivateKey,
		listenPort:  l.ListenPort,
		overlayAddr: l.POPOverlay,
		peerKey:     l.CPEPublicKey,
		peerHost:    l.CPEEndpointHost,
		peerPort:    l.ListenPort,
		peerName:    l.POPInterface + "-peer",
		allowed:     "0.0.0.0/0",
	})
	if mode == accel.ModeOverseasDirect {
		st = configengine.Merge(st, popEgressNAT(l.POPInterface))
	}
	return st
}

type wgEndpoint struct {
	iface, privateKey, overlayAddr, peerKey, peerHost, peerName, allowed string
	listenPort, peerPort                                                   int
}

func buildWGEndpoint(e wgEndpoint) configengine.State {
	port := e.listenPort
	if port == 0 {
		port = 13231
	}
	peerPort := e.peerPort
	if peerPort == 0 {
		peerPort = port
	}
	allowed := e.allowed
	if allowed == "" {
		allowed = "0.0.0.0/0"
	}
	var sts []configengine.Statement
	sts = append(sts,
		configengine.Statement{
			Path: "/interface/wireguard", Key: e.iface,
			Attributes: wgIfaceAttrs(e.iface, port, e.privateKey),
		},
		configengine.Statement{
			Path: "/interface/wireguard/peers", Key: e.peerName,
			Attributes: map[string]string{
				"interface":        e.iface,
				"public-key":       e.peerKey,
				"endpoint-address": e.peerHost,
				"endpoint-port":    itoa(peerPort),
				"allowed-address":  allowed,
				"comment":          "neko-fabric: peer",
			},
		},
	)
	if e.overlayAddr != "" {
		sts = append(sts, configengine.Statement{
			Path: "/ip/address", Key: e.overlayAddr + "@" + e.iface,
			Attributes: map[string]string{"address": e.overlayAddr, "interface": e.iface},
		})
	}
	return configengine.State{Statements: sts}
}

func popEgressNAT(iface string) configengine.State {
	return configengine.State{Statements: []configengine.Statement{
		{
			Path: "/ip/firewall/nat", Key: "neko-pop-egress-" + iface,
			Attributes: map[string]string{
				"chain":         "srcnat",
				"in-interface":  iface,
				"action":        "masquerade",
				"comment":       "neko-fabric: POP egress for CPE traffic",
			},
		},
	}}
}

// AllocateOverlayPair returns (popAddr, cpeAddr) as a valid /30 pair.
func AllocateOverlayPair(cpeID, popID string) (popAddr, cpeAddr string) {
	h := fnv32(cpeID + ":" + popID)
	// 100.64.0.0/16 — pick a /30 block; host .1 = POP, .2 = CPE.
	third := byte((h >> 8) % 254)
	base := byte((h % 63) * 4) // 0,4,8,...252 — aligned /30
	if base < 4 {
		base = 4
	}
	return fmt.Sprintf("100.64.%d.%d/30", third, base+1),
		fmt.Sprintf("100.64.%d.%d/30", third, base+2)
}

func applyCPEOverlay(link *WGLink, cpeOverlay string) error {
	if _, _, err := net.ParseCIDR(cpeOverlay); err != nil {
		return fmt.Errorf("invalid cpe_overlay: %w", err)
	}
	popGW, err := popGatewayFromCPEOverlay(cpeOverlay)
	if err != nil {
		return err
	}
	link.CPEOverlay = cpeOverlay
	link.POPGateway = popGW
	_, prefix := splitHostPrefix(cpeOverlay)
	link.POPOverlay = popGW + "/" + prefix
	return nil
}

// popGatewayFromCPEOverlay returns the POP-side IP (CPE next-hop) in the same /30.
func popGatewayFromCPEOverlay(cpeOverlay string) (string, error) {
	ip, ipNet, err := net.ParseCIDR(cpeOverlay)
	if err != nil {
		return "", err
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "", fmt.Errorf("only IPv4 overlay supported")
	}
	ones, _ := ipNet.Mask.Size()
	if ones != 30 {
		return "", fmt.Errorf("overlay must be /30")
	}
	base := int(ip4[3]) &^ 3
	host1 := base + 1
	host2 := base + 2
	last := int(ip4[3])
	switch last {
	case host1:
		return fmt.Sprintf("%d.%d.%d.%d", ip4[0], ip4[1], ip4[2], host2), nil
	case host2:
		return fmt.Sprintf("%d.%d.%d.%d", ip4[0], ip4[1], ip4[2], host1), nil
	default:
		return "", fmt.Errorf("cpe overlay %s is not a valid /30 host address", cpeOverlay)
	}
}

func splitHostPrefix(cidr string) (host, prefix string) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		parts := strings.Split(cidr, "/")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return cidr, "30"
	}
	ones, _ := ipNet.Mask.Size()
	return ip.String(), strconv.Itoa(ones)
}

func tunnelNameForCPE(cpeName string) string {
	name := "wg-cpe-" + cpeName
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	out := b.String()
	if len(out) > 24 {
		out = out[:24]
	}
	if out == "" {
		return "wg-cpe"
	}
	return out
}

func fnv32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
