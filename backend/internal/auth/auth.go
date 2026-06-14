// Package auth provides API token authentication and principal resolution.
// Tokens are stored only as SHA-256 hashes; plaintext is never persisted.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
)

// ErrUnauthorized indicates a missing or invalid token.
var ErrUnauthorized = errors.New("unauthorized")

// Principal is the authenticated identity for a request.
type Principal struct {
	TokenID    string // user id (or token id)
	Email      string
	TenantID   string // empty for platform operators
	IsOperator bool
}

// Scoped returns the tenant id this principal is constrained to. Operators are
// unconstrained (empty string = cross-tenant).
func (p Principal) Scope() string {
	if p.IsOperator {
		return ""
	}
	return p.TenantID
}

// Authenticator validates a bearer token and returns its principal.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (Principal, error)
}

// HashToken returns the hex SHA-256 of a token for storage/lookup.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// MemoryAuthenticator is an in-memory token store keyed by token hash.
type MemoryAuthenticator struct {
	mu     sync.RWMutex
	byHash map[string]Principal
}

// NewMemoryAuthenticator builds an empty in-memory authenticator.
func NewMemoryAuthenticator() *MemoryAuthenticator {
	return &MemoryAuthenticator{byHash: map[string]Principal{}}
}

// AddToken registers a plaintext token mapped to a principal.
func (m *MemoryAuthenticator) AddToken(token string, p Principal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.TokenID = HashToken(token)[:12]
	m.byHash[HashToken(token)] = p
}

// Authenticate implements Authenticator.
func (m *MemoryAuthenticator) Authenticate(_ context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, ErrUnauthorized
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.byHash[HashToken(token)]
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	return p, nil
}
