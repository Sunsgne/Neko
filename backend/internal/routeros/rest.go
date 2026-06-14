package routeros

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RestCollector talks to the RouterOS v7 REST API (https://<host>/rest/...).
// It uses HTTP Basic auth over TLS. Self-signed certificates are common on
// RouterOS, so TLS verification can be disabled per-target via Target.UseTLS
// and the InsecureSkipVerify option below.
type RestCollector struct {
	client *http.Client
	// Insecure skips TLS certificate verification (typical for RouterOS with
	// self-signed certs). Default true for convenience; set false to enforce.
	Insecure bool
	// Scheme defaults to "https"; RouterOS REST requires the www-ssl service.
	Scheme string
}

// NewRestCollector builds a REST collector with a sane timeout.
func NewRestCollector() *RestCollector {
	return &RestCollector{
		client:   &http.Client{Timeout: 10 * time.Second},
		Insecure: true,
		Scheme:   "https",
	}
}

// Collect gathers all required facts from a device via the REST API.
func (c *RestCollector) Collect(ctx context.Context, t Target) (*DeviceFacts, error) {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "https"
	}
	base := fmt.Sprintf("%s://%s/rest", scheme, t.Address)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.Insecure}, //nolint:gosec // RouterOS self-signed certs
	}
	client := &http.Client{Timeout: c.client.Timeout, Transport: transport}

	facts := &DeviceFacts{}

	// /system/resource (single object)
	var res map[string]any
	if err := c.get(ctx, client, base+"/system/resource", t, &res); err != nil {
		return nil, err
	}
	facts.Resource = SystemResource{
		BoardName:        str(res["board-name"]),
		Architecture:     str(res["architecture-name"]),
		Version:          str(res["version"]),
		CPU:              str(res["cpu"]),
		CPUCount:         str(res["cpu-count"]),
		TotalMemoryBytes: atoi64(res["total-memory"]),
		Platform:         str(res["platform"]),
	}

	// /system/routerboard (single object). Absent/false on CHR/x86.
	var rb map[string]any
	if err := c.get(ctx, client, base+"/system/routerboard", t, &rb); err == nil {
		facts.Routerboard = RouterboardInfo{
			Routerboard:     boolish(rb["routerboard"]),
			Model:           str(rb["model"]),
			SerialNumber:    str(rb["serial-number"]),
			CurrentFirmware: str(rb["current-firmware"]),
			UpgradeFirmware: str(rb["upgrade-firmware"]),
			FirmwareType:    str(rb["firmware-type"]),
		}
	}

	// /system/package (array)
	var pkgs []map[string]any
	if err := c.get(ctx, client, base+"/system/package", t, &pkgs); err == nil {
		for _, p := range pkgs {
			facts.Packages = append(facts.Packages, Package{
				Name:     str(p["name"]),
				Version:  str(p["version"]),
				Disabled: boolish(p["disabled"]),
			})
		}
	}

	// /system/license (single object; CHR)
	var lic map[string]any
	if err := c.get(ctx, client, base+"/system/license", t, &lic); err == nil {
		facts.License = License{
			Level:  int(atoi64(lic["nlevel"])),
			Level6: str(lic["level"]),
		}
	}

	// /system/device-mode (single object; ROS 7.x)
	var dm map[string]any
	if err := c.get(ctx, client, base+"/system/device-mode", t, &dm); err == nil {
		facts.DeviceMode = DeviceMode{Mode: str(dm["mode"])}
	}

	// /interface (array)
	var ifaces []map[string]any
	if err := c.get(ctx, client, base+"/interface", t, &ifaces); err == nil {
		for _, i := range ifaces {
			facts.Interfaces = append(facts.Interfaces, Interface{
				Name:    str(i["name"]),
				Type:    str(i["type"]),
				Running: boolish(i["running"]),
				MTU:     int(atoi64(i["mtu"])),
			})
		}
	}

	return facts, nil
}

func (c *RestCollector) get(ctx context.Context, client *http.Client, url string, t Target, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(t.Username, t.Secret)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("routeros auth failed (%d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("routeros %s -> %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, dst)
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func atoi64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	}
	return 0
}

func boolish(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		return s == "true" || s == "yes"
	}
	return false
}
