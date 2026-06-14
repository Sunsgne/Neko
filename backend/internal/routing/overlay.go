package routing

import (
	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/store"
)

// TunnelType enumerates supported overlay encapsulations.
type TunnelType string

const (
	TunnelWireGuard TunnelType = "wireguard"
	TunnelIPIP      TunnelType = "ipip"
	TunnelEoIP      TunnelType = "eoip"
	TunnelGRE       TunnelType = "gre"
)

// Tunnel is one site-to-site overlay link.
type Tunnel struct {
	Name       string     `json:"name"`
	Type       TunnelType `json:"type"`
	LocalIP    string     `json:"local_ip"`
	RemoteIP   string     `json:"remote_ip"`
	TunnelAddr string     `json:"tunnel_addr"` // address on the overlay, e.g. 100.64.0.1/30
	PublicKey  string     `json:"public_key,omitempty"`
	ListenPort int        `json:"listen_port,omitempty"`
}

// SelectTunnelType picks the best tunnel encapsulation a device supports.
// WireGuard is preferred (encrypted, modern) when the device runs ROS 7+;
// otherwise it falls back to EoIP (L2) or GRE/IPIP (L3).
func SelectTunnelType(caps *store.CapabilityMatrix, preferL2 bool) TunnelType {
	if caps != nil && caps.SupportsWireGuard {
		return TunnelWireGuard
	}
	if preferL2 {
		return TunnelEoIP
	}
	return TunnelGRE
}

// BuildTunnelState generates declarative config statements for an overlay
// tunnel (RouterOS v7 paths). WireGuard uses /interface/wireguard + peer;
// other types use their respective /interface/<type> path. The tunnel's
// overlay address is added via /ip/address.
func BuildTunnelState(vrf string, t Tunnel) configengine.State {
	var sts []configengine.Statement

	switch t.Type {
	case TunnelWireGuard:
		port := t.ListenPort
		if port == 0 {
			port = 13231
		}
		sts = append(sts,
			configengine.Statement{
				Path: "/interface/wireguard", Key: t.Name,
				Attributes: map[string]string{"name": t.Name, "listen-port": itoa(port)},
			},
			configengine.Statement{
				Path: "/interface/wireguard/peers", Key: t.Name + "-peer",
				Attributes: map[string]string{
					"interface":        t.Name,
					"public-key":       t.PublicKey,
					"endpoint-address": t.RemoteIP,
					"endpoint-port":    itoa(port),
					"allowed-address":  "0.0.0.0/0",
				},
			},
		)
	default:
		sts = append(sts, configengine.Statement{
			Path: "/interface/" + string(t.Type), Key: t.Name,
			Attributes: map[string]string{
				"name":           t.Name,
				"local-address":  t.LocalIP,
				"remote-address": t.RemoteIP,
			},
		})
	}

	if t.TunnelAddr != "" {
		attrs := map[string]string{"address": t.TunnelAddr, "interface": t.Name}
		if vrf != "" {
			attrs["routing-table"] = vrf
		}
		sts = append(sts, configengine.Statement{
			Path: "/ip/address", Key: t.TunnelAddr + "@" + t.Name, Attributes: attrs,
		})
	}

	return configengine.State{Statements: sts}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
