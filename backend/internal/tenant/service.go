// Package tenant implements multi-tenant management.
package tenant

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/neko/sdwan/backend/internal/store"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$`)

// ErrInvalidInput indicates a validation failure.
var ErrInvalidInput = errors.New("invalid input")

// IDFunc generates unique identifiers.
type IDFunc func() string

// NowFunc returns the current time (injectable for tests).
type NowFunc func() time.Time

// Service contains tenant business logic.
type Service struct {
	repo store.TenantRepository
	id   IDFunc
	now  NowFunc
}

// NewService builds a tenant service.
func NewService(repo store.TenantRepository, id IDFunc, now NowFunc) *Service {
	return &Service{repo: repo, id: id, now: now}
}

// CreateInput is the payload for creating a tenant.
type CreateInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Create validates and persists a new tenant.
func (s *Service) Create(ctx context.Context, in CreateInput) (*store.Tenant, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = slugify(name)
	}
	if !slugRe.MatchString(slug) {
		return nil, fmt.Errorf("%w: slug must be 1-40 chars [a-z0-9-]", ErrInvalidInput)
	}
	now := s.now()
	t := &store.Tenant{
		ID:        s.id(),
		Name:      name,
		Slug:      slug,
		Status:    store.TenantActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Get returns a tenant by id.
func (s *Service) Get(ctx context.Context, id string) (*store.Tenant, error) {
	return s.repo.Get(ctx, id)
}

// List returns a page of tenants.
func (s *Service) List(ctx context.Context, page store.Page) ([]*store.Tenant, int, error) {
	return s.repo.List(ctx, page)
}

func slugify(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 40 {
		out = strings.Trim(out[:40], "-")
	}
	if out == "" {
		out = "tenant"
	}
	return out
}
