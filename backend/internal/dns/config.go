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

// BuildConfig produces RouterOS /ip/dns statements: a primary server set plus
// optional per-domain forwarding rules (/ip/dns/forwarders + static matchers).
func BuildConfig(primary []Server, splits []SplitRule) configengine.State {
	var sts []configengine.Statement

	addrs := make([]string, 0, len(primary))
	for _, s := range primary {
		addrs = append(addrs, s.Address)
	}
	sts = append(sts, configengine.Statement{
		Path: "/ip/dns", Key: "settings",
		Attributes: map[string]string{
			"servers":               strings.Join(addrs, ","),
			"allow-remote-requests": "yes",
		},
	})

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
