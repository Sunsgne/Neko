package lifecycle

// StepKind enumerates the kinds of upgrade steps.
type StepKind string

const (
	StepDownload          StepKind = "download"
	StepVerifyChecksum    StepKind = "verify_checksum"
	StepUpgradeRouterOS   StepKind = "upgrade_routeros"
	StepReboot            StepKind = "reboot"
	StepVerifyVersion     StepKind = "verify_version"
	StepUpgradeRouterBOOT StepKind = "upgrade_routerboot"
	StepHealthCheck       StepKind = "health_check"
)

// Step is one ordered action in an upgrade plan.
type Step struct {
	Kind   StepKind `json:"kind"`
	Detail string   `json:"detail"`
}

// UpgradeInput describes the current state and desired target.
type UpgradeInput struct {
	CurrentVersion string
	TargetVersion  string
	// RouterBOOT firmware: current installed vs. the version bundled with the
	// (post-upgrade) RouterOS. If they differ, a RouterBOOT upgrade + second
	// reboot is appended.
	RouterBootCurrent string
	RouterBootBundled string
	// IsRouterBOARD indicates RouterBOOT applies (CHR/x86 have no RouterBOOT).
	IsRouterBOARD bool
}

// UpgradePlan is the result of planning an upgrade.
type UpgradePlan struct {
	NeedsRouterOS   bool   `json:"needs_routeros"`
	NeedsRouterBOOT bool   `json:"needs_routerboot"`
	Steps           []Step `json:"steps"`
}

// PlanUpgrade computes a safe, ordered upgrade plan. RouterOS is upgraded first
// (download → verify → upgrade → reboot → verify). RouterBOOT (RouterBOARD
// only) is upgraded afterward because it requires the new RouterOS and a
// second reboot.
func PlanUpgrade(in UpgradeInput) UpgradePlan {
	plan := UpgradePlan{}

	if NeedsUpgrade(in.CurrentVersion, in.TargetVersion) {
		plan.NeedsRouterOS = true
		plan.Steps = append(plan.Steps,
			Step{StepDownload, "download RouterOS " + in.TargetVersion},
			Step{StepVerifyChecksum, "verify package checksum/signature"},
			Step{StepUpgradeRouterOS, "install RouterOS " + in.TargetVersion},
			Step{StepReboot, "reboot into new RouterOS"},
			Step{StepVerifyVersion, "verify running version == " + in.TargetVersion},
			Step{StepHealthCheck, "verify connectivity, routing, interfaces"},
		)
	}

	if in.IsRouterBOARD && in.RouterBootBundled != "" &&
		CompareVersions(in.RouterBootCurrent, in.RouterBootBundled) < 0 {
		plan.NeedsRouterBOOT = true
		plan.Steps = append(plan.Steps,
			Step{StepUpgradeRouterBOOT, "upgrade RouterBOOT " + in.RouterBootCurrent + " -> " + in.RouterBootBundled},
			Step{StepReboot, "reboot to apply RouterBOOT"},
			Step{StepHealthCheck, "post-RouterBOOT health check"},
		)
	}

	return plan
}
