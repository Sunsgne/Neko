package session

import (
	"context"
	"testing"
	"time"

	"github.com/neko/sdwan/backend/internal/auth"
)

func TestCreateAndAuthenticate(t *testing.T) {
	s := NewStore(time.Hour)
	sess := s.Create("u1", "a@b.c", "ten_1", false)
	if sess.Token == "" {
		t.Fatal("expected token")
	}
	p, err := s.Authenticate(context.Background(), sess.Token)
	if err != nil {
		t.Fatal(err)
	}
	if p.Scope() != "ten_1" {
		t.Errorf("scope = %q, want ten_1", p.Scope())
	}
}

func TestLogoutInvalidates(t *testing.T) {
	s := NewStore(time.Hour)
	sess := s.Create("u1", "a@b.c", "", true)
	s.Delete(sess.Token)
	if _, err := s.Authenticate(context.Background(), sess.Token); err != auth.ErrUnauthorized {
		t.Errorf("expected unauthorized after logout, got %v", err)
	}
}

func TestExpiry(t *testing.T) {
	s := NewStore(time.Hour)
	base := time.Unix(1000, 0)
	s.now = func() time.Time { return base }
	sess := s.Create("u1", "a@b.c", "", true)

	s.now = func() time.Time { return base.Add(2 * time.Hour) }
	if _, err := s.Authenticate(context.Background(), sess.Token); err != auth.ErrUnauthorized {
		t.Errorf("expected expired token to be unauthorized, got %v", err)
	}
}

func TestInvalidToken(t *testing.T) {
	s := NewStore(time.Hour)
	if _, err := s.Authenticate(context.Background(), "bogus"); err != auth.ErrUnauthorized {
		t.Errorf("expected unauthorized, got %v", err)
	}
}
