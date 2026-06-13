// Package routing models SD-WAN routing intent (static / OSPF / BGP) and turns
// it into declarative config statements, with validations that prevent route
// leaks between tenants and enforce safe BGP practices.
//
// Addresses requirement #4: static routes, OSPF, BGP (eBGP + iBGP), dual POP,
// Route Reflector, BFD, route filtering/advertisement/aggregation/
// redistribution, and route-leak prevention (per-tenant VRF + communities +
// ingress/egress filtering).
package routing

// StaticRoute is a static routing entry.
type StaticRoute struct {
	DstPrefix string `json:"dst_prefix"`
	Gateway   string `json:"gateway"`
	Distance  int    `json:"distance"`
	Mark      string `json:"routing_mark,omitempty"`
}

// OSPFInterface binds an interface into an OSPF area.
type OSPFInterface struct {
	Interface string `json:"interface"`
	Area      string `json:"area"`
	Cost      int    `json:"cost"`
	Passive   bool   `json:"passive"`
	Auth      string `json:"auth,omitempty"`
}

// OSPFConfig describes the OSPF instance.
type OSPFConfig struct {
	RouterID   string          `json:"router_id"`
	Interfaces []OSPFInterface `json:"interfaces"`
}

// BGPNeighbor is a single BGP peering.
type BGPNeighbor struct {
	Name         string `json:"name"`
	PeerAddress  string `json:"peer_address"`
	LocalAS      uint32 `json:"local_as"`
	PeerAS       uint32 `json:"peer_as"`
	RRClient     bool   `json:"rr_client"`     // this neighbor is a Route Reflector client
	BFD          bool   `json:"bfd"`           // enable BFD for fast failure detection
	ImportFilter string `json:"import_filter"` // route-map applied to received routes
	ExportFilter string `json:"export_filter"` // route-map applied to advertised routes
}

// Kind classifies the peering as iBGP (same AS) or eBGP (different AS).
func (n BGPNeighbor) Kind() string {
	if n.LocalAS != 0 && n.LocalAS == n.PeerAS {
		return "ibgp"
	}
	return "ebgp"
}

// Aggregate is a route summarization entry.
type Aggregate struct {
	Prefix      string `json:"prefix"`
	SummaryOnly bool   `json:"summary_only"`
}

// Redistribution injects routes from one protocol into BGP/OSPF.
type Redistribution struct {
	Source string `json:"source"` // connected | static | ospf | bgp
	Into   string `json:"into"`   // bgp | ospf
	Filter string `json:"filter"` // route-map controlling what is redistributed
}

// Intent is the full routing configuration for one device within a tenant.
type Intent struct {
	TenantID        string           `json:"tenant_id"`
	DeviceID        string           `json:"device_id"`
	VRF             string           `json:"vrf"`              // per-tenant routing table isolation
	TenantCommunity string           `json:"tenant_community"` // e.g. "65000:1001"
	Static          []StaticRoute    `json:"static"`
	OSPF            *OSPFConfig      `json:"ospf,omitempty"`
	BGP             []BGPNeighbor    `json:"bgp"`
	Aggregates      []Aggregate      `json:"aggregates"`
	Redistributions []Redistribution `json:"redistributions"`
}
