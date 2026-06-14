// Command worker runs background jobs. Currently: periodic health polling of
// enrolled (managed) devices, refreshing their live status via stored
// credentials over the RouterOS REST API.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neko/sdwan/backend/internal/config"
	"github.com/neko/sdwan/backend/internal/idgen"
	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/monitoring"
	"github.com/neko/sdwan/backend/internal/observability"
	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/secret"
	"github.com/neko/sdwan/backend/internal/store"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger(cfg.LogLevel, cfg.Env)
	logger.Info("worker started", "env", cfg.Env, "store", cfg.Store)

	if cfg.Store != "postgres" {
		logger.Warn("worker requires postgres store to share device state; idling", "store", cfg.Store)
		idle()
		return
	}

	pg, err := store.OpenPostgres(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres unavailable", "err", err)
		os.Exit(1)
	}
	defer pg.Close()

	sealer, err := secret.New(cfg.SecretKey)
	if err != nil {
		logger.Error("init sealer", "err", err)
		os.Exit(1)
	}
	svc := inventory.NewService(inventory.Deps{
		Devices:     pg.Devices(),
		Credentials: pg.Credentials(),
		Collector:   routeros.NewRestCollector(),
		Probe:       routeros.ClientProbe{},
		Sealer:      sealer,
		ID:          func() string { return "" },
		Now:         func() time.Time { return time.Now().UTC() },
	})

	interval := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	logger.Info("device health poller running", "interval", interval.String())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	pollAll(context.Background(), logger, svc, pg)
	for {
		select {
		case <-ticker.C:
			pollAll(context.Background(), logger, svc, pg)
		case <-stop:
			logger.Info("worker stopped")
			return
		}
	}
}

func pollAll(ctx context.Context, logger interface {
	Info(string, ...any)
	Warn(string, ...any)
}, svc *inventory.Service, pg *store.PostgresStore) {
	// Operator scope ("" tenant) lists all devices.
	devices, _, err := pg.Devices().List(ctx, "", store.Page{Number: 1, Size: 1000})
	if err != nil {
		logger.Warn("poll: list devices failed", "err", err)
		return
	}
	var polled, online, fired, resolved int
	th := monitoring.DefaultThresholds()
	now := time.Now().UTC()
	for _, d := range devices {
		if !d.Enrolled {
			continue
		}
		dd, err := svc.Poll(ctx, "", d.ID)
		if err != nil {
			dd = d // still evaluate (likely offline)
		}
		polled++
		if dd.Status != nil && dd.Status.Online {
			online++
		}
		// Turn health into deduplicated, persisted alerts.
		for _, c := range monitoring.Evaluate(dd, th) {
			if c.Active {
				a := store.Alert{
					ID: idgen.New("al"), TenantID: dd.TenantID, DeviceID: dd.ID,
					Code: c.Code, Severity: c.Severity, Title: c.Title, Detail: c.Detail,
				}
				if _, created, _ := pg.Alerts().Fire(ctx, a); created {
					fired++
				}
			} else {
				if ok, _ := pg.Alerts().Resolve(ctx, dd.ID, c.Code, now); ok {
					resolved++
				}
			}
		}
	}
	if polled > 0 {
		logger.Info("device poll cycle", "polled", polled, "online", online, "alerts_fired", fired, "alerts_resolved", resolved)
	}
}

func idle() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
