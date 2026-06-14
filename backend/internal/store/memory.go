package store

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore is an in-memory Store implementation used for development,
// tests, and zero-dependency runs.
type MemoryStore struct {
	tenants *memTenantRepo
	devices *memDeviceRepo
	creds   *memCredentialRepo
}

// NewMemory builds a ready-to-use in-memory store.
func NewMemory() *MemoryStore {
	return &MemoryStore{
		tenants: &memTenantRepo{items: map[string]*Tenant{}},
		devices: &memDeviceRepo{items: map[string]*Device{}},
		creds:   &memCredentialRepo{items: map[string]Credential{}},
	}
}

func (m *MemoryStore) Tenants() TenantRepository         { return m.tenants }
func (m *MemoryStore) Devices() DeviceRepository         { return m.devices }
func (m *MemoryStore) Credentials() CredentialRepository { return m.creds }

type memCredentialRepo struct {
	mu    sync.RWMutex
	items map[string]Credential
}

func (r *memCredentialRepo) Put(_ context.Context, c Credential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[c.DeviceID] = c
	return nil
}

func (r *memCredentialRepo) Get(_ context.Context, deviceID string) (*Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.items[deviceID]
	if !ok {
		return nil, ErrNotFound
	}
	cp := c
	return &cp, nil
}

type memTenantRepo struct {
	mu    sync.RWMutex
	items map[string]*Tenant
}

func (r *memTenantRepo) Create(_ context.Context, t *Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.items {
		if existing.Slug == t.Slug {
			return ErrConflict
		}
	}
	cp := *t
	r.items[t.ID] = &cp
	return nil
}

func (r *memTenantRepo) Get(_ context.Context, id string) (*Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *memTenantRepo) List(_ context.Context, page Page) ([]*Tenant, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]*Tenant, 0, len(r.items))
	for _, t := range r.items {
		cp := *t
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
	return paginate(all, page)
}

type memDeviceRepo struct {
	mu    sync.RWMutex
	items map[string]*Device
}

func (r *memDeviceRepo) Create(_ context.Context, d *Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *d
	r.items[d.ID] = &cp
	return nil
}

func (r *memDeviceRepo) Get(_ context.Context, tenantID, id string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.items[id]
	if !ok || (tenantID != "" && d.TenantID != tenantID) {
		return nil, ErrNotFound
	}
	cp := *d
	return &cp, nil
}

func (r *memDeviceRepo) List(_ context.Context, tenantID string, page Page) ([]*Device, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]*Device, 0, len(r.items))
	for _, d := range r.items {
		if tenantID != "" && d.TenantID != tenantID {
			continue
		}
		cp := *d
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
	return paginate(all, page)
}

func (r *memDeviceRepo) Update(_ context.Context, d *Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[d.ID]; !ok {
		return ErrNotFound
	}
	cp := *d
	r.items[d.ID] = &cp
	return nil
}

func paginate[T any](items []T, page Page) ([]T, int, error) {
	page = page.Normalize()
	total := len(items)
	start := page.Offset()
	if start >= total {
		return []T{}, total, nil
	}
	end := start + page.Size
	if end > total {
		end = total
	}
	return items[start:end], total, nil
}
