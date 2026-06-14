package routeros

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neko/sdwan/backend/internal/store"
)

// fakeRouterOS serves canned REST responses mimicking a RouterBOARD running v7.
func fakeRouterOS() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/system/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"board-name":"RB5009UG+S+IN","architecture-name":"arm64","version":"7.14.3 (stable)","total-memory":"1073741824"}`))
	})
	mux.HandleFunc("/rest/system/routerboard", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"routerboard":"true","model":"RB5009UG+S+IN","serial-number":"HEX123","current-firmware":"7.14.3","upgrade-firmware":"7.14.3"}`))
	})
	mux.HandleFunc("/rest/system/package", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"name":"routeros","version":"7.14.3","disabled":"false"},{"name":"container","version":"7.14.3","disabled":"false"}]`))
	})
	mux.HandleFunc("/rest/system/license", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"nlevel":"0","level":"6"}`))
	})
	mux.HandleFunc("/rest/system/device-mode", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"mode":"enterprise"}`))
	})
	mux.HandleFunc("/rest/interface", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"name":"ether1","type":"ether","running":"true","mtu":"1500"},{"name":"sfp-sfpplus1","type":"ether","running":"false","mtu":"1500"}]`))
	})
	return httptest.NewServer(mux)
}

func TestRestCollectorParsesFacts(t *testing.T) {
	srv := fakeRouterOS()
	defer srv.Close()

	c := NewRestCollector()
	// Point at the test server (http scheme, no TLS).
	c.Scheme = "http"
	addr := strings.TrimPrefix(srv.URL, "http://")

	facts, err := c.Collect(context.Background(), Target{Address: addr, Username: "admin", Secret: "x"})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	det := Detect(*facts)
	if det.Platform != store.PlatformRouterBOARD {
		t.Errorf("platform = %q, want routerboard", det.Platform)
	}
	if det.Model != "RB5009UG+S+IN" || det.Serial != "HEX123" {
		t.Errorf("model/serial = %q/%q", det.Model, det.Serial)
	}
	if det.Capabilities.LicenseLevel != 6 {
		t.Errorf("license = %d, want 6", det.Capabilities.LicenseLevel)
	}
	if !det.Capabilities.SupportsContainer {
		t.Error("container package should be detected")
	}
	if len(det.Capabilities.Interfaces) != 2 {
		t.Errorf("interfaces = %d, want 2", len(det.Capabilities.Interfaces))
	}
}

func TestRestCollectorAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := NewRestCollector()
	c.Scheme = "http"
	addr := strings.TrimPrefix(srv.URL, "http://")
	if _, err := c.Collect(context.Background(), Target{Address: addr}); err == nil {
		t.Error("expected auth failure error")
	}
}
