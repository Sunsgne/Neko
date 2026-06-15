package routing

import (
	"fmt"
	"hash/fnv"
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
// e.g. 100.64.0.2/30 → 100.64.0.1
func PopPeerOf(cpeOverlay string) string {
	ip := strings.Split(cpeOverlay, "/")[0]
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		parts[3] = "1"
		return strings.Join(parts, ".")
	}
	return ip
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

// AllocateOverlay picks a deterministic /30 in 100.64.0.0/16 for a CPE↔POP pair.
func AllocateOverlay(cpeID, popID string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(cpeID + ":" + popID))
	v := h.Sum32()
	o3 := byte(v%250 + 1)
	o4 := byte((v>>8)%250 + 2)
	if o4 < 2 {
		o4 = 2
	}
	return fmt.Sprintf("100.64.%d.%d/30", o3, o4)
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
	if cpeOverlay == "" && cpe != nil {
		cpeOverlay = AllocateOverlay(cpe.ID, pop.ID)
	}
	popPeer := PopPeerOf(cpeOverlay)
	tn := TunnelNameForPOP(pop.Name)
	endpoint := HostFromMgmt(pop.MgmtAddress)

	priv, pub, err := GenerateWGKeyPair()
	if err != nil {
		return AccelProposal{}, err
	}

	tunnel := Tunnel{
		Name:       tn,
		Type:       TunnelWireGuard,
		RemoteIP:   endpoint,
		TunnelAddr: cpeOverlay,
		PrivateKey: priv,
		ListenPort: 13231,
	}

	profile := accel.Profile{
		Mode:            mode,
		TunnelInterface: tn,
		OverseasGateway: popPeer,
		LocalWANGateway: localWAN,
	}
	switch mode {
	case accel.ModeOverseasDirect:
		profile.OverseasDNS = []string{"8.8.8.8", "1.1.1.1"}
	case accel.ModeSmartSplit:
		if localWAN == "" {
			return AccelProposal{}, fmt.Errorf("smart_split 需要 local_wan_gateway")
		}
		profile.DomesticDNS = []string{"223.5.5.5", "114.114.114.114"}
	case accel.ModeDomesticDirect:
		if localWAN == "" {
			return AccelProposal{}, fmt.Errorf("domestic_direct 需要 local_wan_gateway")
		}
		profile.DomesticDNS = []string{"223.5.5.5", "114.114.114.114"}
	}

	return AccelProposal{
		Tunnel:          tunnel,
		CpePrivateKey:   priv,
		CpePublicKey:    pub,
		PopPeer:         popPeer,
		PopEndpoint:     endpoint,
		TunnelInterface: tn,
		CpeOverlay:      cpeOverlay,
		Accel:           profile,
	}, nil
}
