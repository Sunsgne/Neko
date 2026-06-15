// Package chnroutes maintains the China mainland IPv4 prefix table used by the
// 国内外加速 (domestic/overseas split) acceleration mode.
//
// The list is the well-known chnroutes2 aggregation (per-country APNIC export):
// every CIDR in the list is a destination that should egress directly via the
// local ISP uplink (国内直连). Everything NOT in the list is routed overseas
// through the SD-WAN tunnel via the 0.0.0.0/1 + 128.0.0.0/1 default-override
// pair, which are more specific than 0.0.0.0/0 yet less specific than any China
// /8../24 prefix — so longest-prefix-match naturally keeps China traffic local
// and pushes the rest through the tunnel.
package chnroutes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// DefaultURL is the canonical chnroutes2 source (raw text, one CIDR per line).
const DefaultURL = "https://raw.githubusercontent.com/misakaio/chnroutes2/master/chnroutes.txt"

// maxBody caps the download size (the list is ~200KB; 8MB is a safe ceiling).
const maxBody = 8 << 20

// Cache holds the most recently fetched China prefix table. It is safe for
// concurrent use.
type Cache struct {
	mu        sync.RWMutex
	url       string
	prefixes  []string
	updatedAt time.Time
	client    *http.Client
}

// NewCache builds an empty cache bound to the default source URL.
func NewCache() *Cache {
	return &Cache{
		url:    DefaultURL,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Status is a serializable summary of the cache state.
type Status struct {
	URL       string    `json:"url"`
	Count     int       `json:"count"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Loaded    bool      `json:"loaded"`
}

// Status returns the current cache summary.
func (c *Cache) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Status{
		URL:       c.url,
		Count:     len(c.prefixes),
		UpdatedAt: c.updatedAt,
		Loaded:    len(c.prefixes) > 0,
	}
}

// Prefixes returns a copy of the cached China CIDR list.
func (c *Cache) Prefixes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, len(c.prefixes))
	copy(out, c.prefixes)
	return out
}

// Refresh downloads and parses the prefix list from url (or the cached/default
// URL when url is empty) and atomically replaces the cache contents.
func (c *Cache) Refresh(ctx context.Context, url string) (Status, error) {
	if url == "" {
		c.mu.RLock()
		url = c.url
		c.mu.RUnlock()
	}
	if url == "" {
		url = DefaultURL
	}
	prefixes, err := c.fetch(ctx, url)
	if err != nil {
		return c.Status(), err
	}
	if len(prefixes) == 0 {
		return c.Status(), fmt.Errorf("chnroutes: source %q returned no valid prefixes", url)
	}
	c.mu.Lock()
	c.url = url
	c.prefixes = prefixes
	c.updatedAt = time.Now().UTC()
	c.mu.Unlock()
	return c.Status(), nil
}

// EnsureLoaded refreshes from the default/cached URL only if the cache is empty.
func (c *Cache) EnsureLoaded(ctx context.Context) (Status, error) {
	if st := c.Status(); st.Loaded {
		return st, nil
	}
	return c.Refresh(ctx, "")
}

func (c *Cache) fetch(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chnroutes: fetch %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chnroutes: fetch %q: status %d", url, resp.StatusCode)
	}
	return Parse(io.LimitReader(resp.Body, maxBody))
}

// Parse reads one CIDR per line, ignoring blank lines and comments (# or ;).
// Only syntactically valid IPv4 prefixes are kept, de-duplicated, in order.
func Parse(r io.Reader) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// Some exports append trailing comments after whitespace.
		if i := strings.IndexAny(line, " \t"); i > 0 {
			line = line[:i]
		}
		p, err := netip.ParsePrefix(line)
		if err != nil || !p.Addr().Is4() {
			continue
		}
		norm := p.Masked().String()
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
