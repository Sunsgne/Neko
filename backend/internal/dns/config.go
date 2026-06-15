package dns

import (
	"net"
	"net/url"
	"strings"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// SplitRule routes specific domains to specific upstream servers (分流), used
// for China acceleration (e.g. domestic domains → carrier DNS, others → public).
type SplitRule struct {
	MatchSuffix string   // e.g. ".cn" or "qq.com"
	Servers     []string // upstream DNS addresses for matched domains
}

// Options tunes how the /ip/dns settings are generated to match real RouterOS
// fields (see WebFig: IP → DNS → Settings).
type Options struct {
	// VerifyDoHCert overrides DoH certificate verification. nil = auto-detect:
	// an IP-literal DoH endpoint (e.g. https://202.101.51.194:9291/dns-query)
	// cannot present a verifiable certificate, so verification is disabled;
	// a hostname endpoint (e.g. https://dns.alidns.com/dns-query) enables it.
	VerifyDoHCert *bool
	// DisableAllowRemote turns off allow-remote-requests (default is on so the
	// router serves LAN clients).
	DisableAllowRemote bool
}

// BuildConfig produces RouterOS /ip/dns statements matching the device's real
// parameter names: plain (UDP) servers go into `servers`, while a DoH (DNS over
// HTTPS) endpoint sets `use-doh-server` + `verify-doh-cert` (RouterOS supports
// one DoH endpoint). Optional per-domain forwarding rules are appended.
func BuildConfig(primary []Server, splits []SplitRule, opts Options) configengine.State {
	var sts []configengine.Statement

	addrs := make([]string, 0, len(primary))
	doh := ""
	for _, s := range primary {
		if s.Kind == "doh" {
			if doh == "" {
				doh = strings.TrimSpace(s.Address)
			}
			continue
		}
		addrs = append(addrs, s.Address)
	}

	settings := map[string]string{}
	if !opts.DisableAllowRemote {
		settings["allow-remote-requests"] = "yes"
	}
	if len(addrs) > 0 {
		settings["servers"] = strings.Join(addrs, ",")
	}
	if doh != "" {
		settings["use-doh-server"] = doh
		settings["verify-doh-cert"] = boolToYesNo(verifyDoHCert(doh, opts.VerifyDoHCert))
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

// verifyDoHCert decides whether to enable DoH certificate verification. An
// explicit override wins; otherwise it is enabled only for hostname endpoints
// (IP-literal endpoints cannot present a verifiable certificate).
func verifyDoHCert(dohURL string, override *bool) bool {
	if override != nil {
		return *override
	}
	return !dohHostIsIP(dohURL)
}

// dohHostIsIP reports whether the DoH endpoint's host is an IP literal.
func dohHostIsIP(dohURL string) bool {
	u, err := url.Parse(dohURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		// Bare "host:port" or "host" without scheme.
		host = dohURL
		if h, _, err := net.SplitHostPort(dohURL); err == nil {
			host = h
		}
	}
	return net.ParseIP(strings.Trim(host, "[]")) != nil
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
