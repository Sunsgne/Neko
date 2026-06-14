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
	// AuthEnabled gates bearer-token authentication. Default off in
	// development for zero-friction; set NEKO_AUTH=on in production.
	AuthEnabled   bool
	OperatorToken string // seed operator token when auth is enabled
	// Seed populates demo data into the memory store/catalog at startup.
	Seed bool
	// AdminEmail/AdminPassword seed the initial platform operator account when
	// auth is enabled. Empty values fall back to demo defaults.
	AdminEmail    string
	AdminPassword string
}

// Load reads configuration from the environment, applying sensible defaults
// so the API can run with zero external dependencies (memory store).
func Load() Config {
	return Config{
		Env:           env("NEKO_ENV", "development"),
		HTTPAddr:      env("NEKO_HTTP_ADDR", ":8080"),
		LogLevel:      env("NEKO_LOG_LEVEL", "info"),
		Store:         env("NEKO_STORE", "memory"),
		DatabaseURL:   env("DATABASE_URL", ""),
		RedisURL:      env("REDIS_URL", ""),
		NATSURL:       env("NATS_URL", ""),
		VMURL:         env("VM_URL", ""),
		OTLPEndpoint:  env("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		ServiceName:   env("OTEL_SERVICE_NAME", "neko-api"),
		AuthEnabled:   strings.EqualFold(env("NEKO_AUTH", "off"), "on"),
		OperatorToken: env("NEKO_OPERATOR_TOKEN", ""),
		Seed:          strings.EqualFold(env("NEKO_SEED", "false"), "true"),
		AdminEmail:    env("NEKO_ADMIN_EMAIL", ""),
		AdminPassword: env("NEKO_ADMIN_PASSWORD", ""),
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
