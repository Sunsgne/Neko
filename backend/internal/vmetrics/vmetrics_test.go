package vmetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWriteDeviceLineProtocol(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		r.Body.Read(b)
		got = string(b)
	}))
	defer srv.Close()
	c := New(srv.URL)
	err := c.WriteDevice(context.Background(), DeviceSample{TenantID: "t1", DeviceID: "d1", Online: true, CPU: 12, MemRatio: 0.4, IfacesUp: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "neko_device,tenant=t1,device=d1") {
		t.Errorf("bad line: %q", got)
	}
	if !strings.Contains(got, "online=1i") || !strings.Contains(got, "cpu=12") {
		t.Errorf("missing fields: %q", got)
	}
}

func TestQueryRangeParses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1700000000,"12.5"],[1700000060,"15"]]}]}}`))
	}))
	defer srv.Close()
	c := New(srv.URL)
	s, err := c.QueryRange(context.Background(), "cpu", `neko_device_cpu{device="d1"}`, time.Unix(1700000000, 0), time.Unix(1700000060, 0), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Points) != 2 || s.Points[0].V != 12.5 || s.Points[1].V != 15 {
		t.Errorf("unexpected points: %+v", s.Points)
	}
}

func TestDisabledClientNoop(t *testing.T) {
	c := New("")
	if c.Enabled() {
		t.Error("empty base should be disabled")
	}
	if err := c.WriteDevice(context.Background(), DeviceSample{}); err != nil {
		t.Errorf("disabled write should be noop, got %v", err)
	}
	s, _ := c.QueryRange(context.Background(), "x", "y", time.Now(), time.Now(), time.Minute)
	if len(s.Points) != 0 {
		t.Error("disabled query should be empty")
	}
}
