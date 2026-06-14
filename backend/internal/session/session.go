// Package session issues and validates opaque bearer tokens backed by an
// in-memory store. It implements auth.Authenticator so the existing HTTP
// middleware can authenticate requests by token.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/neko/sdwan/backend/internal/auth"
)

// Session is an issued login session.
type Session struct {
	Token      string
	UserID     string
	Email      string
	TenantID   string
	IsOperator bool
	ExpiresAt  time.Time
}

// Store holds active sessions in memory.
type Store struct {
	mu      sync.RWMutex
	byToken map[string]Session
	ttl     time.Duration
	now     func() time.Time
}

// NewStore builds a session store with the given token TTL.
func NewStore(ttl time.Duration) *Store {
	return &Store{
		byToken: map[string]Session{},
		ttl:     ttl,
		now:     time.Now,
	}
}

// Create issues a new session token for a principal.
func (s *Store) Create(userID, email, tenantID string, isOperator bool) Session {
	tok := newToken()
	sess := Session{
		Token:      tok,
		UserID:     userID,
		Email:      email,
		TenantID:   tenantID,
		IsOperator: isOperator,
		ExpiresAt:  s.now().Add(s.ttl),
	}
	s.mu.Lock()
	s.byToken[tok] = sess
	s.mu.Unlock()
	return sess
}

// Delete invalidates a token (logout).
func (s *Store) Delete(token string) {
	s.mu.Lock()
	delete(s.byToken, token)
	s.mu.Unlock()
}

// Authenticate implements auth.Authenticator.
func (s *Store) Authenticate(_ context.Context, token string) (auth.Principal, error) {
	if token == "" {
		return auth.Principal{}, auth.ErrUnauthorized
	}
	s.mu.RLock()
	sess, ok := s.byToken[token]
	s.mu.RUnlock()
	if !ok {
		return auth.Principal{}, auth.ErrUnauthorized
	}
	if s.now().After(sess.ExpiresAt) {
		s.Delete(token)
		return auth.Principal{}, auth.ErrUnauthorized
	}
	return auth.Principal{
		TokenID:    sess.UserID,
		Email:      sess.Email,
		TenantID:   sess.TenantID,
		IsOperator: sess.IsOperator,
	}, nil
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
