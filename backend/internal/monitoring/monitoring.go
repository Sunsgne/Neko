// Package monitoring evaluates device health into alert checks. The worker
// applies the result against the persisted alert store (fire/resolve with
// de-duplication), turning raw poll data into actionable, deduplicated alerts.
package monitoring

import (
	"fmt"

	"github.com/neko/sdwan/backend/internal/store"
)

// Thresholds configures the health rules.
type Thresholds struct {
	CPUPercent int // warn above this CPU load
	MemPercent int // warn above this memory usage
}

// DefaultThresholds returns production-reasonable defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{CPUPercent: 85, MemPercent: 90}
}

// Check is one evaluated health condition for a device.
type Check struct {
	Code     string
	Severity string
	Title    string
	Detail   string
	Active   bool // true => should be firing; false => should be resolved
}

// Evaluate derives alert checks from a device's current state. Only enrolled
// devices are evaluated (others have no telemetry). The worker fires Active
// checks and resolves inactive ones.
func Evaluate(d *store.Device, th Thresholds) []Check {
	if !d.Enrolled {
		return nil
	}
	checks := []Check{}
	st := d.Status

	offline := st == nil || !st.Online
	checks = append(checks, Check{
		Code: "device_offline", Severity: "critical",
		Title:  fmt.Sprintf("设备离线：%s", d.Name),
		Detail: detailOrDefault(st, "无法连接设备", ""),
		Active: offline,
	})

	// Resource checks only meaningful when online.
	cpuActive := false
	cpuDetail := ""
	memActive := false
	memDetail := ""
	if st != nil && st.Online {
		if st.CPULoadPercent > th.CPUPercent {
			cpuActive = true
			cpuDetail = fmt.Sprintf("CPU 负载 %d%% 超过阈值 %d%%", st.CPULoadPercent, th.CPUPercent)
		}
		if st.TotalMemoryBytes > 0 {
			used := int(float64(st.TotalMemoryBytes-st.FreeMemoryBytes) / float64(st.TotalMemoryBytes) * 100)
			if used > th.MemPercent {
				memActive = true
				memDetail = fmt.Sprintf("内存使用 %d%% 超过阈值 %d%%", used, th.MemPercent)
			}
		}
	}
	checks = append(checks,
		Check{Code: "cpu_high", Severity: "warning", Title: fmt.Sprintf("CPU 偏高：%s", d.Name), Detail: cpuDetail, Active: cpuActive},
		Check{Code: "mem_high", Severity: "warning", Title: fmt.Sprintf("内存偏高：%s", d.Name), Detail: memDetail, Active: memActive},
	)
	return checks
}

func detailOrDefault(st *store.DeviceStatus, def, _ string) string {
	if st != nil && st.LastError != "" {
		return st.LastError
	}
	return def
}
