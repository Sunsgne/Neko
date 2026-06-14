// Package store defines repository interfaces and their implementations.
//
// Per ADR-0004, interfaces come first with an in-memory implementation so the
// platform runs and is testable from day one. A pgx-backed implementation
// replaces it in Epic 1 without changing callers.
package store

import (
	"context"
	"errors"
	"time"
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
	Delete(ctx context.Context, tenantID, id string) error
}

// Credential is an encrypted credential blob for a device.
type Credential struct {
	DeviceID string
	Kind     string // api | ssh-password | ssh-key
	Sealed   string // base64 AES-GCM ciphertext (never plaintext)
}

// CredentialRepository persists encrypted device credentials.
type CredentialRepository interface {
	Put(ctx context.Context, c Credential) error
	Get(ctx context.Context, deviceID string) (*Credential, error)
}

// AlertRepository persists deduplicated alerts. Fire is idempotent per
// (device_id, code) while an alert is firing; Resolve closes it.
type AlertRepository interface {
	// Fire ensures an open alert exists for (device_id, code). Returns the
	// alert and whether it was newly created (for notification on transition).
	Fire(ctx context.Context, a Alert) (*Alert, bool, error)
	// Resolve closes any open alert for (device_id, code). Returns whether one
	// was resolved (transition).
	Resolve(ctx context.Context, deviceID, code string, at time.Time) (bool, error)
	// List returns alerts scoped to tenant ("" = all), firing first.
	List(ctx context.Context, tenantID string, limit int) ([]*Alert, error)
}

// ConfigSnapshotRepository persists device config snapshots for backup history
// and drift detection.
type ConfigSnapshotRepository interface {
	Save(ctx context.Context, s ConfigSnapshot) error
	List(ctx context.Context, deviceID string, limit int) ([]*ConfigSnapshot, error)
	Get(ctx context.Context, id string) (*ConfigSnapshot, error)
}

// SessionRepository persists login sessions so they survive API restarts.
type SessionRepository interface {
	Save(ctx context.Context, s SessionRecord) error
	Get(ctx context.Context, token string) (*SessionRecord, error)
	Delete(ctx context.Context, token string) error
}

// Store aggregates all repositories.
type Store interface {
	Tenants() TenantRepository
	Devices() DeviceRepository
	Credentials() CredentialRepository
	Alerts() AlertRepository
	Snapshots() ConfigSnapshotRepository
	Sessions() SessionRepository
}
