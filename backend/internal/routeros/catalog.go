package routeros

import "regexp"

// Section is a single configurable RouterOS REST resource path.
type Section struct {
	Path  string `json:"path"`  // e.g. "/ip/address"
	Label string `json:"label"` // human label, e.g. "Addresses"
	// Singleton marks endpoints that are a single settings object (no list/add),
	// e.g. /ip/dns or /system/identity — edited in place.
	Singleton bool `json:"singleton,omitempty"`
}

// SectionGroup groups sections under a top-level WebFig menu.
type SectionGroup struct {
	Menu     string    `json:"menu"`
	Sections []Section `json:"sections"`
}

// Catalog enumerates the RouterOS configuration surface the platform can drive
// remotely over REST — covering every WebFig menu so all functions support
// remote configuration without logging into the device. The generic CRUD API
// is not limited to this list (any valid path is accepted); the catalog drives
// the console's menu tree and discoverability.
var Catalog = []SectionGroup{
	{Menu: "Interfaces", Sections: []Section{
		{Path: "/interface", Label: "Interface List"},
		{Path: "/interface/ethernet", Label: "Ethernet"},
		{Path: "/interface/list", Label: "Interface Lists"},
		{Path: "/interface/list/member", Label: "List Members"},
		{Path: "/interface/vlan", Label: "VLAN"},
		{Path: "/interface/bonding", Label: "Bonding"},
		{Path: "/interface/eoip", Label: "EoIP Tunnel"},
		{Path: "/interface/gre", Label: "GRE Tunnel"},
		{Path: "/interface/ipip", Label: "IPIP Tunnel"},
		{Path: "/interface/vrrp", Label: "VRRP"},
	}},
	{Menu: "WiFi / Wireless", Sections: []Section{
		{Path: "/interface/wifi", Label: "WiFi (wifiwave2)"},
		{Path: "/interface/wifi/configuration", Label: "WiFi Configurations"},
		{Path: "/interface/wifi/security", Label: "WiFi Security"},
		{Path: "/interface/wifi/channel", Label: "WiFi Channels"},
		{Path: "/interface/wireless", Label: "Wireless"},
		{Path: "/interface/wireless/security-profiles", Label: "Security Profiles"},
		{Path: "/interface/wireless/access-list", Label: "Access List"},
	}},
	{Menu: "WireGuard", Sections: []Section{
		{Path: "/interface/wireguard", Label: "WireGuard Interfaces"},
		{Path: "/interface/wireguard/peers", Label: "Peers"},
	}},
	{Menu: "PPP", Sections: []Section{
		{Path: "/interface/pppoe-client", Label: "PPPoE Client"},
		{Path: "/interface/pppoe-server/server", Label: "PPPoE Server"},
		{Path: "/interface/l2tp-client", Label: "L2TP Client"},
		{Path: "/interface/l2tp-server/server", Label: "L2TP Server", Singleton: true},
		{Path: "/interface/pptp-client", Label: "PPTP Client"},
		{Path: "/interface/sstp-client", Label: "SSTP Client"},
		{Path: "/ppp/secret", Label: "Secrets"},
		{Path: "/ppp/profile", Label: "Profiles"},
	}},
	{Menu: "Bridge", Sections: []Section{
		{Path: "/interface/bridge", Label: "Bridge"},
		{Path: "/interface/bridge/port", Label: "Ports"},
		{Path: "/interface/bridge/vlan", Label: "VLANs"},
		{Path: "/interface/bridge/host", Label: "Hosts"},
	}},
	{Menu: "Switch", Sections: []Section{
		{Path: "/interface/ethernet/switch", Label: "Switch"},
		{Path: "/interface/ethernet/switch/port", Label: "Switch Ports"},
		{Path: "/interface/ethernet/switch/vlan", Label: "Switch VLAN"},
	}},
	{Menu: "Mesh", Sections: []Section{
		{Path: "/interface/mesh", Label: "Mesh"},
	}},
	{Menu: "IP", Sections: []Section{
		{Path: "/ip/address", Label: "Addresses"},
		{Path: "/ip/route", Label: "Routes"},
		{Path: "/ip/pool", Label: "Pools"},
		{Path: "/ip/dhcp-server", Label: "DHCP Server"},
		{Path: "/ip/dhcp-server/network", Label: "DHCP Networks"},
		{Path: "/ip/dhcp-server/lease", Label: "DHCP Leases"},
		{Path: "/ip/dhcp-client", Label: "DHCP Client"},
		{Path: "/ip/dns", Label: "DNS Settings", Singleton: true},
		{Path: "/ip/dns/static", Label: "DNS Static"},
		{Path: "/ip/dns/forwarders", Label: "DNS Forwarders"},
		{Path: "/ip/firewall/filter", Label: "Firewall Filter"},
		{Path: "/ip/firewall/nat", Label: "Firewall NAT"},
		{Path: "/ip/firewall/mangle", Label: "Firewall Mangle"},
		{Path: "/ip/firewall/raw", Label: "Firewall Raw"},
		{Path: "/ip/firewall/address-list", Label: "Address Lists"},
		{Path: "/ip/firewall/connection/tracking", Label: "Connection Tracking", Singleton: true},
		{Path: "/ip/service", Label: "Services"},
		{Path: "/ip/cloud", Label: "Cloud (DDNS)", Singleton: true},
		{Path: "/ip/hotspot", Label: "Hotspot"},
		{Path: "/ip/ipsec/peer", Label: "IPsec Peers"},
		{Path: "/ip/ipsec/policy", Label: "IPsec Policies"},
		{Path: "/ip/ipsec/identity", Label: "IPsec Identities"},
		{Path: "/ip/proxy", Label: "Web Proxy", Singleton: true},
		{Path: "/ip/upnp", Label: "UPnP", Singleton: true},
	}},
	{Menu: "IPv6", Sections: []Section{
		{Path: "/ipv6/address", Label: "Addresses"},
		{Path: "/ipv6/route", Label: "Routes"},
		{Path: "/ipv6/pool", Label: "Pools"},
		{Path: "/ipv6/dhcp-server", Label: "DHCP Server"},
		{Path: "/ipv6/dhcp-client", Label: "DHCP Client"},
		{Path: "/ipv6/nd", Label: "Neighbor Discovery"},
		{Path: "/ipv6/firewall/filter", Label: "Firewall Filter"},
		{Path: "/ipv6/firewall/nat", Label: "Firewall NAT"},
		{Path: "/ipv6/firewall/address-list", Label: "Address Lists"},
		{Path: "/ipv6/settings", Label: "Settings", Singleton: true},
	}},
	{Menu: "MPLS", Sections: []Section{
		{Path: "/mpls", Label: "MPLS Settings", Singleton: true},
		{Path: "/mpls/ldp", Label: "LDP"},
		{Path: "/mpls/interface", Label: "Interfaces"},
	}},
	{Menu: "Routing", Sections: []Section{
		{Path: "/routing/ospf/instance", Label: "OSPF Instances"},
		{Path: "/routing/ospf/area", Label: "OSPF Areas"},
		{Path: "/routing/ospf/interface-template", Label: "OSPF Interfaces"},
		{Path: "/routing/bgp/connection", Label: "BGP Connections"},
		{Path: "/routing/bgp/template", Label: "BGP Templates"},
		{Path: "/routing/filter/rule", Label: "Filter Rules"},
		{Path: "/routing/filter/community-list", Label: "Community Lists"},
		{Path: "/routing/table", Label: "Routing Tables"},
		{Path: "/routing/rule", Label: "Routing Rules"},
		{Path: "/routing/bfd/configuration", Label: "BFD"},
	}},
	{Menu: "System", Sections: []Section{
		{Path: "/system/identity", Label: "Identity", Singleton: true},
		{Path: "/system/clock", Label: "Clock", Singleton: true},
		{Path: "/system/ntp/client", Label: "NTP Client", Singleton: true},
		{Path: "/system/ntp/server", Label: "NTP Server", Singleton: true},
		{Path: "/system/scheduler", Label: "Scheduler"},
		{Path: "/system/script", Label: "Scripts"},
		{Path: "/system/user", Label: "Users"},
		{Path: "/system/user/group", Label: "User Groups"},
		{Path: "/system/logging", Label: "Logging"},
		{Path: "/system/logging/action", Label: "Logging Actions"},
		{Path: "/system/note", Label: "Note", Singleton: true},
	}},
	{Menu: "Queues", Sections: []Section{
		{Path: "/queue/simple", Label: "Simple Queues"},
		{Path: "/queue/tree", Label: "Queue Tree"},
		{Path: "/queue/type", Label: "Queue Types"},
	}},
	{Menu: "Dot1X", Sections: []Section{
		{Path: "/interface/dot1x/client", Label: "Dot1X Client"},
		{Path: "/interface/dot1x/server", Label: "Dot1X Server"},
	}},
	{Menu: "RADIUS", Sections: []Section{
		{Path: "/radius", Label: "RADIUS Servers"},
	}},
	{Menu: "SNMP", Sections: []Section{
		{Path: "/snmp", Label: "SNMP Settings", Singleton: true},
		{Path: "/snmp/community", Label: "Communities"},
	}},
	{Menu: "Tools", Sections: []Section{
		{Path: "/tool/netwatch", Label: "Netwatch"},
		{Path: "/tool/romon", Label: "RoMON", Singleton: true},
		{Path: "/tool/mac-server", Label: "MAC Server", Singleton: true},
		{Path: "/tool/graphing", Label: "Graphing"},
		{Path: "/tool/e-mail", Label: "E-mail", Singleton: true},
		{Path: "/tool/sniffer", Label: "Packet Sniffer", Singleton: true},
	}},
}

// validSectionPath restricts generic CRUD paths to RouterOS-style lowercase
// resource paths (segments of letters/digits/hyphens), e.g. /ip/firewall/nat.
// This blocks traversal and command/query injection while still allowing any
// configuration menu.
var validSectionPath = regexp.MustCompile(`^(/[a-z0-9][a-z0-9-]*)+$`)

// ValidPath reports whether p is an acceptable generic config path.
func ValidPath(p string) bool { return validSectionPath.MatchString(p) }
