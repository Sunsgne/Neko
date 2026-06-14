// Command api runs the Neko control-plane HTTP API.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neko/sdwan/backend/internal/audit"
	"github.com/neko/sdwan/backend/internal/auth"
	"github.com/neko/sdwan/backend/internal/catalog"
	"github.com/neko/sdwan/backend/internal/config"
	"github.com/neko/sdwan/backend/internal/httpapi"
	"github.com/neko/sdwan/backend/internal/idgen"
	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/metrics"
	"github.com/neko/sdwan/backend/internal/observability"
	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/seed"
	"github.com/neko/sdwan/backend/internal/session"
	"github.com/neko/sdwan/backend/internal/store"
	"github.com/neko/sdwan/backend/internal/tenant"
	"github.com/neko/sdwan/backend/internal/users"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger(cfg.LogLevel, cfg.Env)

	// Store selection. memory is the zero-dependency default (ADR-0004);
	// postgres provides durable, RLS-isolated storage (T1.1/T1.3).
	var st store.Store
	switch cfg.Store {
	case "postgres":
		pg, err := store.OpenPostgres(context.Background(), cfg.DatabaseURL)
		if err != nil {
			logger.Error("postgres unavailable", "err", err)
			os.Exit(1)
		}
		if err := pg.Migrate(context.Background()); err != nil {
			logger.Error("migration failed", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		st = pg
		logger.Info("using postgres store (migrations applied)")
	case "memory", "":
		st = store.NewMemory()
		cfg.Store = "memory"
	default:
		logger.Warn("unknown store, falling back to memory", "store", cfg.Store)
		st = store.NewMemory()
		cfg.Store = "memory"
	}

	cat := catalog.New()
	if cfg.Seed {
		if err := seed.Demo(context.Background(), st, cat); err != nil {
			logger.Error("seed failed", "err", err)
		} else {
			logger.Info("demo data seeded (NEKO_SEED=true)")
		}
	}

	now := func() time.Time { return time.Now().UTC() }
	tenantSvc := tenant.NewService(st.Tenants(), func() string { return idgen.New("ten") }, now)
	// RouterOS v7 REST collector drives capability detection against real
	// devices (credentials carried per-target; tolerant of self-signed TLS).
	collector := routeros.NewRestCollector()
	inventorySvc := inventory.NewService(st.Devices(), collector, func() string { return idgen.New("dev") }, now)

	// Authentication: enabled when seeding a demo or when NEKO_AUTH=on. Backed
	// by a session store (bearer tokens) with user accounts.
	var authn auth.Authenticator
	var userRepo users.Repository
	var sessions *session.Store
	if cfg.AuthEnabled || cfg.Seed {
		ur := users.NewMemoryRepository()
		sessions = session.NewStore(12 * time.Hour)
		seed.Users(context.Background(), ur, cfg.AdminEmail, cfg.AdminPassword)
		userRepo = ur
		authn = sessions
		logger.Info("authentication enabled (session tokens)", "demo_accounts", cfg.Seed, "admin", firstNonEmpty(cfg.AdminEmail, "admin@neko.io"))
	}

	srv := httpapi.New(httpapi.Deps{
		Logger:    logger,
		Tenants:   tenantSvc,
		Inventory: inventorySvc,
		Catalog:   cat,
		Users:     userRepo,
		Sessions:  sessions,
		Audit:     audit.NewMemoryRecorder(),
		IDGen:     idgen.New,
		Metrics:   metrics.NewRegistry(),
		StoreKind: cfg.Store,
		Auth:      authn,
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.Env, "store", cfg.Store)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	logger.Info("bye")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
