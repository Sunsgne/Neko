package accel

import (
	"fmt"
	"net/netip"
	"strings"
)

// Route comment tags used so the generated script is fully idempotent: a
// re-run first removes everything it previously installed, then re-adds the
// current set. They are also used by the platform to identify managed routes.
const (
	CommentOverseas = "neko-accel-overseas"
	CommentDomestic = "neko-accel-cn"
)

// OverseasHalves are the two /1 prefixes that, together, cover the entire IPv4
// space while staying strictly more specific than the 0.0.0.0/0 default route.
// Routing them via the SD-WAN tunnel sends ALL traffic overseas except the more
// specific China prefixes (which point at the local WAN gateway).
var OverseasHalves = []string{"0.0.0.0/1", "128.0.0.0/1"}

// ChinaSplitParams parameterizes the route-table based 国内外加速 script.
type ChinaSplitParams struct {
	// WANGateway is the local ISP next-hop for domestic (China) traffic. May be
	// a gateway IP or an interface name (e.g. "ether1" / "pppoe-out1").
	WANGateway string
	// OverseasGateway is the next-hop toward the overseas exit — typically the
	// POP-side address of the SD-WAN tunnel, or the tunnel interface name.
	OverseasGateway string
	// RoutingTable optionally scopes the routes to a named routing table
	// (RouterOS routing-table / VRF). Empty = main.
	RoutingTable string
	// Distance for the generated routes (default 1).
	Distance int
}

// BuildChinaSplitScript returns a RouterOS (.rsc) script implementing
// route-table based domestic/overseas split:
//
//   - China prefixes  -> WANGateway      (国内直连本地出口)
//   - 0.0.0.0/1 + 128.0.0.0/1 -> OverseasGateway (海外经隧道出口)
//
// The script is idempotent: it removes previously installed neko-accel routes
// before adding the current set. count is the total number of routes emitted.
func BuildChinaSplitScript(prefixes []string, p ChinaSplitParams) (script string, count int, err error) {
	if p.WANGateway == "" {
		return "", 0, ErrInvalidProfile{Reason: "china_split 需要 wan_gateway（国内直连出口）"}
	}
	if p.OverseasGateway == "" {
		return "", 0, ErrInvalidProfile{Reason: "china_split 需要 overseas_gateway（海外隧道出口）"}
	}
	dist := p.Distance
	if dist <= 0 {
		dist = 1
	}
	// Validate + de-duplicate China prefixes.
	clean := make([]string, 0, len(prefixes))
	seen := make(map[string]struct{}, len(prefixes))
	for _, c := range prefixes {
		pp, e := netip.ParsePrefix(strings.TrimSpace(c))
		if e != nil || !pp.Addr().Is4() {
			continue
		}
		n := pp.Masked().String()
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		clean = append(clean, n)
	}

	tableAttr := ""
	if p.RoutingTable != "" && p.RoutingTable != "main" {
		tableAttr = fmt.Sprintf(" routing-table=%s", p.RoutingTable)
	}

	var b strings.Builder
	b.WriteString("# neko-sdwan 国内外加速 (chnroutes route-table split)\n")
	b.WriteString(fmt.Sprintf("# 海外 -> %s via %s ; 国内(%d 条) -> %s\n",
		strings.Join(OverseasHalves, ","), p.OverseasGateway, len(clean), p.WANGateway))
	b.WriteString("/ip/route\n")
	// Idempotent cleanup of prior runs.
	b.WriteString(fmt.Sprintf(":foreach r in=[find comment=\"%s\"] do={ remove $r }\n", CommentOverseas))
	b.WriteString(fmt.Sprintf(":foreach r in=[find comment=\"%s\"] do={ remove $r }\n", CommentDomestic))

	// Overseas: the two /1 halves via the tunnel.
	for _, half := range OverseasHalves {
		b.WriteString(fmt.Sprintf("add dst-address=%s gateway=%s distance=%d%s comment=\"%s\"\n",
			half, p.OverseasGateway, dist, tableAttr, CommentOverseas))
		count++
	}
	// Domestic: every China prefix via the local WAN.
	for _, c := range clean {
		b.WriteString(fmt.Sprintf("add dst-address=%s gateway=%s distance=%d%s comment=\"%s\"\n",
			c, p.WANGateway, dist, tableAttr, CommentDomestic))
		count++
	}
	return b.String(), count, nil
}
