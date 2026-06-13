// Package snmp provides SNMP metric math used by the poller: converting raw
// interface octet counters into rates (bps) with counter-wrap handling, and
// deriving interface utilization. The polling transport (gosnmp) is wired in
// the worker; this package holds the pure, testable computation.
package snmp

import "time"

// CounterWidth is the bit width of an SNMP counter.
type CounterWidth int

const (
	Counter32 CounterWidth = 32 // ifInOctets / ifOutOctets
	Counter64 CounterWidth = 64 // ifHCInOctets / ifHCOutOctets (preferred)
)

const (
	max32 = uint64(1)<<32 - 1
	max64 = ^uint64(0)
)

// OctetRate returns the octets-per-second between two counter samples,
// handling 32-bit counter wrap. For 64-bit counters a decrease is treated as a
// device/counter reset and reported as not-ok (callers should skip the point).
func OctetRate(prev, cur uint64, dt time.Duration, width CounterWidth) (float64, bool) {
	if dt <= 0 {
		return 0, false
	}
	var delta uint64
	if cur >= prev {
		delta = cur - prev
	} else {
		switch width {
		case Counter32:
			delta = (max32 - prev) + cur + 1
		default:
			// 64-bit decrease ⇒ assume reset; cannot compute a valid rate.
			return 0, false
		}
	}
	return float64(delta) / dt.Seconds(), true
}

// BitsPerSecond converts an octet rate to bits per second.
func BitsPerSecond(octetsPerSec float64) float64 { return octetsPerSec * 8 }

// Utilization returns the fraction (0..1) of capacity used given a bps rate and
// the interface capacity in bps. Returns 0 if capacity is unknown.
func Utilization(bps, capacityBps float64) float64 {
	if capacityBps <= 0 {
		return 0
	}
	u := bps / capacityBps
	if u < 0 {
		return 0
	}
	if u > 1 {
		return 1
	}
	return u
}
