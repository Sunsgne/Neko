// Package store defines repository interfaces and their implementations.
//
// Per ADR-0004, interfaces come first with an in-memory implementation so the
// platform runs and is testable from day one. A pgx-backed implementation
// replaces it in Epic 1 without changing callers.
package store

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on uniqueness violations (e.g. duplicate slug).
var ErrConflict = errors.New("conflict")

// Page describes pagination input.
type Page struct {
	Number int
	Size   int
}

// Normalize applies sane defaults and bounds to pagination input.
func (p Page) Normalize() Page {
	if p.Number < 1 {
		p.Number = 1
	}
	if p.Size <= 0 {
		p.Size = 20
	}
	if p.Size > 200 {
		p.Size = 200
	}
	return p
}

// Offset returns the SQL/slice offset for the page.
func (p Page) Offset() int { return (p.Number - 1) * p.Size }

// TenantRepository persists tenants.
type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	Get(ctx context.Context, id string) (*Tenant, error)
	List(ctx context.Context, page Page) ([]*Tenant, int, error)
}

// DeviceRepository persists devices, scoped by tenant when tenantID != "".
type DeviceRepository interface {
	Create(ctx context.Context, d *Device) error
	Get(ctx context.Context, tenantID, id string) (*Device, error)
	List(ctx context.Context, tenantID string, page Page) ([]*Device, int, error)
	Update(ctx context.Context, d *Device) error
}

// Store aggregates all repositories.
type Store interface {
	Tenants() TenantRepository
	Devices() DeviceRepository
}
