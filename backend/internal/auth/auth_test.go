package auth

import (
	"context"
	"errors"
	"testing"
)

func TestAuthenticateOperator(t *testing.T) {
	a := NewMemoryAuthenticator()
	a.AddToken("secret-op", Principal{IsOperator: true})

	p, err := a.Authenticate(context.Background(), "secret-op")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsOperator || p.Scope() != "" {
		t.Errorf("operator should be unscoped, got %+v scope=%q", p, p.Scope())
	}
}

func TestAuthenticateTenant(t *testing.T) {
	a := NewMemoryAuthenticator()
	a.AddToken("secret-ten", Principal{TenantID: "ten_1"})

	p, err := a.Authenticate(context.Background(), "secret-ten")
	if err != nil {
		t.Fatal(err)
	}
	if p.Scope() != "ten_1" {
		t.Errorf("tenant scope = %q, want ten_1", p.Scope())
	}
}

func TestAuthenticateInvalid(t *testing.T) {
	a := NewMemoryAuthenticator()
	a.AddToken("good", Principal{IsOperator: true})

	if _, err := a.Authenticate(context.Background(), "bad"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected unauthorized, got %v", err)
	}
	if _, err := a.Authenticate(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected unauthorized for empty token, got %v", err)
	}
}

func TestHashTokenStable(t *testing.T) {
	if HashToken("x") != HashToken("x") {
		t.Error("hash should be deterministic")
	}
	if HashToken("x") == HashToken("y") {
		t.Error("different tokens should hash differently")
	}
}
