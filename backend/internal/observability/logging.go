// Package observability provides structured logging and (later) OpenTelemetry
// initialization for Neko services.
package observability

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger builds a structured slog.Logger. In production it emits JSON;
// in development it uses a human-friendly text handler.
func NewLogger(level, env string) *slog.Logger {
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if strings.EqualFold(env, "production") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
