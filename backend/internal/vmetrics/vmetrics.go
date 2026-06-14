// Package vmetrics writes device/link metrics to VictoriaMetrics (Influx line
// protocol) and queries time series back (Prometheus query_range), powering
// the historical charts in the console.
package vmetrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client talks to a VictoriaMetrics instance.
type Client struct {
	BaseURL string
	http    *http.Client
}

// New builds a client. An empty baseURL yields a no-op client (writes/queries
// return nil/empty), so the platform runs without VM in dev/demo.
func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 8 * time.Second}}
}

// Enabled reports whether a VM backend is configured.
func (c *Client) Enabled() bool { return c.BaseURL != "" }

// DeviceSample is one device metric datapoint.
type DeviceSample struct {
	TenantID string
	DeviceID string
	Name     string
	Online   bool
	CPU      float64 // percent 0..100
	MemRatio float64 // 0..1 used
	IfacesUp int
}

// WriteDevice pushes device metrics in Influx line protocol to /write.
func (c *Client) WriteDevice(ctx context.Context, s DeviceSample) error {
	if !c.Enabled() {
		return nil
	}
	online := 0
	if s.Online {
		online = 1
	}
	tags := fmt.Sprintf("tenant=%s,device=%s", esc(s.TenantID), esc(s.DeviceID))
	line := fmt.Sprintf("neko_device,%s cpu=%g,mem_ratio=%g,online=%di,ifaces_up=%di %d\n",
		tags, s.CPU, s.MemRatio, online, s.IfacesUp, time.Now().UnixNano())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/write", bytes.NewBufferString(line))
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("vm write status %d", resp.StatusCode)
	}
	return nil
}

// Point is a single (timestamp, value) sample.
type Point struct {
	T float64 `json:"t"` // unix seconds
	V float64 `json:"v"`
}

// Series is a named time series.
type Series struct {
	Name   string  `json:"name"`
	Points []Point `json:"points"`
}

// QueryRange runs a Prometheus range query and returns the first matching
// series' points (callers pass one metric at a time).
func (c *Client) QueryRange(ctx context.Context, name, query string, start, end time.Time, step time.Duration) (Series, error) {
	s := Series{Name: name}
	if !c.Enabled() {
		return s, nil
	}
	q := url.Values{}
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", strconv.Itoa(int(step.Seconds())))
	u := c.BaseURL + "/api/v1/query_range?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return s, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return s, err
	}
	defer resp.Body.Close()
	var out struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Values [][]any `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return s, err
	}
	if len(out.Data.Result) == 0 {
		return s, nil
	}
	for _, v := range out.Data.Result[0].Values {
		if len(v) != 2 {
			continue
		}
		ts, _ := toFloat(v[0])
		val, _ := toFloat(v[1])
		s.Points = append(s.Points, Point{T: ts, V: val})
	}
	return s, nil
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

func esc(s string) string {
	if s == "" {
		return "none"
	}
	return strings.NewReplacer(" ", "\\ ", ",", "\\,", "=", "\\=").Replace(s)
}
