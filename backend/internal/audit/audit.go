// Package audit records an append-only trail of who changed what and when.
// Entries are immutable: the recorder exposes append and query, never update
// or delete.
package audit

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Entry is a single immutable audit record.
type Entry struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	ActorID    string            `json:"actor_id"`
	Action     string            `json:"action"`      // create | update | delete | apply ...
	ObjectType string            `json:"object_type"` // tenant | device | config ...
	ObjectID   string            `json:"object_id"`
	Before     map[string]string `json:"before,omitempty"`
	After      map[string]string `json:"after,omitempty"`
	At         time.Time         `json:"at"`
}

// Recorder appends and queries audit entries. Implementations MUST NOT expose
// mutation or deletion of existing entries.
type Recorder interface {
	Record(ctx context.Context, e Entry) error
	List(ctx context.Context, tenantID string, limit int) ([]Entry, error)
}

// MemoryRecorder is an append-only in-memory recorder.
type MemoryRecorder struct {
	mu      sync.RWMutex
	entries []Entry
}

// NewMemoryRecorder builds an empty recorder.
func NewMemoryRecorder() *MemoryRecorder { return &MemoryRecorder{} }

// Record appends an entry.
func (r *MemoryRecorder) Record(_ context.Context, e Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
	return nil
}

// List returns the most recent entries, optionally filtered by tenant.
func (r *MemoryRecorder) List(_ context.Context, tenantID string, limit int) ([]Entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.entries))
	for _, e := range r.entries {
		if tenantID == "" || e.TenantID == tenantID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
