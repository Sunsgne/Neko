package snmp

import (
	"testing"
	"time"
)

func TestOctetRateSimple(t *testing.T) {
	// 1,000,000 octets over 10s = 100,000 octets/s = 800,000 bps.
	r, ok := OctetRate(0, 1_000_000, 10*time.Second, Counter64)
	if !ok {
		t.Fatal("expected ok")
	}
	if r != 100_000 {
		t.Errorf("rate = %.0f, want 100000", r)
	}
	if bps := BitsPerSecond(r); bps != 800_000 {
		t.Errorf("bps = %.0f, want 800000", bps)
	}
}

func TestOctetRate32Wrap(t *testing.T) {
	prev := max32 - 100 // close to wrap
	cur := uint64(50)   // wrapped past zero
	r, ok := OctetRate(prev, cur, 1*time.Second, Counter32)
	if !ok {
		t.Fatal("expected ok")
	}
	// delta = 100 + 50 + 1 = 151
	if r != 151 {
		t.Errorf("rate = %.0f, want 151", r)
	}
}

func TestOctetRate64ResetNotOK(t *testing.T) {
	if _, ok := OctetRate(1_000_000, 5, time.Second, Counter64); ok {
		t.Error("64-bit decrease should be treated as reset (not ok)")
	}
}

func TestOctetRateZeroInterval(t *testing.T) {
	if _, ok := OctetRate(0, 100, 0, Counter64); ok {
		t.Error("zero interval should not be ok")
	}
}

func TestUtilization(t *testing.T) {
	if u := Utilization(500e6, 1e9); u != 0.5 {
		t.Errorf("util = %.2f, want 0.5", u)
	}
	if u := Utilization(2e9, 1e9); u != 1 {
		t.Errorf("util should clamp to 1, got %.2f", u)
	}
	if u := Utilization(100, 0); u != 0 {
		t.Errorf("unknown capacity should be 0, got %.2f", u)
	}
}
