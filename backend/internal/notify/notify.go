// Package notify delivers alert notifications to external channels: a generic
// JSON webhook, DingTalk (钉钉) robots, and WeCom (企业微信) robots.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Message is a channel-agnostic notification.
type Message struct {
	Title    string
	Text     string
	Severity string
}

// Notifier delivers a message to a channel.
type Notifier interface {
	Send(ctx context.Context, m Message) error
	Kind() string
}

func post(ctx context.Context, url string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("notify %s -> %d", url, resp.StatusCode)
	}
	return nil
}

// Webhook posts a generic JSON payload.
type Webhook struct{ URL string }

func (n Webhook) Kind() string { return "webhook" }
func (n Webhook) Send(ctx context.Context, m Message) error {
	return post(ctx, n.URL, map[string]string{"title": m.Title, "text": m.Text, "severity": m.Severity})
}

// DingTalk posts to a DingTalk custom-robot webhook (text message).
type DingTalk struct{ URL string }

func (n DingTalk) Kind() string { return "dingtalk" }
func (n DingTalk) Send(ctx context.Context, m Message) error {
	return post(ctx, n.URL, map[string]any{
		"msgtype": "text",
		"text":    map[string]string{"content": fmt.Sprintf("[Neko][%s] %s\n%s", m.Severity, m.Title, m.Text)},
	})
}

// WeCom posts to a WeCom (企业微信) group-robot webhook (markdown message).
type WeCom struct{ URL string }

func (n WeCom) Kind() string { return "wecom" }
func (n WeCom) Send(ctx context.Context, m Message) error {
	return post(ctx, n.URL, map[string]any{
		"msgtype":  "markdown",
		"markdown": map[string]string{"content": fmt.Sprintf("**[Neko][%s] %s**\n%s", m.Severity, m.Title, m.Text)},
	})
}

// FanOut sends to all configured notifiers, collecting errors but never
// blocking on a single failure.
type FanOut struct{ Notifiers []Notifier }

// Send delivers to every notifier; returns the first error (if any).
func (f FanOut) Send(ctx context.Context, m Message) error {
	var firstErr error
	for _, n := range f.Notifiers {
		if err := n.Send(ctx, m); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// FromEnv builds a FanOut from webhook URLs (empty entries are skipped).
func FromEnv(webhook, dingtalk, wecom string) FanOut {
	var ns []Notifier
	if webhook != "" {
		ns = append(ns, Webhook{URL: webhook})
	}
	if dingtalk != "" {
		ns = append(ns, DingTalk{URL: dingtalk})
	}
	if wecom != "" {
		ns = append(ns, WeCom{URL: wecom})
	}
	return FanOut{Notifiers: ns}
}

// Enabled reports whether any channel is configured.
func (f FanOut) Enabled() bool { return len(f.Notifiers) > 0 }
