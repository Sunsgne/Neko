package chnroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	in := `# comment line
; another comment

1.0.1.0/24
1.0.2.0/23
1.0.1.0/24
223.255.252.0/22  trailing comment
2001:db8::/32
not-a-cidr
`
	got, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1.0.1.0/24", "1.0.2.0/23", "223.255.252.0/22"}
	if len(got) != len(want) {
		t.Fatalf("want %d prefixes, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("prefix %d: want %s got %s", i, want[i], got[i])
		}
	}
}

func TestCacheRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("1.0.1.0/24\n8.8.8.0/24\n"))
	}))
	defer srv.Close()

	c := NewCache()
	st, err := c.Refresh(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if st.Count != 2 || !st.Loaded {
		t.Fatalf("unexpected status: %+v", st)
	}
	if got := c.Prefixes(); len(got) != 2 {
		t.Fatalf("want 2 cached prefixes, got %d", len(got))
	}
}

func TestRefreshEmptyErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("# only comments\n\n"))
	}))
	defer srv.Close()
	c := NewCache()
	if _, err := c.Refresh(context.Background(), srv.URL); err == nil {
		t.Error("empty prefix list should error")
	}
}

func TestRefreshFallback(t *testing.T) {
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer fail.Close()
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("1.0.1.0/24\n"))
	}))
	defer ok.Close()

	c := NewCacheWithSources([]string{fail.URL, ok.URL})
	st, err := c.Refresh(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if st.URL != ok.URL || st.Count != 1 {
		t.Fatalf("unexpected status: %+v", st)
	}
}

func TestRefreshAllSourcesFail(t *testing.T) {
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer fail.Close()
	c := NewCacheWithSources([]string{fail.URL})
	if _, err := c.Refresh(context.Background(), ""); err == nil {
		t.Fatal("expected error when all sources fail")
	}
}
