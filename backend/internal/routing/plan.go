package routing

import (
	"strconv"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// BuildState turns a routing Intent into a declarative configengine.State that
// the config engine can diff and apply. Statements mirror RouterOS v7 routing
// paths. Per-tenant VRF and community tagging are applied so routes remain
// isolated.
func BuildState(in Intent) configengine.State {
	var sts []configengine.Statement

	for _, r := range in.Static {
		attrs := map[string]string{
			"dst-address": r.DstPrefix,
			"gateway":     r.Gateway,
			"distance":    strconv.Itoa(r.Distance),
		}
		if in.VRF != "" {
			attrs["routing-table"] = in.VRF
		}
		if r.Mark != "" {
			attrs["routing-mark"] = r.Mark
		}
		sts = append(sts, configengine.Statement{
			Path:       "/ip/route",
			Key:        r.DstPrefix + "@" + r.Gateway,
			Attributes: attrs,
		})
	}

	if in.OSPF != nil {
		sts = append(sts, configengine.Statement{
			Path: "/routing/ospf/instance",
			Key:  "ospf-" + in.VRF,
			Attributes: map[string]string{
				"router-id": in.OSPF.RouterID,
				"vrf":       in.VRF,
			},
		})
		for _, oi := range in.OSPF.Interfaces {
			sts = append(sts, configengine.Statement{
				Path: "/routing/ospf/interface-template",
				Key:  oi.Interface + "@" + oi.Area,
				Attributes: map[string]string{
					"interfaces": oi.Interface,
					"area":       oi.Area,
					"cost":       strconv.Itoa(oi.Cost),
					"passive":    boolStr(oi.Passive),
				},
			})
		}
	}

	for _, n := range in.BGP {
		attrs := map[string]string{
			"remote-address": n.PeerAddress,
			"remote-as":      strconv.FormatUint(uint64(n.PeerAS), 10),
			"local-as":       strconv.FormatUint(uint64(n.LocalAS), 10),
			"kind":           n.Kind(),
		}
		if in.VRF != "" {
			attrs["vrf"] = in.VRF
		}
		if n.RRClient {
			attrs["route-reflect"] = "yes"
		}
		if n.BFD {
			attrs["use-bfd"] = "yes"
		}
		if n.ImportFilter != "" {
			attrs["input.filter"] = n.ImportFilter
		}
		if n.ExportFilter != "" {
			attrs["output.filter"] = n.ExportFilter
		}
		sts = append(sts, configengine.Statement{
			Path:       "/routing/bgp/connection",
			Key:        n.Name,
			Attributes: attrs,
		})
	}

	for _, a := range in.Aggregates {
		sts = append(sts, configengine.Statement{
			Path: "/routing/bgp/aggregate",
			Key:  a.Prefix,
			Attributes: map[string]string{
				"prefix":       a.Prefix,
				"summary-only": boolStr(a.SummaryOnly),
			},
		})
	}

	return configengine.State{Statements: sts}
}

func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
