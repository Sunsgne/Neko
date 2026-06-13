package configengine

import "strings"

// Risk classifies the blast radius of a configuration change.
type Risk string

const (
	RiskLow      Risk = "low"
	RiskMedium   Risk = "medium"
	RiskHigh     Risk = "high"
	RiskCritical Risk = "critical"
)

func riskRank(r Risk) int {
	switch r {
	case RiskCritical:
		return 3
	case RiskHigh:
		return 2
	case RiskMedium:
		return 1
	default:
		return 0
	}
}

func bumpRisk(r Risk) Risk {
	switch r {
	case RiskLow:
		return RiskMedium
	case RiskMedium:
		return RiskHigh
	default:
		return RiskCritical
	}
}

// RiskOptions tells the classifier which resources protect the management
// channel, so changes touching them are escalated to critical.
type RiskOptions struct {
	ManagementInterfaces []string // e.g. ["ether1"]
	ManagementAddresses  []string // e.g. ["10.0.0.1/24"]
}

// basePathRisk maps RouterOS path prefixes to their inherent base risk.
var basePathRisk = []struct {
	prefix string
	risk   Risk
}{
	{"/ip/firewall", RiskHigh},
	{"/ipv6/firewall", RiskHigh},
	{"/ip/address", RiskHigh},
	{"/ipv6/address", RiskHigh},
	{"/routing", RiskHigh},
	{"/ip/route", RiskHigh},
	{"/interface", RiskMedium},
	{"/system/identity", RiskMedium},
	{"/system/clock", RiskLow},
	{"/system/ntp", RiskLow},
	{"/ip/dns", RiskLow},
	{"/snmp", RiskLow},
}

func basePath(path string) Risk {
	for _, e := range basePathRisk {
		if strings.HasPrefix(path, e.prefix) {
			return e.risk
		}
	}
	return RiskLow
}

// classifyChange returns the risk and a human-readable reason for one change.
func classifyChange(c Change, opts RiskOptions) (Risk, string) {
	risk := basePath(c.Path)
	reason := "base risk for " + c.Path

	// Removals are riskier than additions/updates.
	if c.Type == ChangeRemove {
		risk = bumpRisk(risk)
		reason = "removal of " + c.Path
	}

	// Management-channel protection: any change touching a management
	// interface or address is critical (it could sever access).
	if touchesManagement(c, opts) {
		return RiskCritical, "touches management channel (" + c.Key + ")"
	}

	return risk, reason
}

func touchesManagement(c Change, opts RiskOptions) bool {
	for _, mi := range opts.ManagementInterfaces {
		if mi == "" {
			continue
		}
		if c.Key == mi || attrMatches(c, mi) {
			return true
		}
	}
	for _, ma := range opts.ManagementAddresses {
		if ma == "" {
			continue
		}
		if c.Key == ma || attrMatches(c, ma) {
			return true
		}
	}
	return false
}

func attrMatches(c Change, needle string) bool {
	for _, a := range c.Attrs {
		if a.Old == needle || a.New == needle || strings.Contains(a.Old, needle) || strings.Contains(a.New, needle) {
			return true
		}
	}
	return false
}
