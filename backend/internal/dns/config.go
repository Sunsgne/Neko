package dns

import (
	"strings"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// SplitRule routes specific domains to specific upstream servers (分流), used
// for China acceleration (e.g. domestic domains → carrier DNS, others → public).
type SplitRule struct {
	MatchSuffix string   // e.g. ".cn" or "qq.com"
	Servers     []string // upstream DNS addresses for matched domains
}

// BuildConfig produces RouterOS /ip/dns statements: plain (UDP) servers go
// into `servers`, while a DoH (DNS over HTTPS) server sets `use-doh-server` +
// `verify-doh-cert` (RouterOS supports one DoH endpoint). Optional per-domain
// forwarding rules are appended.
func BuildConfig(primary []Server, splits []SplitRule) configengine.State {
	var sts []configengine.Statement

	addrs := make([]string, 0, len(primary))
	doh := ""
	for _, s := range primary {
		if s.Kind == "doh" {
			if doh == "" {
				doh = s.Address
			}
			continue
		}
		addrs = append(addrs, s.Address)
	}
	settings := map[string]string{"allow-remote-requests": "yes"}
	if len(addrs) > 0 {
		settings["servers"] = strings.Join(addrs, ",")
	}
	if doh != "" {
		settings["use-doh-server"] = doh
		settings["verify-doh-cert"] = "yes"
	}
	sts = append(sts, configengine.Statement{Path: "/ip/dns", Key: "settings", Attributes: settings})

	for _, rule := range splits {
		sts = append(sts, configengine.Statement{
			Path: "/ip/dns/forwarders", Key: "fwd" + rule.MatchSuffix,
			Attributes: map[string]string{
				"name":        "fwd" + rule.MatchSuffix,
				"dns-servers": strings.Join(rule.Servers, ","),
			},
		})
		sts = append(sts, configengine.Statement{
			Path: "/ip/dns/static", Key: "match" + rule.MatchSuffix,
			Attributes: map[string]string{
				"type":            "FWD",
				"match-subdomain": "yes",
				"name":            rule.MatchSuffix,
				"forward-to":      "fwd" + rule.MatchSuffix,
			},
		})
	}

	return configengine.State{Statements: sts}
}
