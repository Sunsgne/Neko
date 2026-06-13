package audit

import (
	"context"
	"testing"
	"time"
)

func TestRecordAndList(t *testing.T) {
	r := NewMemoryRecorder()
	ctx := context.Background()
	base := time.Unix(0, 0)

	_ = r.Record(ctx, Entry{ID: "1", TenantID: "a", Action: "create", ObjectType: "device", At: base})
	_ = r.Record(ctx, Entry{ID: "2", TenantID: "b", Action: "delete", ObjectType: "device", At: base.Add(time.Second)})
	_ = r.Record(ctx, Entry{ID: "3", TenantID: "a", Action: "update", ObjectType: "tenant", At: base.Add(2 * time.Second)})

	all, _ := r.List(ctx, "", 0)
	if len(all) != 3 {
		t.Fatalf("len all = %d, want 3", len(all))
	}
	// Most recent first.
	if all[0].ID != "3" {
		t.Errorf("expected newest first, got %s", all[0].ID)
	}

	scoped, _ := r.List(ctx, "a", 0)
	if len(scoped) != 2 {
		t.Errorf("tenant a entries = %d, want 2", len(scoped))
	}

	limited, _ := r.List(ctx, "", 1)
	if len(limited) != 1 {
		t.Errorf("limited = %d, want 1", len(limited))
	}
}
