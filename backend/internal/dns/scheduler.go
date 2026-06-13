// Package dns implements DNS server pool management and region/ISP-aware
// scheduling for China network acceleration (requirement #6): given a client's
// region and ISP, pick the DNS servers most likely to resolve quickly and to
// return CDN answers close to the client.
package dns

import "sort"

// ISP identifies a Chinese carrier (or a public/anycast resolver).
type ISP string

const (
	ISPTelecom ISP = "telecom" // 电信
	ISPUnicom  ISP = "unicom"  // 联通
	ISPMobile  ISP = "mobile"  // 移动
	ISPEdu     ISP = "edu"     // 教育网
	ISPPublic  ISP = "public"  // 公共/anycast（如 114/223/公共 DNS）
	ISPUnknown ISP = ""
)

// Server is a configured upstream DNS server in the pool.
type Server struct {
	ID          string `json:"id"`
	Address     string `json:"address"`
	Region      string `json:"region"` // province/region code, e.g. "cn-east", "shanghai"
	ISP         ISP    `json:"isp"`
	SupportsECS bool   `json:"supports_ecs"`
	Healthy     bool   `json:"healthy"`
	LatencyMs   int    `json:"latency_ms"`
}

// ClientContext describes the resolving client's network location.
type ClientContext struct {
	Region string
	ISP    ISP
}

// scored pairs a server with its computed selection score.
type scored struct {
	srv   Server
	score float64
}

// Select returns up to limit healthy servers ordered best-first for the given
// client context. Selection favors same-ISP, same-region, ECS-capable, and
// low-latency servers, with public/anycast servers as a universal fallback.
func Select(pool []Server, client ClientContext, limit int) []Server {
	if limit <= 0 {
		limit = 3
	}
	ranked := make([]scored, 0, len(pool))
	for _, s := range pool {
		if !s.Healthy {
			continue
		}
		ranked = append(ranked, scored{srv: s, score: scoreServer(s, client)})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].srv.LatencyMs != ranked[j].srv.LatencyMs {
			return ranked[i].srv.LatencyMs < ranked[j].srv.LatencyMs
		}
		return ranked[i].srv.ID < ranked[j].srv.ID
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]Server, len(ranked))
	for i, r := range ranked {
		out[i] = r.srv
	}
	return out
}

func scoreServer(s Server, c ClientContext) float64 {
	var score float64

	switch {
	case c.ISP != ISPUnknown && s.ISP == c.ISP:
		score += 100 // same carrier: best CDN locality
	case s.ISP == ISPPublic || s.ISP == ISPUnknown:
		score += 30 // public/anycast: works for everyone
	}

	if c.Region != "" && s.Region == c.Region {
		score += 40
	}

	if c.Region != "" && s.SupportsECS {
		score += 10 // ECS helps return geo-correct CDN answers
	}

	// Latency penalty: each ms shaves a little off the score.
	score -= float64(s.LatencyMs) * 0.1

	return score
}
