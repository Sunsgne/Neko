// Package httpapi wires HTTP routes, middleware and handlers for the Neko API.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/neko/sdwan/backend/internal/audit"
	"github.com/neko/sdwan/backend/internal/auth"
	"github.com/neko/sdwan/backend/internal/catalog"
	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/metrics"
	"github.com/neko/sdwan/backend/internal/session"
	"github.com/neko/sdwan/backend/internal/store"
	"github.com/neko/sdwan/backend/internal/tenant"
	"github.com/neko/sdwan/backend/internal/users"
	"github.com/neko/sdwan/backend/internal/vmetrics"
)

// Server holds dependencies for the HTTP API.
type Server struct {
	logger    *slog.Logger
	tenants   *tenant.Service
	inventory *inventory.Service
	catalog   *catalog.Catalog
	users     users.Repository
	sessions  *session.Store
	audit     audit.Recorder
	alerts    store.AlertRepository
	dns       store.DNSRepository
	vm        *vmetrics.Client
	idgen     func(string) string
	metrics   *metrics.Registry
	storeKind string
	auth      auth.Authenticator // nil = auth disabled
}

// Deps are the dependencies required to build the server.
type Deps struct {
	Logger    *slog.Logger
	Tenants   *tenant.Service
	Inventory *inventory.Service
	Catalog   *catalog.Catalog
	Users     users.Repository
	Sessions  *session.Store
	Audit     audit.Recorder
	Alerts    store.AlertRepository
	Dns       store.DNSRepository
	VM        *vmetrics.Client
	IDGen     func(string) string
	Metrics   *metrics.Registry
	StoreKind string
	Auth      auth.Authenticator
}

// New builds a Server.
func New(d Deps) *Server {
	m := d.Metrics
	if m == nil {
		m = metrics.NewRegistry()
	}
	return &Server{
		logger:    d.Logger,
		tenants:   d.Tenants,
		inventory: d.Inventory,
		catalog:   d.Catalog,
		users:     d.Users,
		sessions:  d.Sessions,
		audit:     d.Audit,
		alerts:    d.Alerts,
		dns:       d.Dns,
		vm:        d.VM,
		idgen:     d.IDGen,
		metrics:   m,
		storeKind: d.StoreKind,
		auth:      d.Auth,
	}
}

// Handler returns the fully configured http.Handler with all routes and
// middleware applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health & metrics
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	mux.HandleFunc("GET /metrics", s.handleMetrics)

	// Auth
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/me", s.handleMe)

	// Tenants (operator scope)
	mux.HandleFunc("GET /api/v1/tenants", s.handleListTenants)
	mux.HandleFunc("POST /api/v1/tenants", s.handleCreateTenant)
	mux.HandleFunc("GET /api/v1/tenants/{id}", s.handleGetTenant)

	// Devices (tenant-scoped)
	mux.HandleFunc("GET /api/v1/devices", s.handleListDevices)
	mux.HandleFunc("POST /api/v1/devices", s.handleCreateDevice)
	mux.HandleFunc("GET /api/v1/devices/{id}", s.handleGetDevice)
	mux.HandleFunc("DELETE /api/v1/devices/{id}", s.handleDeleteDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/detect", s.handleDetectDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/trust", s.handleSetDeviceTrust)
	// Device hosting (托管): enroll stores encrypted creds + connects; poll
	// refreshes live status using those stored credentials.
	mux.HandleFunc("POST /api/v1/devices/{id}/enroll", s.handleEnroll)
	mux.HandleFunc("POST /api/v1/devices/{id}/poll", s.handlePoll)
	// Full-function configuration over REST (no device login required).
	mux.HandleFunc("GET /api/v1/devices/{id}/metrics", s.handleDeviceMetrics)
	// Config backup history + drift detection.
	mux.HandleFunc("POST /api/v1/devices/{id}/snapshot", s.handleSnapshotSave)
	mux.HandleFunc("GET /api/v1/devices/{id}/snapshots", s.handleSnapshotList)
	mux.HandleFunc("GET /api/v1/devices/{id}/drift", s.handleDrift)
	mux.HandleFunc("GET /api/v1/devices/{id}/config", s.handleSnapshotConfig)
	mux.HandleFunc("PUT /api/v1/devices/{id}/config", s.handlePushConfig)
	// Unified orchestration: link selection + acceleration → preview/deliver.
	mux.HandleFunc("POST /api/v1/devices/{id}/orchestrate", s.handleOrchestrate)

	// Acceleration business modes (incl. overseas-direct) + config sections.
	mux.HandleFunc("GET /api/v1/accel/modes", s.handleAccelModes)
	mux.HandleFunc("POST /api/v1/accel/preview", s.handleAccelPreview)
	mux.HandleFunc("GET /api/v1/config/sections", s.handleConfigSections)

	// Discovery + batch onboarding.
	mux.HandleFunc("POST /api/v1/discover", s.handleDiscover)
	mux.HandleFunc("POST /api/v1/devices/batch", s.handleBatchOnboard)

	// Audit log (operator-scoped query).
	mux.HandleFunc("GET /api/v1/audit", s.handleListAudit)

	// Server-sent events: live summary stream for the console.
	mux.HandleFunc("GET /api/v1/events", s.handleEvents)

	// Monitoring read models.
	mux.HandleFunc("GET /api/v1/links", s.handleListLinks)
	mux.HandleFunc("GET /api/v1/alerts", s.handleListAlerts)
	mux.HandleFunc("GET /api/v1/dns/servers", s.handleListDNSServers)
	mux.HandleFunc("POST /api/v1/dns/servers", s.handleCreateDNSServer)
	mux.HandleFunc("DELETE /api/v1/dns/servers/{id}", s.handleDeleteDNSServer)
	mux.HandleFunc("POST /api/v1/devices/{id}/dns", s.handleDNSApply)

	// Stateless planning/validation tools (preview before apply).
	mux.HandleFunc("POST /api/v1/tools/config-diff", s.handleConfigDiff)
	mux.HandleFunc("POST /api/v1/tools/routing/validate", s.handleRoutingValidate)
	mux.HandleFunc("POST /api/v1/tools/link-score", s.handleLinkScore)

	mws := []func(http.Handler) http.Handler{
		recoverer(s.logger),
		requestID,
		instrument(s.metrics),
		logging(s.logger),
		cors,
	}
	if s.auth != nil {
		// Token auth derives tenant scope from the principal.
		mws = append(mws, authenticate(s.auth))
	} else {
		// Dev mode: scope comes from the X-Tenant-Id header.
		mws = append(mws, tenantScope)
	}
	return chain(mux, mws...)
}
