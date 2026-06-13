// Package config loads runtime configuration from environment variables
// following 12-factor conventions. No secrets are hardcoded.
package config

import (
	"os"
	"strings"
)

// Config holds all runtime configuration for Neko services.
type Config struct {
	Env          string // development | production
	HTTPAddr     string // e.g. ":8080"
	LogLevel     string // debug | info | warn | error
	Store        string // memory | postgres
	DatabaseURL  string
	RedisURL     string
	NATSURL      string
	VMURL        string
	OTLPEndpoint string
	ServiceName  string
}

// Load reads configuration from the environment, applying sensible defaults
// so the API can run with zero external dependencies (memory store).
func Load() Config {
	return Config{
		Env:          env("NEKO_ENV", "development"),
		HTTPAddr:     env("NEKO_HTTP_ADDR", ":8080"),
		LogLevel:     env("NEKO_LOG_LEVEL", "info"),
		Store:        env("NEKO_STORE", "memory"),
		DatabaseURL:  env("DATABASE_URL", ""),
		RedisURL:     env("REDIS_URL", ""),
		NATSURL:      env("NATS_URL", ""),
		VMURL:        env("VM_URL", ""),
		OTLPEndpoint: env("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		ServiceName:  env("OTEL_SERVICE_NAME", "neko-api"),
	}
}

// IsProduction reports whether the service runs in production mode.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
