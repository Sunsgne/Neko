// Package discovery scans an IP range for reachable RouterOS devices via the
// REST API, producing onboarding candidates for batch enrollment.
package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/neko/sdwan/backend/internal/routeros"
)

// Candidate is a discovered, reachable RouterOS device.
type Candidate struct {
	Address string `json:"address"` // host:port
	Board   string `json:"board"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

// Options parameterize a scan.
type Options struct {
	CIDR     string
	Port     int // RouterOS REST port (default 443)
	Username string
	Password string
	Workers  int
	// MaxHosts caps the scan size to avoid scanning huge ranges.
	MaxHosts int
}

// ErrTooLarge indicates the CIDR exceeds MaxHosts.
type ErrTooLarge struct{ Hosts, Max int }

func (e ErrTooLarge) Error() string {
	return fmt.Sprintf("range too large: %d hosts (max %d)", e.Hosts, e.Max)
}

// Scan probes every host in the CIDR for a RouterOS REST endpoint and returns
// the reachable ones with basic identity.
func Scan(ctx context.Context, o Options) ([]Candidate, error) {
	_, ipnet, err := net.ParseCIDR(o.CIDR)
	if err != nil {
		return nil, err
	}
	port := o.Port
	if port == 0 {
		port = 443
	}
	workers := o.Workers
	if workers <= 0 {
		workers = 32
	}
	maxHosts := o.MaxHosts
	if maxHosts <= 0 {
		maxHosts = 1024
	}

	hosts := enumerate(ipnet)
	if len(hosts) > maxHosts {
		return nil, ErrTooLarge{Hosts: len(hosts), Max: maxHosts}
	}

	in := make(chan string)
	go func() {
		defer close(in)
		for _, h := range hosts {
			select {
			case <-ctx.Done():
				return
			case in <- h:
			}
		}
	}()

	var (
		mu    sync.Mutex
		found []Candidate
		wg    sync.WaitGroup
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range in {
				if ctx.Err() != nil {
					return
				}
				addr := fmt.Sprintf("%s:%d", host, port)
				c := routeros.NewClient(routeros.Target{Address: addr, Username: o.Username, Secret: o.Password})
				res, err := c.List(ctx, "/system/resource")
				if err != nil || len(res) == 0 {
					continue
				}
				r := res[0]
				mu.Lock()
				found = append(found, Candidate{
					Address: addr,
					Board:   asString(r["board-name"]),
					Version: asString(r["version"]),
					Arch:    asString(r["architecture-name"]),
				})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return found, ctx.Err()
}

func enumerate(ipnet *net.IPNet) []string {
	var out []string
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		out = append(out, ip.String())
	}
	return out
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
