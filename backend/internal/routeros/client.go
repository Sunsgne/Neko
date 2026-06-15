package routeros

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/neko/sdwan/backend/internal/store"
)

// Client is a full RouterOS v7 REST client supporting CRUD on any config path,
// enabling complete device configuration WITHOUT logging into the device.
type Client struct {
	Target   Target
	Scheme   string // https (default) | http
	Insecure bool   // skip TLS verify (RouterOS self-signed certs)
	http     *http.Client
}

// NewClient builds a REST client for a target.
func NewClient(t Target) *Client {
	return &Client{
		Target:   t,
		Scheme:   "https",
		Insecure: true,
		http: &http.Client{
			Timeout:   15 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
		},
	}
}

func (c *Client) base() string {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/rest", scheme, c.Target.Address)
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base()+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(c.Target.Username, c.Target.Secret)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return data, resp.StatusCode, fmt.Errorf("routeros %s %s -> %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, resp.StatusCode, nil
}

// List returns all items at a config path (e.g. "/ip/address").
func (c *Client) List(ctx context.Context, path string) ([]map[string]any, error) {
	data, _, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	if len(data) == 0 {
		return items, nil
	}
	// Some singleton endpoints return an object instead of an array.
	if trimmed := strings.TrimSpace(string(data)); strings.HasPrefix(trimmed, "{") {
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, err
		}
		return []map[string]any{obj}, nil
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Create adds a new item at a config path (RouterOS REST: PUT /rest/<path>).
func (c *Client) Create(ctx context.Context, path string, attrs map[string]string) error {
	_, _, err := c.do(ctx, http.MethodPut, path, attrs)
	return err
}

// Update modifies an existing item by RouterOS .id (PATCH /rest/<path>/<id>).
func (c *Client) Update(ctx context.Context, path, id string, attrs map[string]string) error {
	_, _, err := c.do(ctx, http.MethodPatch, path+"/"+id, attrs)
	return err
}

// Set updates a singleton settings resource (e.g. /ip/dns, /system/identity)
// via the RouterOS "set" command: POST /rest/<path>/set.
func (c *Client) Set(ctx context.Context, path string, attrs map[string]string) error {
	_, _, err := c.do(ctx, http.MethodPost, path+"/set", attrs)
	return err
}

// Delete removes an item by RouterOS .id (DELETE /rest/<path>/<id>).
func (c *Client) Delete(ctx context.Context, path, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, path+"/"+id, nil)
	return err
}

// RunScript installs a RouterOS script under the given name and executes it
// once, returning the run output. It is idempotent w.r.t. the script object:
// any pre-existing script with the same name is removed first. This is how the
// platform delivers large rule sets (e.g. the chnroutes table) in a single
// REST round-trip instead of thousands of per-item calls.
func (c *Client) RunScript(ctx context.Context, name, source string) (string, error) {
	// Remove any prior script with this name (best-effort).
	if existing, err := c.List(ctx, "/system/script"); err == nil {
		for _, s := range existing {
			if str(s["name"]) == name {
				if id := str(s[".id"]); id != "" {
					_ = c.Delete(ctx, "/system/script", id)
				}
			}
		}
	}
	if err := c.Create(ctx, "/system/script", map[string]string{
		"name":                     name,
		"source":                   source,
		"dont-require-permissions": "yes",
		"owner":                    "neko",
	}); err != nil {
		return "", fmt.Errorf("install script: %w", err)
	}
	// Execute it. RouterOS REST exposes the CLI "run" command at
	// POST /rest/system/script/run with the script name as "number".
	data, _, err := c.do(ctx, http.MethodPost, "/system/script/run", map[string]string{"number": name})
	if err != nil {
		return string(data), fmt.Errorf("run script: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// Status reads live operational metrics (/system/resource + /interface) and
// returns them as a store.DeviceStatus. Online is set true on success.
func (c *Client) Status(ctx context.Context) (store.DeviceStatus, error) {
	st := store.DeviceStatus{}
	res, err := c.List(ctx, "/system/resource")
	if err != nil || len(res) == 0 {
		if err == nil {
			err = ErrUnreachable
		}
		return st, err
	}
	r := res[0]
	st.Online = true
	st.Version = str(r["version"])
	st.Uptime = str(r["uptime"])
	st.CPULoadPercent = int(atoi64(r["cpu-load"]))
	st.FreeMemoryBytes = atoi64(r["free-memory"])
	st.TotalMemoryBytes = atoi64(r["total-memory"])

	if ifaces, err := c.List(ctx, "/interface"); err == nil {
		st.InterfacesTotal = len(ifaces)
		for _, i := range ifaces {
			if boolish(i["running"]) {
				st.InterfacesUp++
			}
		}
	}
	// Optional board temperature via /system/health (best-effort).
	if health, err := c.List(ctx, "/system/health"); err == nil {
		for _, h := range health {
			if str(h["name"]) == "temperature" {
				st.BoardTempC = int(atoi64(h["value"]))
			}
		}
	}
	return st, nil
}

// ManagedSections is the catalog of RouterOS config paths the platform fully
// manages — covering interfaces, addressing, routing, firewall/NAT, DHCP, DNS,
// VLAN/bridge/tunnels, queues, SNMP and system. This is the "full-function
// configuration" surface (no device login required).
var ManagedSections = []string{
	"/system/identity",
	"/system/clock",
	"/system/ntp/client",
	"/ip/address",
	"/ip/route",
	"/ip/pool",
	"/ip/dhcp-server",
	"/ip/dhcp-server/network",
	"/ip/dns",
	"/ip/dns/static",
	"/ip/firewall/filter",
	"/ip/firewall/nat",
	"/ip/firewall/mangle",
	"/ip/firewall/address-list",
	"/ip/service",
	"/interface/bridge",
	"/interface/vlan",
	"/interface/wireguard",
	"/interface/wireguard/peers",
	"/interface/list",
	"/queue/simple",
	"/routing/ospf/instance",
	"/routing/bgp/connection",
	"/routing/filter/rule",
	"/snmp",
	"/snmp/community",
}
