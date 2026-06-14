package monitoring

import (
	"testing"

	"github.com/neko/sdwan/backend/internal/store"
)

func find(checks []Check, code string) (Check, bool) {
	for _, c := range checks {
		if c.Code == code {
			return c, true
		}
	}
	return Check{}, false
}

func TestUnenrolledNoChecks(t *testing.T) {
	if len(Evaluate(&store.Device{Enrolled: false}, DefaultThresholds())) != 0 {
		t.Error("unenrolled device should produce no checks")
	}
}

func TestOfflineFires(t *testing.T) {
	d := &store.Device{Enrolled: true, Status: &store.DeviceStatus{Online: false, LastError: "timeout"}}
	c, _ := find(Evaluate(d, DefaultThresholds()), "device_offline")
	if !c.Active || c.Severity != "critical" {
		t.Errorf("offline device should fire critical, got %+v", c)
	}
}

func TestOnlineHealthyResolves(t *testing.T) {
	d := &store.Device{Enrolled: true, Status: &store.DeviceStatus{Online: true, CPULoadPercent: 10, TotalMemoryBytes: 1000, FreeMemoryBytes: 800}}
	checks := Evaluate(d, DefaultThresholds())
	for _, c := range checks {
		if c.Active {
			t.Errorf("healthy device should have no active checks, got %s active", c.Code)
		}
	}
}

func TestHighCPUAndMemFire(t *testing.T) {
	d := &store.Device{Enrolled: true, Status: &store.DeviceStatus{
		Online: true, CPULoadPercent: 95, TotalMemoryBytes: 1000, FreeMemoryBytes: 50, // 95% used
	}}
	checks := Evaluate(d, DefaultThresholds())
	cpu, _ := find(checks, "cpu_high")
	mem, _ := find(checks, "mem_high")
	if !cpu.Active {
		t.Error("high CPU should fire")
	}
	if !mem.Active {
		t.Error("high mem should fire")
	}
}
