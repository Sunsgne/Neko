// Package session issues and validates opaque bearer tokens backed by a
// persistent store, so sessions survive API restarts. It implements
// auth.Authenticator for the HTTP middleware.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/neko/sdwan/backend/internal/auth"
	"github.com/neko/sdwan/backend/internal/store"
)

// Session is an issued login session (returned to the caller on create).
type Session struct {
	Token      string
	UserID     string
	Email      string
	TenantID   string
	IsOperator bool
	ExpiresAt  time.Time
}

// Store issues and validates sessions against a backing repository.
type Store struct {
	repo store.SessionRepository
	ttl  time.Duration
	now  func() time.Time
}

// NewStore builds a session store backed by repo with the given token TTL.
func NewStore(repo store.SessionRepository, ttl time.Duration) *Store {
	return &Store{repo: repo, ttl: ttl, now: time.Now}
}

// Create issues a new session token for a principal and persists it.
func (s *Store) Create(userID, email, tenantID string, isOperator bool) Session {
	tok := newToken()
	rec := store.SessionRecord{
		Token:      tok,
		UserID:     userID,
		Email:      email,
		TenantID:   tenantID,
		IsOperator: isOperator,
		ExpiresAt:  s.now().Add(s.ttl),
	}
	_ = s.repo.Save(context.Background(), rec)
	return Session{
		Token: tok, UserID: userID, Email: email, TenantID: tenantID,
		IsOperator: isOperator, ExpiresAt: rec.ExpiresAt,
	}
}

// Delete invalidates a token (logout).
func (s *Store) Delete(token string) { _ = s.repo.Delete(context.Background(), token) }

// Authenticate implements auth.Authenticator.
func (s *Store) Authenticate(ctx context.Context, token string) (auth.Principal, error) {
	if token == "" {
		return auth.Principal{}, auth.ErrUnauthorized
	}
	rec, err := s.repo.Get(ctx, token)
	if err != nil {
		return auth.Principal{}, auth.ErrUnauthorized
	}
	if s.now().After(rec.ExpiresAt) {
		_ = s.repo.Delete(ctx, token)
		return auth.Principal{}, auth.ErrUnauthorized
	}
	return auth.Principal{
		TokenID:    rec.UserID,
		Email:      rec.Email,
		TenantID:   rec.TenantID,
		IsOperator: rec.IsOperator,
	}, nil
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
