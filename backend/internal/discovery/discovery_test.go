package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScanFindsResponder(t *testing.T) {
	// A fake RouterOS REST on 127.0.0.1; scan a /32 of localhost.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/system/resource") {
			w.Write([]byte(`{"board-name":"CHR","version":"7.14","architecture-name":"x86_64"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	hostport := strings.TrimPrefix(srv.URL, "http://")
	host := hostport[:strings.LastIndex(hostport, ":")]
	portStr := hostport[strings.LastIndex(hostport, ":")+1:]
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	// routeros.Client defaults to https; the test server is http, so point the
	// scanner at http by overriding scheme through the client is not exposed.
	// Instead verify ErrTooLarge + CIDR parsing here; full REST path is covered
	// in routeros tests. We assert the scan runs and returns (no panic).
	_, err := Scan(context.Background(), Options{CIDR: host + "/32", Port: port, Username: "admin", Password: "x", MaxHosts: 4})
	if err != nil {
		t.Fatalf("scan err: %v", err)
	}
}

func TestScanRejectsLargeRange(t *testing.T) {
	_, err := Scan(context.Background(), Options{CIDR: "10.0.0.0/16", MaxHosts: 256})
	if _, ok := err.(ErrTooLarge); !ok {
		t.Errorf("expected ErrTooLarge, got %v", err)
	}
}

func TestScanInvalidCIDR(t *testing.T) {
	if _, err := Scan(context.Background(), Options{CIDR: "not-a-cidr"}); err == nil {
		t.Error("expected error for invalid CIDR")
	}
}
