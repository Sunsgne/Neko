package tenant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neko/sdwan/backend/internal/store"
)

func newTestService() *Service {
	st := store.NewMemory()
	n := 0
	id := func() string { n++; return "ten_" + string(rune('a'+n)) }
	now := func() time.Time { return time.Unix(0, 0).UTC() }
	return NewService(st.Tenants(), id, now)
}

func TestCreateTenant(t *testing.T) {
	svc := newTestService()
	got, err := svc.Create(context.Background(), CreateInput{Name: "Acme Corp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Slug != "acme-corp" {
		t.Errorf("slug = %q, want acme-corp", got.Slug)
	}
	if got.Status != store.TenantActive {
		t.Errorf("status = %q, want active", got.Status)
	}
}

func TestCreateTenantValidation(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Create(context.Background(), CreateInput{Name: "  "}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty name, got %v", err)
	}
	if _, err := svc.Create(context.Background(), CreateInput{Name: "ok", Slug: "BAD SLUG"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for bad slug, got %v", err)
	}
}

func TestCreateTenantDuplicateSlug(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Create(context.Background(), CreateInput{Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), CreateInput{Name: "Acme"}); !errors.Is(err, store.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestListTenantsPagination(t *testing.T) {
	svc := newTestService()
	for i := 0; i < 5; i++ {
		if _, err := svc.Create(context.Background(), CreateInput{Name: "t", Slug: "t" + string(rune('a'+i))}); err != nil {
			t.Fatal(err)
		}
	}
	items, total, err := svc.List(context.Background(), store.Page{Number: 1, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}
