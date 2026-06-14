// Package users manages user accounts and credential verification.
//
// Password hashing uses a salted, iterated SHA-256 (PBKDF2-style) from the
// standard library to avoid external dependencies. NOTE: production should use
// argon2id or bcrypt; this is sufficient for the demo and is isolated here so
// it can be swapped without touching callers.
package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

// ErrNotFound is returned when a user does not exist.
var ErrNotFound = errors.New("user not found")

const hashIterations = 50_000

// User is an account that can authenticate.
type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	TenantID    string `json:"tenant_id"` // empty for platform operators
	IsOperator  bool   `json:"is_operator"`
	salt        string
	hash        string
}

// Public returns a safe view of the user (no credentials).
func (u User) Public() map[string]any {
	return map[string]any{
		"id":           u.ID,
		"email":        u.Email,
		"display_name": u.DisplayName,
		"tenant_id":    u.TenantID,
		"is_operator":  u.IsOperator,
	}
}

// Repository stores and looks up users.
type Repository interface {
	Add(ctx context.Context, u User, password string) error
	ByEmail(ctx context.Context, email string) (User, error)
	Verify(ctx context.Context, email, password string) (User, error)
	SetPassword(ctx context.Context, email, password string) error
}

// MemoryRepository is an in-memory user store.
type MemoryRepository struct {
	mu      sync.RWMutex
	byEmail map[string]User
}

// NewMemoryRepository builds an empty repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{byEmail: map[string]User{}}
}

// Add registers a user with a plaintext password (hashed before storage).
func (r *MemoryRepository) Add(_ context.Context, u User, password string) error {
	salt, hash := hashPassword(password)
	u.salt = salt
	u.hash = hash
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byEmail[strings.ToLower(u.Email)] = u
	return nil
}

// ByEmail returns a user by email.
func (r *MemoryRepository) ByEmail(_ context.Context, email string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

// SetPassword updates a user's password (re-hashed with a fresh salt).
func (r *MemoryRepository) SetPassword(_ context.Context, email, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(email))
	u, ok := r.byEmail[key]
	if !ok {
		return ErrNotFound
	}
	u.salt, u.hash = hashPassword(password)
	r.byEmail[key] = u
	return nil
}

// Verify checks credentials and returns the user on success.
func (r *MemoryRepository) Verify(ctx context.Context, email, password string) (User, error) {
	u, err := r.ByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	if !verifyPassword(password, u.salt, u.hash) {
		return User{}, ErrNotFound
	}
	return u, nil
}

func hashPassword(password string) (salt, hash string) {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	salt = hex.EncodeToString(b)
	return salt, deriveHash(password, salt)
}

func verifyPassword(password, salt, hash string) bool {
	got := deriveHash(password, salt)
	return subtle.ConstantTimeCompare([]byte(got), []byte(hash)) == 1
}

func deriveHash(password, salt string) string {
	data := []byte(salt + ":" + password)
	for i := 0; i < hashIterations; i++ {
		sum := sha256.Sum256(data)
		data = sum[:]
	}
	return hex.EncodeToString(data)
}
