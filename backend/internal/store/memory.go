package store

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"
)

// MemoryStore is an in-memory Store implementation used for development,
// tests, and zero-dependency runs.
type MemoryStore struct {
	tenants *memTenantRepo
	devices *memDeviceRepo
	creds   *memCredentialRepo
	alerts  *memAlertRepo
	snaps   *memSnapshotRepo
	sess    *memSessionRepo
}

// NewMemory builds a ready-to-use in-memory store.
func NewMemory() *MemoryStore {
	return &MemoryStore{
		tenants: &memTenantRepo{items: map[string]*Tenant{}},
		devices: &memDeviceRepo{items: map[string]*Device{}},
		creds:   &memCredentialRepo{items: map[string]Credential{}},
		alerts:  &memAlertRepo{items: map[string]*Alert{}},
		snaps:   &memSnapshotRepo{items: map[string]*ConfigSnapshot{}},
		sess:    &memSessionRepo{items: map[string]SessionRecord{}},
	}
}

func (m *MemoryStore) Tenants() TenantRepository           { return m.tenants }
func (m *MemoryStore) Devices() DeviceRepository           { return m.devices }
func (m *MemoryStore) Credentials() CredentialRepository   { return m.creds }
func (m *MemoryStore) Alerts() AlertRepository             { return m.alerts }
func (m *MemoryStore) Snapshots() ConfigSnapshotRepository { return m.snaps }
func (m *MemoryStore) Sessions() SessionRepository         { return m.sess }

type memSessionRepo struct {
	mu    sync.RWMutex
	items map[string]SessionRecord
}

func (r *memSessionRepo) Save(_ context.Context, s SessionRecord) error {
	r.mu.Lock()
	r.items[s.Token] = s
	r.mu.Unlock()
	return nil
}

func (r *memSessionRepo) Get(_ context.Context, token string) (*SessionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.items[token]
	if !ok {
		return nil, ErrNotFound
	}
	cp := s
	return &cp, nil
}

func (r *memSessionRepo) Delete(_ context.Context, token string) error {
	r.mu.Lock()
	delete(r.items, token)
	r.mu.Unlock()
	return nil
}

type memSnapshotRepo struct {
	mu    sync.RWMutex
	items map[string]*ConfigSnapshot
}

func (r *memSnapshotRepo) Save(_ context.Context, s ConfigSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := s
	r.items[s.ID] = &cp
	return nil
}

func (r *memSnapshotRepo) List(_ context.Context, deviceID string, limit int) ([]*ConfigSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ConfigSnapshot, 0)
	for _, s := range r.items {
		if s.DeviceID == deviceID {
			cp := *s
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TakenAt.After(out[j].TakenAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *memSnapshotRepo) Get(_ context.Context, id string) (*ConfigSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *s
	return &cp, nil
}

type memAlertRepo struct {
	mu    sync.RWMutex
	items map[string]*Alert // id -> alert
	seq   int
}

func alertKey(deviceID, code string) string { return deviceID + "\x00" + code }

func (r *memAlertRepo) Fire(_ context.Context, a Alert) (*Alert, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ex := range r.items {
		if ex.State == "firing" && ex.DeviceID == a.DeviceID && ex.Code == a.Code {
			cp := *ex
			return &cp, false, nil
		}
	}
	r.seq++
	a.ID = "al_" + strconv.Itoa(r.seq)
	a.State = "firing"
	if a.FiredAt.IsZero() {
		a.FiredAt = time.Now().UTC()
	}
	cp := a
	r.items[a.ID] = &cp
	out := a
	return &out, true, nil
}

func (r *memAlertRepo) Resolve(_ context.Context, deviceID, code string, at time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ex := range r.items {
		if ex.State == "firing" && ex.DeviceID == deviceID && ex.Code == code {
			ex.State = "resolved"
			t := at
			ex.ResolvedAt = &t
			return true, nil
		}
	}
	return false, nil
}

func (r *memAlertRepo) List(_ context.Context, tenantID string, limit int) ([]*Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Alert, 0, len(r.items))
	for _, a := range r.items {
		if tenantID == "" || a.TenantID == tenantID {
			cp := *a
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].State != out[j].State {
			return out[i].State == "firing"
		}
		return out[i].FiredAt.After(out[j].FiredAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

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

func (r *memDeviceRepo) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.items[id]
	if !ok || (tenantID != "" && d.TenantID != tenantID) {
		return ErrNotFound
	}
	delete(r.items, id)
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
