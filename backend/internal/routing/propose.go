package routing

import (
	"fmt"
	"strings"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/store"
)

// AccelProposal is a ready-to-deliver CPE→POP acceleration plan: WireGuard
// tunnel negotiation parameters plus the acceleration profile layered on top.
type AccelProposal struct {
	Tunnel            Tunnel        `json:"tunnel"`
	CpePrivateKey     string        `json:"cpe_private_key"`
	CpePublicKey      string        `json:"cpe_public_key"`
	PopPeer           string        `json:"pop_peer"`
	PopEndpoint       string        `json:"pop_endpoint"`
	TunnelInterface   string        `json:"tunnel_interface"`
	CpeOverlay        string        `json:"cpe_overlay"`
	Accel             accel.Profile `json:"accel"`
	PopPublicKeyHint  string        `json:"pop_public_key_hint,omitempty"`
}

// HostFromMgmt strips an optional :port suffix from a management address.
func HostFromMgmt(addr string) string {
	i := strings.LastIndex(addr, ":")
	if i > 0 && strings.Contains(addr, ".") && !strings.Contains(addr[i:], "]") {
		return addr[:i]
	}
	return addr
}

// PopPeerOf derives the POP-side overlay peer IP from the CPE tunnel /30.
// e.g. 100.64.0.2/30 → 100.64.0.1 ; 100.64.88.118/30 → 100.64.88.117
func PopPeerOf(cpeOverlay string) string {
	gw, err := popGatewayFromCPEOverlay(cpeOverlay)
	if err != nil {
		ip := strings.Split(cpeOverlay, "/")[0]
		return ip
	}
	return gw
}

// TunnelNameForPOP builds a stable WireGuard interface name from the POP name.
func TunnelNameForPOP(popName string) string {
	name := "wg-" + popName
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
		return "wg-pop"
	}
	return out
}

// AllocateOverlay picks a deterministic CPE-side /30 in 100.64.0.0/16 for a CPE↔POP pair.
func AllocateOverlay(cpeID, popID string) string {
	_, cpe := AllocateOverlayPair(cpeID, popID)
	return cpe
}

// ProposeAccelToPOP generates WireGuard tunnel + acceleration parameters for
// connecting a CPE to a backbone POP.
func ProposeAccelToPOP(cpe, pop *store.Device, mode accel.Mode, localWAN string, cpeOverlay string) (AccelProposal, error) {
	if pop == nil {
		return AccelProposal{}, fmt.Errorf("pop device required")
	}
	if mode != accel.ModeDomesticDirect && cpe == nil {
		return AccelProposal{}, fmt.Errorf("cpe device required")
	}
	if mode == accel.ModeDomesticDirect {
		if localWAN == "" {
			return AccelProposal{}, fmt.Errorf("domestic_direct 需要 local_wan_gateway")
		}
		return AccelProposal{
			Accel: accel.Profile{
				Mode:            mode,
				LocalWANGateway: localWAN,
				DomesticDNS:     []string{"223.5.5.5", "114.114.114.114"},
			},
		}, nil
	}
	fabric, err := BuildFabricPlan(cpe, pop, mode, localWAN, cpeOverlay, "", "", nil)
	if err != nil {
		return AccelProposal{}, err
	}
	return FabricToProposal(fabric), nil
}

// FabricToProposal maps a bilateral fabric plan to the legacy AccelProposal DTO.
func FabricToProposal(f FabricPlan) AccelProposal {
	l := f.Link
	return AccelProposal{
		Tunnel: Tunnel{
			Name:       l.CPEInterface,
			Type:       TunnelWireGuard,
			RemoteIP:   l.POPEndpointHost,
			TunnelAddr: l.CPEOverlay,
			PrivateKey: l.CPEPrivateKey,
			PublicKey:  l.POPPublicKey,
			ListenPort: l.ListenPort,
		},
		CpePrivateKey:    l.CPEPrivateKey,
		CpePublicKey:     l.CPEPublicKey,
		PopPeer:          l.POPGateway,
		PopEndpoint:      l.POPEndpointHost,
		TunnelInterface:  l.CPEInterface,
		CpeOverlay:       l.CPEOverlay,
		Accel:            f.Accel,
		PopPublicKeyHint: l.POPPublicKeyHint,
	}
}
