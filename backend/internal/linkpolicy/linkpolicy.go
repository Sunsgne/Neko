// Package linkpolicy turns a site's WAN uplink selection into RouterOS routing
// configuration: active/backup failover or ECMP load-balancing, with gateway
// liveness checks so failover/recovery happens automatically on the device.
//
// This is the "链路选择" half of the SD-WAN orchestration flow; combined with
// the config engine it delivers real route push (路由下发).
package linkpolicy

import (
	"fmt"
	"sort"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// Strategy selects how multiple uplinks are used.
type Strategy string

const (
	// Failover 主备：按优先级使用，主断自动切备，主恢复回切。
	Failover Strategy = "failover"
	// LoadBalance 负载均衡：多条上行 ECMP 并行（按权重）。
	LoadBalance Strategy = "loadbalance"
)

// Uplink describes one WAN uplink at a site.
type Uplink struct {
	Name      string `json:"name"`
	Gateway   string `json:"gateway"`   // next-hop gateway IP
	Interface string `json:"interface"` // egress interface (for NAT)
	Priority  int    `json:"priority"`  // lower = preferred (failover order)
	Weight    int    `json:"weight"`    // ECMP weight (loadbalance)
	// CheckTarget is pinged to detect uplink health (check-gateway). Defaults
	// to the gateway when empty.
	CheckTarget string `json:"check_target"`
}

// Policy is the full link-selection intent for a site/device.
type Policy struct {
	Strategy Strategy `json:"strategy"`
	Uplinks  []Uplink `json:"uplinks"`
}

// ErrInvalid indicates a policy that cannot be realized.
type ErrInvalid struct{ Reason string }

func (e ErrInvalid) Error() string { return "invalid link policy: " + e.Reason }

// BuildConfig generates RouterOS /ip/route (+ NAT masquerade) statements that
// implement the link selection.
func BuildConfig(p Policy) (configengine.State, error) {
	if len(p.Uplinks) == 0 {
		return configengine.State{}, ErrInvalid{Reason: "至少需要一条上行链路"}
	}
	for _, u := range p.Uplinks {
		if u.Gateway == "" {
			return configengine.State{}, ErrInvalid{Reason: "上行 " + u.Name + " 缺少网关"}
		}
	}
	switch p.Strategy {
	case Failover, "":
		return buildFailover(p.Uplinks), nil
	case LoadBalance:
		return buildLoadBalance(p.Uplinks), nil
	default:
		return configengine.State{}, ErrInvalid{Reason: "未知策略 " + string(p.Strategy)}
	}
}

// buildFailover: each uplink gets a default route whose distance follows its
// priority (preferred = lowest distance). check-gateway=ping removes a dead
// route automatically, so traffic fails over to the next distance and recovers
// (回切) when the preferred gateway is healthy again.
func buildFailover(uplinks []Uplink) configengine.State {
	sorted := append([]Uplink(nil), uplinks...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Priority < sorted[j].Priority })

	var sts []configengine.Statement
	for i, u := range sorted {
		distance := i + 1 // 1,2,3...
		sts = append(sts, configengine.Statement{
			Path: "/ip/route", Key: "0.0.0.0/0@" + u.Gateway,
			Attributes: map[string]string{
				"dst-address":   "0.0.0.0/0",
				"gateway":       u.Gateway,
				"distance":      itoa(distance),
				"check-gateway": "ping",
				"comment":       fmt.Sprintf("neko-link: %s (failover #%d)", u.Name, distance),
			},
		})
		if u.Interface != "" {
			sts = append(sts, masq(u))
		}
	}
	return configengine.State{Statements: sts}
}

// buildLoadBalance: a single ECMP default route with all gateways (weighted by
// repetition) so RouterOS distributes flows across uplinks.
func buildLoadBalance(uplinks []Uplink) configengine.State {
	gws := ""
	for _, u := range uplinks {
		w := u.Weight
		if w <= 0 {
			w = 1
		}
		for k := 0; k < w; k++ {
			if gws != "" {
				gws += ","
			}
			gws += u.Gateway
		}
	}
	sts := []configengine.Statement{{
		Path: "/ip/route", Key: "0.0.0.0/0@ecmp",
		Attributes: map[string]string{
			"dst-address":   "0.0.0.0/0",
			"gateway":       gws,
			"distance":      "1",
			"check-gateway": "ping",
			"comment":       "neko-link: ECMP load-balance",
		},
	}}
	for _, u := range uplinks {
		if u.Interface != "" {
			sts = append(sts, masq(u))
		}
	}
	return configengine.State{Statements: sts}
}

func masq(u Uplink) configengine.Statement {
	return configengine.Statement{
		Path: "/ip/firewall/nat", Key: "link-masq-" + u.Name,
		Attributes: map[string]string{
			"chain": "srcnat", "out-interface": u.Interface,
			"action": "masquerade", "comment": "neko-link: masquerade " + u.Name,
		},
	}
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
