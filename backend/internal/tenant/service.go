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
	repo  store.TenantRepository
	store store.Store
	id    IDFunc
	now   NowFunc
}

// NewService builds a tenant service.
func NewService(st store.Store, id IDFunc, now NowFunc) *Service {
	return &Service{repo: st.Tenants(), store: st, id: id, now: now}
}

// CreateInput is the payload for creating a tenant.
type CreateInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// UpdateInput is the payload for updating a tenant.
type UpdateInput struct {
	Name   *string              `json:"name,omitempty"`
	Slug   *string              `json:"slug,omitempty"`
	Status *store.TenantStatus  `json:"status,omitempty"`
}

// DeleteInput confirms tenant deletion.
type DeleteInput struct {
	ConfirmSlug string `json:"confirm_slug"`
}

// TenantView is a tenant with summary stats for list/detail APIs.
type TenantView struct {
	*store.Tenant
	Stats store.TenantStats `json:"stats"`
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

// Update modifies tenant name, slug, or status.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (*store.Tenant, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
		}
		t.Name = name
	}
	if in.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*in.Slug))
		if !slugRe.MatchString(slug) {
			return nil, fmt.Errorf("%w: slug must be 1-40 chars [a-z0-9-]", ErrInvalidInput)
		}
		t.Slug = slug
	}
	if in.Status != nil {
		switch *in.Status {
		case store.TenantActive, store.TenantSuspended:
			t.Status = *in.Status
		default:
			return nil, fmt.Errorf("%w: invalid status", ErrInvalidInput)
		}
	}
	t.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete removes a tenant after slug confirmation. Postgres cascades FK children.
func (s *Service) Delete(ctx context.Context, id string, in DeleteInput) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if in.ConfirmSlug != t.Slug {
		return fmt.Errorf("%w: confirm_slug must match tenant slug %q", ErrInvalidInput, t.Slug)
	}
	if mem, ok := s.store.(*store.MemoryStore); ok {
		return mem.DeleteTenantCascade(ctx, id)
	}
	return s.repo.Delete(ctx, id)
}

// Get returns a tenant by id.
func (s *Service) Get(ctx context.Context, id string) (*store.Tenant, error) {
	return s.repo.Get(ctx, id)
}

// GetView returns a tenant with resource stats.
func (s *Service) GetView(ctx context.Context, id string) (TenantView, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return TenantView{}, err
	}
	st, err := s.Stats(ctx, id)
	if err != nil {
		return TenantView{}, err
	}
	return TenantView{Tenant: t, Stats: st}, nil
}

// List returns a page of tenants.
func (s *Service) List(ctx context.Context, page store.Page) ([]*store.Tenant, int, error) {
	return s.repo.List(ctx, page)
}

// ListViews returns tenants with stats for each row.
func (s *Service) ListViews(ctx context.Context, page store.Page) ([]TenantView, int, error) {
	items, total, err := s.repo.List(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]TenantView, 0, len(items))
	for _, t := range items {
		st, err := s.Stats(ctx, t.ID)
		if err != nil {
			st = store.TenantStats{}
		}
		out = append(out, TenantView{Tenant: t, Stats: st})
	}
	return out, total, nil
}

// Stats returns resource counts for a tenant.
func (s *Service) Stats(ctx context.Context, tenantID string) (store.TenantStats, error) {
	if mem, ok := s.store.(*store.MemoryStore); ok {
		return mem.TenantStats(ctx, tenantID)
	}
	if pg, ok := s.repo.(interface {
		Stats(context.Context, string) (store.TenantStats, error)
	}); ok {
		return pg.Stats(ctx, tenantID)
	}
	_, devTotal, err := s.store.Devices().List(ctx, tenantID, store.Page{Number: 1, Size: 1})
	if err != nil {
		return store.TenantStats{}, err
	}
	alerts, err := s.store.Alerts().List(ctx, tenantID, 0)
	if err != nil {
		return store.TenantStats{}, err
	}
	st := store.TenantStats{DeviceCount: devTotal}
	for _, a := range alerts {
		st.AlertCount++
		if a.State == "firing" {
			st.FiringAlerts++
		}
	}
	return st, nil
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
