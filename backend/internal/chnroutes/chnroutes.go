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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

// DefaultURL is the canonical chnroutes2 source on GitHub (raw text, one CIDR per line).
const DefaultURL = "https://raw.githubusercontent.com/misakaio/chnroutes2/master/chnroutes.txt"

// Fallback mirrors for environments where raw.githubusercontent.com is unreachable
// (e.g. Docker 127.0.0.11 DNS issues or mainland network restrictions).
var DefaultFallbackURLs = []string{
	"https://cdn.jsdelivr.net/gh/misakaio/chnroutes2@master/chnroutes.txt",
	"https://ghproxy.net/https://raw.githubusercontent.com/misakaio/chnroutes2/master/chnroutes.txt",
	DefaultURL,
}

// maxBody caps the download size (the list is ~200KB; 8MB is a safe ceiling).
const maxBody = 8 << 20

// Cache holds the most recently fetched China prefix table. It is safe for
// concurrent use.
type Cache struct {
	mu        sync.RWMutex
	url       string
	sources   []string
	prefixes  []string
	updatedAt time.Time
	client    *http.Client
}

// NewCache builds an empty cache with default source URLs (env overrides).
func NewCache() *Cache {
	return NewCacheWithSources(ResolveSources(""))
}

// NewCacheWithSources builds a cache that tries sources in order on refresh.
func NewCacheWithSources(sources []string) *Cache {
	if len(sources) == 0 {
		sources = DefaultFallbackURLs
	}
	return &Cache{
		sources: append([]string(nil), sources...),
		url:     sources[0],
		client:  &http.Client{Timeout: 45 * time.Second},
	}
}

// ResolveSources returns the ordered list of chnroutes download URLs.
// primary, when non-empty, is tried first (API refresh with explicit url).
// Env: NEKO_CHNROUTES_URL (single) or NEKO_CHNROUTES_URLS (comma-separated).
func ResolveSources(primary string) []string {
	if primary != "" {
		return []string{primary}
	}
	if v := strings.TrimSpace(os.Getenv("NEKO_CHNROUTES_URLS")); v != "" {
		return splitURLs(v)
	}
	if v := strings.TrimSpace(os.Getenv("NEKO_CHNROUTES_URL")); v != "" {
		return []string{v}
	}
	return append([]string(nil), DefaultFallbackURLs...)
}

func splitURLs(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Status is a serializable summary of the cache state.
type Status struct {
	URL       string    `json:"url"`
	Sources   []string  `json:"sources,omitempty"`
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
		Sources:   append([]string(nil), c.sources...),
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

// Refresh downloads and parses the prefix list. When url is empty, tries every
// configured source in order until one succeeds.
func (c *Cache) Refresh(ctx context.Context, url string) (Status, error) {
	sources := ResolveSources(url)
	if url == "" {
		c.mu.RLock()
		if len(c.sources) > 0 {
			sources = append([]string(nil), c.sources...)
		}
		c.mu.RUnlock()
	}
	if len(sources) == 0 {
		sources = DefaultFallbackURLs
	}

	var errs []error
	for _, src := range sources {
		prefixes, err := c.fetch(ctx, src)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", src, err))
			continue
		}
		if len(prefixes) == 0 {
			errs = append(errs, fmt.Errorf("%s: no valid prefixes", src))
			continue
		}
		c.mu.Lock()
		c.url = src
		c.sources = append([]string(nil), sources...)
		c.prefixes = prefixes
		c.updatedAt = time.Now().UTC()
		c.mu.Unlock()
		return c.Status(), nil
	}
	return c.Status(), fmt.Errorf("chnroutes: all sources failed: %w", errors.Join(errs...))
}

// EnsureLoaded refreshes from configured sources only if the cache is empty.
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
	req.Header.Set("User-Agent", "neko-sdwan/chnroutes")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
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
