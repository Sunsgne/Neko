package linkqos

import "math"

// Aggregate turns a set of per-probe RTT samples (ms) and the number of probes
// sent into latency/jitter/loss metrics:
//   - latency = mean RTT of received replies
//   - jitter  = mean absolute successive RTT difference (RFC 3550 style)
//   - loss    = (sent - received) / sent
func Aggregate(rttsMs []float64, sent int) Metrics {
	recv := len(rttsMs)
	var m Metrics
	if sent > 0 {
		m.Loss = float64(sent-recv) / float64(sent)
		if m.Loss < 0 {
			m.Loss = 0
		}
	}
	if recv == 0 {
		return m
	}
	var sum float64
	for _, r := range rttsMs {
		sum += r
	}
	m.LatencyMs = sum / float64(recv)
	if recv > 1 {
		var jsum float64
		for i := 1; i < recv; i++ {
			jsum += math.Abs(rttsMs[i] - rttsMs[i-1])
		}
		m.JitterMs = jsum / float64(recv-1)
	}
	return m
}

// Status classifies a link's operational state from its loss and score:
//   - down     when fully unreachable (100% loss) or the score is very low
//   - degraded when the score is below the healthy threshold
//   - up       otherwise
func Status(score, loss float64) string {
	if loss >= 1.0 || score < 40 {
		return "down"
	}
	if score < 75 {
		return "degraded"
	}
	return "up"
}
