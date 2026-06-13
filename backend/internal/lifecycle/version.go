// Package lifecycle handles RouterOS/RouterBOOT version validation, upgrade
// planning, and device initialization templates (requirements #6/#9).
package lifecycle

import (
	"strconv"
	"strings"
)

// ParseVersion extracts numeric components from a RouterOS version string such
// as "7.14.3 (stable)" -> [7,14,3]. Missing components are treated as 0.
func ParseVersion(v string) []int {
	v = strings.TrimSpace(v)
	if i := strings.IndexByte(v, ' '); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return []int{0}
	}
	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			n = 0
		}
		out[i] = n
	}
	return out
}

// CompareVersions returns -1 if a<b, 0 if equal, 1 if a>b.
func CompareVersions(a, b string) int {
	pa, pb := ParseVersion(a), ParseVersion(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

// NeedsUpgrade reports whether current is older than target.
func NeedsUpgrade(current, target string) bool {
	return CompareVersions(current, target) < 0
}
