package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func capture(t *testing.T) (*httptest.Server, *string) {
	body := new(string)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*body = string(b)
	}))
	t.Cleanup(srv.Close)
	return srv, body
}

func TestWebhook(t *testing.T) {
	srv, body := capture(t)
	err := Webhook{URL: srv.URL}.Send(context.Background(), Message{Title: "T", Text: "x", Severity: "critical"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(*body, `"severity":"critical"`) {
		t.Errorf("webhook body: %s", *body)
	}
}

func TestDingTalkFormat(t *testing.T) {
	srv, body := capture(t)
	_ = DingTalk{URL: srv.URL}.Send(context.Background(), Message{Title: "设备离线", Text: "x", Severity: "critical"})
	var m map[string]any
	json.Unmarshal([]byte(*body), &m)
	if m["msgtype"] != "text" {
		t.Errorf("dingtalk msgtype = %v", m["msgtype"])
	}
}

func TestWeComFormat(t *testing.T) {
	srv, body := capture(t)
	_ = WeCom{URL: srv.URL}.Send(context.Background(), Message{Title: "T", Text: "x", Severity: "warning"})
	if !strings.Contains(*body, "markdown") {
		t.Errorf("wecom body: %s", *body)
	}
}

func TestFromEnvAndFanOut(t *testing.T) {
	srv, _ := capture(t)
	f := FromEnv(srv.URL, "", srv.URL)
	if len(f.Notifiers) != 2 || !f.Enabled() {
		t.Fatalf("expected 2 notifiers, got %d", len(f.Notifiers))
	}
	if err := f.Send(context.Background(), Message{Title: "T"}); err != nil {
		t.Errorf("fanout send: %v", err)
	}
	if FromEnv("", "", "").Enabled() {
		t.Error("no channels should be disabled")
	}
}
