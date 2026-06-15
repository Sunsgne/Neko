package qos

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// Rule is one RouterOS simple-queue entry.
type Rule struct {
	Name       string `json:"name"`
	Target     string `json:"target"`      // IP/CIDR, /32 host, or interface name
	MaxLimit   string `json:"max_limit"`   // tx/rx e.g. 10M/10M
	LimitAt    string `json:"limit_at,omitempty"`
	BurstLimit string `json:"burst_limit,omitempty"`
	Priority   int    `json:"priority,omitempty"` // 1 (highest) .. 8
	Comment    string `json:"comment,omitempty"`
}

var rateToken = regexp.MustCompile(`(?i)^\d+(\.\d+)?[kKmMgG]?$`)

// NormalizeRate turns "10" into "10M", leaves "10M/5M" pairs intact.
func NormalizeRate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		return NormalizeRate(parts[0]) + "/" + NormalizeRate(parts[1])
	}
	if rateToken.MatchString(s) {
		last := s[len(s)-1]
		if last >= '0' && last <= '9' {
			return s + "M"
		}
		return s
	}
	return s
}

// ValidateRule checks required fields and rate syntax.
func ValidateRule(r Rule) error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("策略名称不能为空")
	}
	if strings.TrimSpace(r.Target) == "" {
		return fmt.Errorf("%s: 目标 (target) 不能为空", r.Name)
	}
	if strings.TrimSpace(r.MaxLimit) == "" {
		return fmt.Errorf("%s: 最大速率 (max-limit) 不能为空", r.Name)
	}
	ml := NormalizeRate(r.MaxLimit)
	if !strings.Contains(ml, "/") {
		ml = ml + "/" + ml
	}
	for _, part := range strings.Split(ml, "/") {
		if !rateToken.MatchString(part) {
			return fmt.Errorf("%s: 无效速率 %q", r.Name, part)
		}
	}
	if r.Priority != 0 && (r.Priority < 1 || r.Priority > 8) {
		return fmt.Errorf("%s: priority 须在 1..8", r.Name)
	}
	return nil
}

// BuildSimpleQueues turns rules into /queue/simple statements.
func BuildSimpleQueues(rules []Rule) (configengine.State, error) {
	var sts []configengine.Statement
	seen := map[string]struct{}{}
	for _, r := range rules {
		if err := ValidateRule(r); err != nil {
			return configengine.State{}, err
		}
		key := strings.TrimSpace(r.Name)
		if _, ok := seen[key]; ok {
			return configengine.State{}, fmt.Errorf("duplicate queue name %q", key)
		}
		seen[key] = struct{}{}

		maxLimit := NormalizeRate(r.MaxLimit)
		if !strings.Contains(maxLimit, "/") {
			maxLimit = maxLimit + "/" + maxLimit
		}
		attrs := map[string]string{
			"name":      key,
			"target":    strings.TrimSpace(r.Target),
			"max-limit": maxLimit,
			"comment":   firstNonEmpty(r.Comment, "neko-qos"),
		}
		if la := strings.TrimSpace(r.LimitAt); la != "" {
			attrs["limit-at"] = NormalizeRate(la)
			if !strings.Contains(attrs["limit-at"], "/") {
				attrs["limit-at"] = attrs["limit-at"] + "/" + attrs["limit-at"]
			}
		}
		if bl := strings.TrimSpace(r.BurstLimit); bl != "" {
			attrs["burst-limit"] = NormalizeRate(bl)
			if !strings.Contains(attrs["burst-limit"], "/") {
				attrs["burst-limit"] = attrs["burst-limit"] + "/" + attrs["burst-limit"]
			}
		}
		if r.Priority > 0 {
			attrs["priority"] = strconv.Itoa(r.Priority)
		}
		sts = append(sts, configengine.Statement{
			Path:       "/queue/simple",
			Key:        key,
			Attributes: attrs,
		})
	}
	return configengine.State{Statements: sts}, nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
}
