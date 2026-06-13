// Command worker runs background workers (device onboarding, SNMP polling,
// probing, lifecycle). Bootstrap stub: it starts, logs, and exits cleanly on
// signal. Concrete consumers land with their respective epics (see docs/TASKS.md).
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/neko/sdwan/backend/internal/config"
	"github.com/neko/sdwan/backend/internal/observability"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger(cfg.LogLevel, cfg.Env)
	logger.Info("worker started", "env", cfg.Env, "nats", cfg.NATSURL)
	logger.Info("no consumers registered yet; see docs/TASKS.md Epics 2/5/8")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("worker stopped")
}
