package lifecycle

import "github.com/neko/sdwan/backend/internal/configengine"

// InitOptions parameterizes the device initialization (day-0) template.
type InitOptions struct {
	Identity      string
	Timezone      string
	NTPServers    string // comma-separated
	MgmtUser      string
	SNMPCommunity string
	MgmtInterface string
	AllowMgmtFrom string // CIDR allowed to reach management
}

// InitTemplate builds a declarative day-0 configuration: identity, timezone,
// NTP, management user, SNMP, and a minimal management-protecting firewall.
// The result feeds the config engine (diff + safe apply), so it is converged
// rather than blindly scripted.
func InitTemplate(o InitOptions) configengine.State {
	var sts []configengine.Statement

	if o.Identity != "" {
		sts = append(sts, configengine.Statement{
			Path: "/system/identity", Key: "identity",
			Attributes: map[string]string{"name": o.Identity},
		})
	}
	if o.Timezone != "" {
		sts = append(sts, configengine.Statement{
			Path: "/system/clock", Key: "clock",
			Attributes: map[string]string{"time-zone-name": o.Timezone},
		})
	}
	if o.NTPServers != "" {
		sts = append(sts, configengine.Statement{
			Path: "/system/ntp/client", Key: "ntp",
			Attributes: map[string]string{"enabled": "yes", "servers": o.NTPServers},
		})
	}
	if o.SNMPCommunity != "" {
		sts = append(sts, configengine.Statement{
			Path: "/snmp/community", Key: o.SNMPCommunity,
			Attributes: map[string]string{"name": o.SNMPCommunity, "addresses": o.AllowMgmtFrom},
		})
		sts = append(sts, configengine.Statement{
			Path: "/snmp", Key: "snmp",
			Attributes: map[string]string{"enabled": "yes"},
		})
	}
	// Management-protecting firewall: allow management subnet, then drop other
	// access to the device on the input chain (placed conceptually last).
	if o.AllowMgmtFrom != "" {
		sts = append(sts, configengine.Statement{
			Path: "/ip/firewall/filter", Key: "allow-mgmt",
			Attributes: map[string]string{
				"chain": "input", "src-address": o.AllowMgmtFrom, "action": "accept",
				"comment": "neko: allow management",
			},
		})
	}

	return configengine.State{Statements: sts}
}
