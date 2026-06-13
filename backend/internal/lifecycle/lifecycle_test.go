package lifecycle

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"7.14.3", "7.14.3", 0},
		{"7.14.2", "7.14.3", -1},
		{"7.15", "7.14.3", 1},
		{"6.49.10 (stable)", "7.1", -1},
		{"7.14", "7.14.0", 0},
		{"7.2", "7.10", -1}, // numeric, not lexical
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestPlanUpgradeRouterOSOnly(t *testing.T) {
	plan := PlanUpgrade(UpgradeInput{
		CurrentVersion: "7.13", TargetVersion: "7.14.3",
		IsRouterBOARD: true, RouterBootCurrent: "7.14.3", RouterBootBundled: "7.14.3",
	})
	if !plan.NeedsRouterOS || plan.NeedsRouterBOOT {
		t.Fatalf("expected RouterOS-only upgrade, got %+v", plan)
	}
	if plan.Steps[0].Kind != StepDownload || plan.Steps[len(plan.Steps)-1].Kind != StepHealthCheck {
		t.Errorf("unexpected step ordering: %+v", plan.Steps)
	}
}

func TestPlanUpgradeWithRouterBOOT(t *testing.T) {
	plan := PlanUpgrade(UpgradeInput{
		CurrentVersion: "7.13", TargetVersion: "7.14.3",
		IsRouterBOARD: true, RouterBootCurrent: "7.13", RouterBootBundled: "7.14.3",
	})
	if !plan.NeedsRouterOS || !plan.NeedsRouterBOOT {
		t.Fatalf("expected both upgrades, got %+v", plan)
	}
	// RouterBOOT upgrade must come after RouterOS steps.
	var osIdx, bootIdx int
	for i, s := range plan.Steps {
		if s.Kind == StepUpgradeRouterOS {
			osIdx = i
		}
		if s.Kind == StepUpgradeRouterBOOT {
			bootIdx = i
		}
	}
	if bootIdx < osIdx {
		t.Error("RouterBOOT upgrade must follow RouterOS upgrade")
	}
}

func TestPlanUpgradeNoop(t *testing.T) {
	plan := PlanUpgrade(UpgradeInput{CurrentVersion: "7.14.3", TargetVersion: "7.14.3"})
	if plan.NeedsRouterOS || len(plan.Steps) != 0 {
		t.Errorf("expected no-op, got %+v", plan)
	}
}

func TestPlanUpgradeCHRNoRouterBOOT(t *testing.T) {
	plan := PlanUpgrade(UpgradeInput{
		CurrentVersion: "7.13", TargetVersion: "7.14",
		IsRouterBOARD: false, RouterBootCurrent: "1.0", RouterBootBundled: "2.0",
	})
	if plan.NeedsRouterBOOT {
		t.Error("CHR/x86 must not get a RouterBOOT upgrade")
	}
}

func TestInitTemplate(t *testing.T) {
	st := InitTemplate(InitOptions{
		Identity: "edge-sh-01", Timezone: "Asia/Shanghai",
		NTPServers: "ntp1.aliyun.com", SNMPCommunity: "neko",
		MgmtInterface: "ether1", AllowMgmtFrom: "10.0.0.0/24",
	})
	paths := map[string]bool{}
	for _, s := range st.Statements {
		paths[s.Path] = true
	}
	for _, want := range []string{"/system/identity", "/system/clock", "/system/ntp/client", "/snmp/community", "/ip/firewall/filter"} {
		if !paths[want] {
			t.Errorf("init template missing %s", want)
		}
	}
}
