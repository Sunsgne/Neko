package users

import (
	"context"
	"errors"
	"testing"
)

func TestAddVerify(t *testing.T) {
	r := NewMemoryRepository()
	ctx := context.Background()
	if err := r.Add(ctx, User{ID: "u1", Email: "Admin@Neko.io", IsOperator: true}, "neko12345"); err != nil {
		t.Fatal(err)
	}

	// Correct password (case-insensitive email).
	u, err := r.Verify(ctx, "admin@neko.io", "neko12345")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !u.IsOperator {
		t.Error("expected operator")
	}

	// Wrong password.
	if _, err := r.Verify(ctx, "admin@neko.io", "wrong"); err == nil {
		t.Error("expected failure for wrong password")
	}

	// Unknown user.
	if _, err := r.Verify(ctx, "nope@neko.io", "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPasswordNotStoredPlaintext(t *testing.T) {
	r := NewMemoryRepository()
	ctx := context.Background()
	_ = r.Add(ctx, User{ID: "u1", Email: "a@b.c"}, "supersecret")
	u, _ := r.ByEmail(ctx, "a@b.c")
	if u.hash == "" || u.hash == "supersecret" {
		t.Error("password must be hashed, not stored plaintext")
	}
	if !u.Public()["is_operator"].(bool) == false {
		// Public view must not contain credentials.
	}
	if _, ok := u.Public()["hash"]; ok {
		t.Error("Public() must not leak hash")
	}
}
