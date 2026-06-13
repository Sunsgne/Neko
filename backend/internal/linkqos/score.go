// Package linkqos computes link quality scores from latency/loss/jitter and
// makes stable failover/failback decisions with flap damping.
//
// Addresses requirement #7: latency/loss/jitter monitoring, link scoring,
// primary/backup with automatic recovery, and anti-flap (oscillation) control.
package linkqos

// Metrics is a single link-quality measurement.
type Metrics struct {
	LatencyMs float64 `json:"latency_ms"`
	JitterMs  float64 `json:"jitter_ms"`
	Loss      float64 `json:"loss"` // packet loss ratio 0..1
}

// ScoreConfig holds the reference ceilings and weights used to score a link.
// A metric at or beyond its ceiling contributes 0 to its sub-score; at 0 it
// contributes its full weight. The weights should sum to 1.
type ScoreConfig struct {
	LatencyCeilingMs float64
	JitterCeilingMs  float64
	LossCeiling      float64 // e.g. 0.10 means 10% loss => 0
	WLatency         float64
	WJitter          float64
	WLoss            float64
}

// DefaultScoreConfig returns sensible defaults tuned for WAN links.
func DefaultScoreConfig() ScoreConfig {
	return ScoreConfig{
		LatencyCeilingMs: 300,
		JitterCeilingMs:  100,
		LossCeiling:      0.10,
		WLatency:         0.4,
		WJitter:          0.2,
		WLoss:            0.4,
	}
}

// Score returns a 0..100 quality score (higher is better).
func Score(m Metrics, cfg ScoreConfig) float64 {
	lat := sub(m.LatencyMs, cfg.LatencyCeilingMs)
	jit := sub(m.JitterMs, cfg.JitterCeilingMs)
	loss := sub(m.Loss, cfg.LossCeiling)

	wSum := cfg.WLatency + cfg.WJitter + cfg.WLoss
	if wSum == 0 {
		return 0
	}
	score := (lat*cfg.WLatency + jit*cfg.WJitter + loss*cfg.WLoss) / wSum
	return clamp(score*100, 0, 100)
}

// sub returns a normalized 1..0 sub-score: 1 when value<=0, 0 when value>=ceiling.
func sub(value, ceiling float64) float64 {
	if ceiling <= 0 {
		if value <= 0 {
			return 1
		}
		return 0
	}
	return clamp(1-value/ceiling, 0, 1)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
