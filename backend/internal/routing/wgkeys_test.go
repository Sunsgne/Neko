package routing

import "testing"

func TestGenerateWGKeyPair(t *testing.T) {
	priv, pub, err := GenerateWGKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if priv == "" || pub == "" || priv == pub {
		t.Fatalf("unexpected keys: priv=%q pub=%q", priv, pub)
	}
	priv2, pub2, err := GenerateWGKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if priv == priv2 && pub == pub2 {
		t.Error("expected random keys to differ")
	}
}

func TestPopPeerOf(t *testing.T) {
	if got := PopPeerOf("100.64.0.2/30"); got != "100.64.0.1" {
		t.Fatalf("got %s", got)
	}
	if got := PopPeerOf("100.64.88.118/30"); got != "100.64.88.117" {
		t.Fatalf("aligned /30 got %s", got)
	}
}

func TestTunnelNameForPOP(t *testing.T) {
	if got := TunnelNameForPOP("pop-sh-core"); got != "wg-pop-sh-core" {
		t.Fatalf("got %s", got)
	}
}

func TestAllocateOverlayDeterministic(t *testing.T) {
	a := AllocateOverlay("cpe1", "pop1")
	b := AllocateOverlay("cpe1", "pop1")
	c := AllocateOverlay("cpe2", "pop1")
	if a != b {
		t.Fatalf("not deterministic: %s vs %s", a, b)
	}
	if a == c {
		t.Fatalf("different pairs should differ: %s", a)
	}
}
