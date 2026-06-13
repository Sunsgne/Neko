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

	"github.com/neko/sdwan/backend/internal/auth"
	"github.com/neko/sdwan/backend/internal/config"
	"github.com/neko/sdwan/backend/internal/httpapi"
	"github.com/neko/sdwan/backend/internal/idgen"
	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/observability"
	"github.com/neko/sdwan/backend/internal/store"
	"github.com/neko/sdwan/backend/internal/tenant"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger(cfg.LogLevel, cfg.Env)

	// Store selection. memory is the zero-dependency default (ADR-0004).
	var st store.Store
	switch cfg.Store {
	case "memory", "":
		st = store.NewMemory()
	default:
		logger.Warn("unsupported store, falling back to memory (postgres lands in Epic 1)", "store", cfg.Store)
		st = store.NewMemory()
		cfg.Store = "memory"
	}

	now := func() time.Time { return time.Now().UTC() }
	tenantSvc := tenant.NewService(st.Tenants(), func() string { return idgen.New("ten") }, now)
	// Collector is nil in bootstrap; a RouterOS REST collector is wired in
	// once device credentials/connectivity land (Epic 2 follow-up).
	inventorySvc := inventory.NewService(st.Devices(), nil, func() string { return idgen.New("dev") }, now)

	var authn auth.Authenticator
	if cfg.AuthEnabled {
		ma := auth.NewMemoryAuthenticator()
		if cfg.OperatorToken != "" {
			ma.AddToken(cfg.OperatorToken, auth.Principal{IsOperator: true})
			logger.Info("auth enabled with seeded operator token")
		} else {
			logger.Warn("auth enabled but NEKO_OPERATOR_TOKEN is empty; no tokens registered")
		}
		authn = ma
	}

	srv := httpapi.New(httpapi.Deps{
		Logger:    logger,
		Tenants:   tenantSvc,
		Inventory: inventorySvc,
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
