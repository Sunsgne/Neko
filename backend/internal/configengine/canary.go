package configengine

// CanaryStage describes one batch of a phased rollout.
type CanaryStage struct {
	Name    string  `json:"name"`
	Percent float64 `json:"percent"` // cumulative percent of fleet, 0..100
}

// DefaultCanaryStages is the standard progression: 1 device, then 5%, 25%, 100%.
var DefaultCanaryStages = []CanaryStage{
	{Name: "canary", Percent: 0}, // special-cased to exactly 1 device
	{Name: "5%", Percent: 5},
	{Name: "25%", Percent: 25},
	{Name: "100%", Percent: 100},
}

// PlanCanaryBatches splits an ordered device list into rollout batches per the
// stages. The first "canary" stage takes exactly one device; subsequent stages
// take up to their cumulative percentage. Each device appears in exactly one
// batch, and the union covers all devices.
func PlanCanaryBatches(devices []string, stages []CanaryStage) [][]string {
	n := len(devices)
	if n == 0 {
		return nil
	}
	if len(stages) == 0 {
		stages = DefaultCanaryStages
	}
	var batches [][]string
	assigned := 0
	for _, st := range stages {
		var target int
		if st.Percent <= 0 {
			target = assigned + 1 // canary: exactly one more
		} else {
			target = int(float64(n) * st.Percent / 100.0)
		}
		if target > n {
			target = n
		}
		if target <= assigned {
			continue
		}
		batches = append(batches, devices[assigned:target])
		assigned = target
	}
	// Ensure full coverage (rounding may leave a remainder).
	if assigned < n {
		batches = append(batches, devices[assigned:n])
	}
	return batches
}
