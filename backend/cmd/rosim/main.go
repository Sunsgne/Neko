// Command rosim is a lightweight RouterOS v7 REST API simulator. It lets the
// platform exercise real device management (enroll → poll → config push) end
// to end without physical MikroTik hardware. It is NOT a full RouterOS; it
// implements the endpoints the platform uses with realistic responses and
// in-memory CRUD for config sections.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/big"
	mrand "math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type item map[string]any

type sim struct {
	mu       sync.Mutex
	started  time.Time
	sections map[string][]item // path -> items
	seq      int
	board    string
	version  string
	arch     string
}

func newSim() *sim {
	board := env("ROSIM_BOARD", "CCR2004-1G-12S+2XS")
	s := &sim{
		started:  time.Now(),
		sections: map[string][]item{},
		board:    board,
		version:  env("ROSIM_VERSION", "7.14.3"),
		arch:     env("ROSIM_ARCH", "arm64"),
	}
	// Seed some interfaces so capability detection + status look real.
	s.sections["/interface"] = []item{
		{".id": "*1", "name": "ether1", "type": "ether", "running": "true", "mtu": "1500"},
		{".id": "*2", "name": "ether2", "type": "ether", "running": "true", "mtu": "1500"},
		{".id": "*3", "name": "sfp-sfpplus1", "type": "ether", "running": "false", "mtu": "1500"},
	}
	s.sections["/ip/address"] = []item{
		{".id": "*1", "address": "10.0.0.1/24", "interface": "ether1"},
	}
	return s
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (s *sim) uptime() string {
	d := time.Since(s.started)
	return fmt.Sprintf("%dh%dm%ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

func (s *sim) handle(w http.ResponseWriter, r *http.Request) {
	// Optional basic-auth check (any non-empty user accepted, like a fresh box
	// where the platform was given valid creds).
	if _, _, ok := r.BasicAuth(); !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="RouterOS"`)
		http.Error(w, `{"error":401,"message":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/rest")
	w.Header().Set("Content-Type", "application/json")

	// Singleton/system endpoints.
	switch {
	case path == "/system/resource" && r.Method == http.MethodGet:
		json.NewEncoder(w).Encode(item{
			"board-name": s.board, "architecture-name": s.arch, "version": s.version,
			"uptime": s.uptime(), "cpu-load": fmt.Sprintf("%d", 3+mrand.Intn(20)),
			"free-memory":  fmt.Sprintf("%d", 700_000_000+mrand.Intn(50_000_000)),
			"total-memory": "1073741824", "cpu-count": "4", "platform": "MikroTik",
		})
		return
	case path == "/system/routerboard" && r.Method == http.MethodGet:
		json.NewEncoder(w).Encode(item{
			"routerboard": "true", "model": s.board, "serial-number": env("ROSIM_SERIAL", "HEXSIM001"),
			"current-firmware": s.version, "upgrade-firmware": s.version,
		})
		return
	case path == "/system/package" && r.Method == http.MethodGet:
		json.NewEncoder(w).Encode([]item{{"name": "routeros", "version": s.version, "disabled": "false"}})
		return
	case path == "/system/license" && r.Method == http.MethodGet:
		json.NewEncoder(w).Encode(item{"nlevel": "0", "level": "6"})
		return
	case path == "/system/device-mode" && r.Method == http.MethodGet:
		json.NewEncoder(w).Encode(item{"mode": "enterprise"})
		return
	case path == "/system/health" && r.Method == http.MethodGet:
		json.NewEncoder(w).Encode([]item{{"name": "temperature", "value": fmt.Sprintf("%d", 38+mrand.Intn(8))}})
		return
	}

	// Command endpoints (e.g. /system/script/run) are POSTed and just return
	// 200 — like running a script on a real box.
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/run") {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(item{"ret": "ok"})
		return
	}
	// Singleton settings "set" command (e.g. POST /ip/dns/set): merge attrs
	// into the singleton object so a subsequent GET returns them.
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/set") {
		sec := strings.TrimSuffix(path, "/set")
		var attrs item
		json.NewDecoder(r.Body).Decode(&attrs)
		s.mu.Lock()
		cur := item{}
		if len(s.sections[sec]) == 1 {
			cur = s.sections[sec][0]
		}
		for k, v := range attrs {
			cur[k] = v
		}
		s.sections[sec] = []item{cur}
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		return
	}

	// Generic config-section CRUD.
	s.mu.Lock()
	defer s.mu.Unlock()
	// /<path>/<id> for PATCH/DELETE.
	if r.Method == http.MethodPatch || r.Method == http.MethodDelete {
		idx := strings.LastIndex(path, "/")
		sec, id := path[:idx], path[idx+1:]
		items := s.sections[sec]
		for i, it := range items {
			if it[".id"] == id {
				if r.Method == http.MethodDelete {
					s.sections[sec] = append(items[:i], items[i+1:]...)
				} else {
					var patch item
					json.NewDecoder(r.Body).Decode(&patch)
					for k, v := range patch {
						it[k] = v
					}
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		http.Error(w, `{"error":404}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if s.sections[path] == nil {
			json.NewEncoder(w).Encode([]item{})
			return
		}
		json.NewEncoder(w).Encode(s.sections[path])
	case http.MethodPut:
		var attrs item
		json.NewDecoder(r.Body).Decode(&attrs)
		s.seq++
		attrs[".id"] = fmt.Sprintf("*%X", 0x100+s.seq)
		s.sections[path] = append(s.sections[path], attrs)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(attrs)
	default:
		http.Error(w, `{"error":405}`, http.StatusMethodNotAllowed)
	}
}

func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "neko-rosim"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func main() {
	// RouterOS REST is served over HTTPS (www-ssl); mirror that with a
	// self-signed cert so the platform's collector/applier work unchanged.
	addr := env("ROSIM_ADDR", ":8729")
	s := newSim()
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/", s.handle)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "neko RouterOS simulator: %s %s\n", s.board, s.version)
	})
	cert, err := selfSignedCert()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cert:", err)
		os.Exit(1)
	}
	fmt.Printf("rosim listening (https) on %s (board=%s version=%s)\n", addr, s.board, s.version)
	srv := &http.Server{
		Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	if err := srv.ListenAndServeTLS("", ""); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
