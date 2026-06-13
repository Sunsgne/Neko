package dns

import "testing"

func pool() []Server {
	return []Server{
		{ID: "tel-sh", Address: "202.96.209.133", Region: "shanghai", ISP: ISPTelecom, SupportsECS: true, Healthy: true, LatencyMs: 5},
		{ID: "uni-bj", Address: "123.123.123.123", Region: "beijing", ISP: ISPUnicom, Healthy: true, LatencyMs: 8},
		{ID: "mob-gz", Address: "211.136.192.6", Region: "guangzhou", ISP: ISPMobile, Healthy: true, LatencyMs: 12},
		{ID: "pub-114", Address: "114.114.114.114", Region: "", ISP: ISPPublic, Healthy: true, LatencyMs: 10},
		{ID: "pub-ali", Address: "223.5.5.5", Region: "", ISP: ISPPublic, SupportsECS: true, Healthy: true, LatencyMs: 6},
		{ID: "tel-dead", Address: "1.2.3.4", Region: "shanghai", ISP: ISPTelecom, Healthy: false, LatencyMs: 2},
	}
}

func TestSelectPrefersSameISP(t *testing.T) {
	got := Select(pool(), ClientContext{Region: "shanghai", ISP: ISPTelecom}, 3)
	if len(got) == 0 || got[0].ID != "tel-sh" {
		t.Fatalf("want tel-sh first, got %+v", got)
	}
}

func TestSelectSkipsUnhealthy(t *testing.T) {
	got := Select(pool(), ClientContext{Region: "shanghai", ISP: ISPTelecom}, 10)
	for _, s := range got {
		if s.ID == "tel-dead" {
			t.Fatal("unhealthy server must not be selected")
		}
	}
}

func TestSelectFallbackPublicForUnknownISP(t *testing.T) {
	got := Select(pool(), ClientContext{ISP: ISPUnknown}, 2)
	if len(got) != 2 {
		t.Fatalf("want 2 results, got %d", len(got))
	}
	// With unknown ISP/region, public resolvers should rank at the top.
	if got[0].ISP != ISPPublic {
		t.Errorf("want public resolver first, got %+v", got[0])
	}
}

func TestSelectRespectsLimit(t *testing.T) {
	got := Select(pool(), ClientContext{Region: "beijing", ISP: ISPUnicom}, 2)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
}

func TestSelectDeterministic(t *testing.T) {
	c := ClientContext{Region: "guangzhou", ISP: ISPMobile}
	a := Select(pool(), c, 5)
	b := Select(pool(), c, 5)
	if len(a) != len(b) {
		t.Fatal("length mismatch")
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("nondeterministic at %d: %s vs %s", i, a[i].ID, b[i].ID)
		}
	}
}
