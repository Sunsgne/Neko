package qos

import "testing"

func TestBuildSimpleQueues(t *testing.T) {
	st, err := BuildSimpleQueues([]Rule{
		{Name: "lan-cap", Target: "192.168.0.0/24", MaxLimit: "10", Priority: 5},
		{Name: "guest", Target: "10.88.0.0/24", MaxLimit: "5M/2M", LimitAt: "1M"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Statements) != 2 {
		t.Fatalf("want 2 statements, got %d", len(st.Statements))
	}
	if st.Statements[0].Attributes["max-limit"] != "10M/10M" {
		t.Fatalf("normalized max-limit: %s", st.Statements[0].Attributes["max-limit"])
	}
	if st.Statements[1].Path != "/queue/simple" {
		t.Fatal("wrong path")
	}
}

func TestValidateRule(t *testing.T) {
	if err := ValidateRule(Rule{}); err == nil {
		t.Fatal("empty rule should fail")
	}
	if err := ValidateRule(Rule{Name: "x", Target: "1.1.1.1", MaxLimit: "abc"}); err == nil {
		t.Fatal("bad rate should fail")
	}
}

func TestNormalizeRate(t *testing.T) {
	if got := NormalizeRate("10"); got != "10M" {
		t.Fatalf("got %s", got)
	}
	if got := NormalizeRate("512K"); got != "512K" {
		t.Fatalf("got %s", got)
	}
}
