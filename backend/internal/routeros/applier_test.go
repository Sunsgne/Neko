package routeros

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// fakeROS is an in-memory RouterOS REST server supporting one path's CRUD.
type fakeROS struct {
	mu    sync.Mutex
	path  string
	items map[string]map[string]any // id -> item
	seq   int
}

func newFakeROS(path string) *fakeROS {
	return &fakeROS{path: path, items: map[string]map[string]any{}}
}

func (f *fakeROS) handler() http.Handler {
	mux := http.NewServeMux()
	rest := "/rest" + f.path
	mux.HandleFunc(rest, func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			out := make([]map[string]any, 0, len(f.items))
			for _, it := range f.items {
				out = append(out, it)
			}
			json.NewEncoder(w).Encode(out)
		case http.MethodPut:
			var attrs map[string]any
			json.NewDecoder(r.Body).Decode(&attrs)
			f.seq++
			id := "*" + string(rune('A'+f.seq))
			attrs[".id"] = id
			f.items[id] = attrs
			w.WriteHeader(201)
		}
	})
	mux.HandleFunc(rest+"/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		id := strings.TrimPrefix(r.URL.Path, rest+"/")
		switch r.Method {
		case http.MethodPatch:
			var attrs map[string]any
			json.NewDecoder(r.Body).Decode(&attrs)
			if it, ok := f.items[id]; ok {
				for k, v := range attrs {
					it[k] = v
				}
			}
		case http.MethodDelete:
			delete(f.items, id)
		}
	})
	return mux
}

func (f *fakeROS) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items)
}

func newApplierFor(srv *httptest.Server, path string) *Applier {
	a := NewApplier(Target{Address: strings.TrimPrefix(srv.URL, "http://"), Username: "admin", Secret: "x"}, []string{path})
	a.client.Scheme = "http"
	return a
}

func TestApplierCreatesAndDeletesWithoutLogin(t *testing.T) {
	f := newFakeROS("/ip/firewall/address-list")
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	a := newApplierFor(srv, "/ip/firewall/address-list")
	ctx := context.Background()

	desired := configengine.State{Statements: []configengine.Statement{
		{Path: "/ip/firewall/address-list", Key: "203.0.113.0/24", Attributes: map[string]string{"address": "203.0.113.0/24", "list": "overseas"}},
	}}

	// Full config push: snapshot (empty) -> diff -> apply, no SSH/login.
	res, _, err := configengine.Execute(ctx, a, nil, desired, configengine.ApplyOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Status != "committed" {
		t.Fatalf("status = %s", res.Status)
	}
	if f.count() != 1 {
		t.Fatalf("expected 1 item created, got %d", f.count())
	}

	// Now converge to empty desired → the item should be removed.
	res2, _, err := configengine.Execute(ctx, a, nil, configengine.State{}, configengine.ApplyOptions{})
	if err != nil {
		t.Fatalf("execute remove: %v", err)
	}
	if res2.Status != "committed" {
		t.Fatalf("status2 = %s", res2.Status)
	}
	if f.count() != 0 {
		t.Fatalf("expected 0 items after removal, got %d", f.count())
	}
}

func TestApplierSnapshotReadsItems(t *testing.T) {
	f := newFakeROS("/ip/firewall/address-list")
	f.items["*A"] = map[string]any{".id": "*A", "address": "1.2.3.0/24", "list": "x"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	a := newApplierFor(srv, "/ip/firewall/address-list")

	st, err := a.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Statements) != 1 || st.Statements[0].Key != "1.2.3.0/24" {
		t.Fatalf("snapshot mismatch: %+v", st.Statements)
	}
}
