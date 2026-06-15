// Package accel models customer-side (CPE) acceleration business modes and
// generates the RouterOS configuration that implements them.
//
// Modes:
//   - smart_split    智能分流：国内直连本地出口，海外流量走 SD-WAN/海外出口。
//   - overseas_direct 海外运营：不做分流，全量流量直接走海外出口 IP。
//   - domestic_direct 国内直连：全量走本地出口（用于回退/对照）。
package accel

import (
	"fmt"
	"strings"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// Mode selects the acceleration business mode.
type Mode string

const (
	ModeSmartSplit     Mode = "smart_split"
	ModeOverseasDirect Mode = "overseas_direct"
	ModeDomesticDirect Mode = "domestic_direct"
	// ModeChinaSplit uses the chnroutes prefix table: domestic via local WAN,
	// overseas 0.0.0.0/1+128.0.0.0/1 via the SD-WAN tunnel (script delivery).
	ModeChinaSplit Mode = "china_split"
)

// Profile parameterizes the acceleration configuration for a CPE.
type Profile struct {
	Mode Mode `json:"mode"`
	// TunnelInterface is the SD-WAN / overseas exit interface (e.g. a WireGuard
	// or EoIP tunnel toward a gateway node). Required for overseas paths.
	TunnelInterface string `json:"tunnel_interface"`
	// OverseasGateway is the next-hop on the overseas exit (the gateway node's
	// tunnel/overlay address), used as the default/overseas route gateway.
	OverseasGateway string `json:"overseas_gateway"`
	// OverseasExitIP is the public IP at the overseas exit (informational /
	// used for NAT to-address when applicable).
	OverseasExitIP string `json:"overseas_exit_ip"`
	// LocalWANGateway is the local ISP gateway used for domestic traffic.
	LocalWANGateway string `json:"local_wan_gateway"`
	// DNS overrides.
	OverseasDNS []string `json:"overseas_dns"`
	DomesticDNS []string `json:"domestic_dns"`
	// OverseasAddressList is the firewall address-list name holding overseas
	// destination prefixes used by smart_split routing.
	OverseasAddressList string `json:"overseas_address_list"`
}

// ErrInvalidProfile indicates a profile that cannot be realized.
type ErrInvalidProfile struct{ Reason string }

func (e ErrInvalidProfile) Error() string { return "invalid accel profile: " + e.Reason }

const overseasMark = "overseas"

// BuildConfig turns a profile into declarative RouterOS config statements that
// the config engine applies. No domain split lists are produced for
// overseas_direct (per requirement: 不做分流，直接使用海外 IP).
func BuildConfig(p Profile) (configengine.State, error) {
	switch p.Mode {
	case ModeOverseasDirect:
		return buildOverseasDirect(p)
	case ModeSmartSplit:
		return buildSmartSplit(p)
	case ModeChinaSplit:
		return configengine.State{}, ErrInvalidProfile{Reason: "china_split 路由表由 chnroutes 脚本下发，请调用 /accel/china-split"}
	case ModeDomesticDirect:
		return buildDomesticDirect(p)
	default:
		return configengine.State{}, ErrInvalidProfile{Reason: "unknown mode " + string(p.Mode)}
	}
}

// 海外运营：全量直连海外出口，不分流。
func buildOverseasDirect(p Profile) (configengine.State, error) {
	if p.TunnelInterface == "" || p.OverseasGateway == "" {
		return configengine.State{}, ErrInvalidProfile{Reason: "overseas_direct 需要 tunnel_interface 与 overseas_gateway"}
	}
	var sts []configengine.Statement

	// Default route: ALL traffic via the overseas exit (no split).
	sts = append(sts, configengine.Statement{
		Path: "/ip/route", Key: "0.0.0.0/0@" + p.OverseasGateway,
		Attributes: map[string]string{
			"dst-address": "0.0.0.0/0",
			"gateway":     p.OverseasGateway,
			"distance":    "1",
			"comment":     "neko-accel: overseas_direct default",
		},
	})
	// NAT masquerade out the overseas tunnel.
	sts = append(sts, configengine.Statement{
		Path: "/ip/firewall/nat", Key: "accel-overseas-masq",
		Attributes: map[string]string{
			"chain":         "srcnat",
			"out-interface": p.TunnelInterface,
			"action":        "masquerade",
			"comment":       "neko-accel: overseas masquerade",
		},
	})
	// DNS directly via overseas resolvers (no domestic split).
	if len(p.OverseasDNS) > 0 {
		sts = append(sts, configengine.Statement{
			Path: "/ip/dns", Key: "settings",
			Attributes: map[string]string{
				"servers":               strings.Join(p.OverseasDNS, ","),
				"allow-remote-requests": "yes",
			},
		})
	}
	return configengine.State{Statements: sts}, nil
}

// 智能分流：国内走本地出口，海外走隧道。
func buildSmartSplit(p Profile) (configengine.State, error) {
	if p.TunnelInterface == "" || p.OverseasGateway == "" {
		return configengine.State{}, ErrInvalidProfile{Reason: "smart_split 需要 tunnel_interface 与 overseas_gateway"}
	}
	list := p.OverseasAddressList
	if list == "" {
		list = "overseas"
	}
	var sts []configengine.Statement

	// Default route stays on the local WAN (domestic direct).
	if p.LocalWANGateway != "" {
		sts = append(sts, configengine.Statement{
			Path: "/ip/route", Key: "0.0.0.0/0@" + p.LocalWANGateway,
			Attributes: map[string]string{
				"dst-address": "0.0.0.0/0", "gateway": p.LocalWANGateway, "distance": "1",
				"comment": "neko-accel: domestic default",
			},
		})
	}
	// Mark connections/routes destined to the overseas address-list.
	sts = append(sts,
		configengine.Statement{
			Path: "/ip/firewall/mangle", Key: "accel-mark-overseas",
			Attributes: map[string]string{
				"chain": "prerouting", "dst-address-list": list,
				"action": "mark-routing", "new-routing-mark": overseasMark,
				"passthrough": "no", "comment": "neko-accel: mark overseas",
			},
		},
		// Route the marked (overseas) traffic via the tunnel.
		configengine.Statement{
			Path: "/ip/route", Key: "0.0.0.0/0@" + overseasMark,
			Attributes: map[string]string{
				"dst-address": "0.0.0.0/0", "gateway": p.OverseasGateway,
				"routing-mark": overseasMark, "distance": "1",
				"comment": "neko-accel: overseas via tunnel",
			},
		},
		configengine.Statement{
			Path: "/ip/firewall/nat", Key: "accel-overseas-masq",
			Attributes: map[string]string{
				"chain": "srcnat", "out-interface": p.TunnelInterface,
				"action": "masquerade", "comment": "neko-accel: overseas masquerade",
			},
		},
	)
	// DNS split: domestic resolvers as base; overseas domains forwarded to
	// overseas resolvers via a forwarder + static FWD (handled by dns package
	// at higher level). Here we set the base domestic servers.
	if len(p.DomesticDNS) > 0 {
		sts = append(sts, configengine.Statement{
			Path: "/ip/dns", Key: "settings",
			Attributes: map[string]string{
				"servers": strings.Join(p.DomesticDNS, ","), "allow-remote-requests": "yes",
			},
		})
	}
	return configengine.State{Statements: sts}, nil
}

// 国内直连：全量走本地出口。
func buildDomesticDirect(p Profile) (configengine.State, error) {
	if p.LocalWANGateway == "" {
		return configengine.State{}, ErrInvalidProfile{Reason: "domestic_direct 需要 local_wan_gateway"}
	}
	sts := []configengine.Statement{
		{
			Path: "/ip/route", Key: "0.0.0.0/0@" + p.LocalWANGateway,
			Attributes: map[string]string{
				"dst-address": "0.0.0.0/0", "gateway": p.LocalWANGateway, "distance": "1",
				"comment": "neko-accel: domestic_direct default",
			},
		},
	}
	if len(p.DomesticDNS) > 0 {
		sts = append(sts, configengine.Statement{
			Path: "/ip/dns", Key: "settings",
			Attributes: map[string]string{
				"servers": strings.Join(p.DomesticDNS, ","), "allow-remote-requests": "yes",
			},
		})
	}
	return configengine.State{Statements: sts}, nil
}

// Describe returns a short human description of a mode for the UI/API.
func Describe(m Mode) string {
	switch m {
	case ModeOverseasDirect:
		return "海外运营：全量流量直连海外出口 IP，不做分流"
	case ModeSmartSplit:
		return "智能分流：国内直连本地出口，海外走 SD-WAN 隧道（地址组）"
	case ModeChinaSplit:
		return "国内外分流：国内 chnroutes 路由表直连本地 WAN，海外 0/1+128/1 走隧道"
	case ModeDomesticDirect:
		return "国内直连：全量走本地出口"
	default:
		return fmt.Sprintf("未知模式 %s", m)
	}
}
