package secret

import "testing"

func TestSealOpenRoundTrip(t *testing.T) {
	s, err := New("test-key")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := s.Seal([]byte("admin:s3cr3t"))
	if err != nil {
		t.Fatal(err)
	}
	if enc == "admin:s3cr3t" {
		t.Fatal("ciphertext must differ from plaintext")
	}
	got, err := s.Open(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "admin:s3cr3t" {
		t.Errorf("roundtrip mismatch: %q", got)
	}
}

func TestWrongKeyFails(t *testing.T) {
	a, _ := New("key-a")
	b, _ := New("key-b")
	enc, _ := a.Seal([]byte("secret"))
	if _, err := b.Open(enc); err == nil {
		t.Error("decryption with wrong key must fail")
	}
}

func TestNonceRandomized(t *testing.T) {
	s, _ := New("k")
	e1, _ := s.Seal([]byte("same"))
	e2, _ := s.Seal([]byte("same"))
	if e1 == e2 {
		t.Error("same plaintext should yield different ciphertext (random nonce)")
	}
}
