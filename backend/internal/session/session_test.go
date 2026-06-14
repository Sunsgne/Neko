package session

import (
	"context"
	"testing"
	"time"

	"github.com/neko/sdwan/backend/internal/auth"
	"github.com/neko/sdwan/backend/internal/store"
)

func newStore(ttl time.Duration) *Store {
	return NewStore(store.NewMemory().Sessions(), ttl)
}

func TestCreateAndAuthenticate(t *testing.T) {
	s := newStore(time.Hour)
	sess := s.Create("u1", "a@b.c", "ten_1", false)
	if sess.Token == "" {
		t.Fatal("expected token")
	}
	p, err := s.Authenticate(context.Background(), sess.Token)
	if err != nil {
		t.Fatal(err)
	}
	if p.Scope() != "ten_1" || p.Email != "a@b.c" {
		t.Errorf("principal = %+v", p)
	}
}

func TestLogoutInvalidates(t *testing.T) {
	s := newStore(time.Hour)
	sess := s.Create("u1", "a@b.c", "", true)
	s.Delete(sess.Token)
	if _, err := s.Authenticate(context.Background(), sess.Token); err != auth.ErrUnauthorized {
		t.Errorf("expected unauthorized after logout, got %v", err)
	}
}

func TestExpiry(t *testing.T) {
	s := newStore(time.Hour)
	base := time.Unix(1000, 0)
	s.now = func() time.Time { return base }
	sess := s.Create("u1", "a@b.c", "", true)
	s.now = func() time.Time { return base.Add(2 * time.Hour) }
	if _, err := s.Authenticate(context.Background(), sess.Token); err != auth.ErrUnauthorized {
		t.Errorf("expected expired token to be unauthorized, got %v", err)
	}
}

func TestInvalidToken(t *testing.T) {
	s := newStore(time.Hour)
	if _, err := s.Authenticate(context.Background(), "bogus"); err != auth.ErrUnauthorized {
		t.Errorf("expected unauthorized, got %v", err)
	}
}

func TestPersistsAcrossStoreInstances(t *testing.T) {
	// Same backing repo => a new Store (simulating restart) still validates.
	repo := store.NewMemory().Sessions()
	s1 := NewStore(repo, time.Hour)
	sess := s1.Create("u1", "a@b.c", "", true)
	s2 := NewStore(repo, time.Hour) // "restarted" store, same repo
	if _, err := s2.Authenticate(context.Background(), sess.Token); err != nil {
		t.Errorf("session should survive store re-creation, got %v", err)
	}
}
