package routing

import "fmt"

// Severity of a validation issue.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Issue is a single validation finding.
type Issue struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
}

// Validate enforces safe-routing and leak-prevention rules. Any Error-severity
// issue means the intent must NOT be applied as-is.
//
// Rules:
//   - Per-tenant isolation: a VRF and a tenant community must be set so routes
//     cannot leak across tenants.
//   - eBGP neighbors MUST have both import and export filters (no implicit
//     transit / leak). This is the primary leak-prevention guard.
//   - Redistribution MUST be filtered to avoid injecting unintended routes.
//   - iBGP without a Route Reflector among many peers is flagged (scaling).
//   - BFD is recommended on eBGP for fast convergence.
func Validate(in Intent) []Issue {
	var issues []Issue

	if in.VRF == "" {
		issues = append(issues, Issue{Error, "missing_vrf",
			"tenant VRF is required to isolate routing tables and prevent leaks"})
	}
	if in.TenantCommunity == "" {
		issues = append(issues, Issue{Error, "missing_community",
			"tenant community is required to tag and filter routes for leak prevention"})
	}

	var ibgp int
	for _, n := range in.BGP {
		if n.PeerAS == 0 {
			issues = append(issues, Issue{Error, "missing_peer_as",
				fmt.Sprintf("neighbor %s: peer_as is required", n.Name)})
		}
		if n.Kind() == "ebgp" {
			if n.ImportFilter == "" || n.ExportFilter == "" {
				issues = append(issues, Issue{Error, "ebgp_unfiltered",
					fmt.Sprintf("eBGP neighbor %s must define both import and export filters to prevent route leaks", n.Name)})
			}
			if !n.BFD {
				issues = append(issues, Issue{Warning, "ebgp_no_bfd",
					fmt.Sprintf("eBGP neighbor %s should enable BFD for fast failure detection", n.Name)})
			}
		} else {
			ibgp++
		}
	}

	// iBGP scaling: many iBGP peers without any RR client suggests a full mesh.
	if ibgp >= 3 {
		hasRR := false
		for _, n := range in.BGP {
			if n.Kind() == "ibgp" && n.RRClient {
				hasRR = true
				break
			}
		}
		if !hasRR {
			issues = append(issues, Issue{Warning, "ibgp_full_mesh",
				"3+ iBGP peers without a Route Reflector; consider RR to avoid full mesh"})
		}
	}

	for _, r := range in.Redistributions {
		if r.Filter == "" {
			issues = append(issues, Issue{Error, "redistribute_unfiltered",
				fmt.Sprintf("redistribution %s->%s must define a filter to prevent leaking unintended routes", r.Source, r.Into)})
		}
	}

	return issues
}

// HasErrors reports whether any issue is Error severity.
func HasErrors(issues []Issue) bool {
	for _, i := range issues {
		if i.Severity == Error {
			return true
		}
	}
	return false
}
