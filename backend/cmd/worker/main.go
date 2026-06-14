// Command worker runs background jobs. Currently: periodic health polling of
// enrolled (managed) devices, refreshing their live status via stored
// credentials over the RouterOS REST API.
package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/neko/sdwan/backend/internal/config"
	"github.com/neko/sdwan/backend/internal/idgen"
	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/monitoring"
	"github.com/neko/sdwan/backend/internal/notify"
	"github.com/neko/sdwan/backend/internal/observability"
	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/secret"
	"github.com/neko/sdwan/backend/internal/store"
	"github.com/neko/sdwan/backend/internal/vmetrics"
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
		Snapshots:   pg.Snapshots(),
		Collector:   routeros.NewRestCollector(),
		Probe:       routeros.ClientProbe{},
		Sealer:      sealer,
		ID:          func() string { return idgen.New("snap") },
		Now:         func() time.Time { return time.Now().UTC() },
	})

	vm := vmetrics.New(cfg.VMURL)
	notifier := notify.FromEnv(cfg.AlertWebhook, cfg.DingTalkURL, cfg.WeComURL)
	logger.Info("alert notifications", "channels", len(notifier.Notifiers))

	interval := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	logger.Info("device health poller running", "interval", interval.String(), "vm", vm.Enabled())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Config snapshot + drift detection runs less frequently than health polls.
	snapInterval := 5 * time.Minute
	snapTicker := time.NewTicker(snapInterval)
	defer snapTicker.Stop()
	logger.Info("config snapshot/drift loop running", "interval", snapInterval.String())

	pollAll(context.Background(), logger, svc, pg, vm, notifier)
	snapshotAll(context.Background(), logger, svc, pg)
	for {
		select {
		case <-ticker.C:
			pollAll(context.Background(), logger, svc, pg, vm, notifier)
		case <-snapTicker.C:
			snapshotAll(context.Background(), logger, svc, pg)
		case <-stop:
			logger.Info("worker stopped")
			return
		}
	}
}

// snapshotAll captures config snapshots for enrolled devices and fires a
// config_drift alert when the running config changed since the last snapshot.
func snapshotAll(ctx context.Context, logger interface {
	Info(string, ...any)
	Warn(string, ...any)
}, svc *inventory.Service, pg *store.PostgresStore) {
	devices, _, err := pg.Devices().List(ctx, "", store.Page{Number: 1, Size: 1000})
	if err != nil {
		return
	}
	var taken, drifted int
	now := time.Now().UTC()
	for _, d := range devices {
		if !d.Enrolled {
			continue
		}
		if _, _, err := svc.SnapshotConfig(ctx, "", d.ID, "scheduled"); err != nil {
			continue
		}
		taken++
		res, err := svc.Drift(ctx, "", d.ID)
		if err != nil {
			continue
		}
		if res.HasBaseline && res.Drifted {
			drifted++
			a := store.Alert{
				ID: idgen.New("al"), TenantID: d.TenantID, DeviceID: d.ID,
				Code: "config_drift", Severity: "warning",
				Title:  "配置漂移：" + d.Name,
				Detail: "检测到设备配置在平台外被修改（" + strconv.Itoa(len(res.Plan.Changes)) + " 处变更）",
			}
			_, _, _ = pg.Alerts().Fire(ctx, a)
		} else {
			_, _ = pg.Alerts().Resolve(ctx, d.ID, "config_drift", now)
		}
	}
	if taken > 0 {
		logger.Info("config snapshot cycle", "taken", taken, "drifted", drifted)
	}
}

func pollAll(ctx context.Context, logger interface {
	Info(string, ...any)
	Warn(string, ...any)
}, svc *inventory.Service, pg *store.PostgresStore, vm *vmetrics.Client, notifier notify.FanOut) {
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
		// Push live metrics to VictoriaMetrics for historical charts.
		if vm.Enabled() {
			sample := vmetrics.DeviceSample{TenantID: dd.TenantID, DeviceID: dd.ID, Name: dd.Name}
			if dd.Status != nil {
				sample.Online = dd.Status.Online
				sample.CPU = float64(dd.Status.CPULoadPercent)
				sample.IfacesUp = dd.Status.InterfacesUp
				if dd.Status.TotalMemoryBytes > 0 {
					sample.MemRatio = float64(dd.Status.TotalMemoryBytes-dd.Status.FreeMemoryBytes) / float64(dd.Status.TotalMemoryBytes)
				}
			}
			_ = vm.WriteDevice(ctx, sample)
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
					if notifier.Enabled() {
						_ = notifier.Send(ctx, notify.Message{Title: c.Title, Text: c.Detail, Severity: c.Severity})
					}
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
