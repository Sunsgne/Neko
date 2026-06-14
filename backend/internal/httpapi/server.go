// Package httpapi wires HTTP routes, middleware and handlers for the Neko API.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/neko/sdwan/backend/internal/auth"
	"github.com/neko/sdwan/backend/internal/catalog"
	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/tenant"
)

// Server holds dependencies for the HTTP API.
type Server struct {
	logger    *slog.Logger
	tenants   *tenant.Service
	inventory *inventory.Service
	catalog   *catalog.Catalog
	storeKind string
	auth      auth.Authenticator // nil = auth disabled
}

// Deps are the dependencies required to build the server.
type Deps struct {
	Logger    *slog.Logger
	Tenants   *tenant.Service
	Inventory *inventory.Service
	Catalog   *catalog.Catalog
	StoreKind string
	Auth      auth.Authenticator
}

// New builds a Server.
func New(d Deps) *Server {
	return &Server{
		logger:    d.Logger,
		tenants:   d.Tenants,
		inventory: d.Inventory,
		catalog:   d.Catalog,
		storeKind: d.StoreKind,
		auth:      d.Auth,
	}
}

// Handler returns the fully configured http.Handler with all routes and
// middleware applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)

	// Tenants (operator scope)
	mux.HandleFunc("GET /api/v1/tenants", s.handleListTenants)
	mux.HandleFunc("POST /api/v1/tenants", s.handleCreateTenant)
	mux.HandleFunc("GET /api/v1/tenants/{id}", s.handleGetTenant)

	// Devices (tenant-scoped)
	mux.HandleFunc("GET /api/v1/devices", s.handleListDevices)
	mux.HandleFunc("POST /api/v1/devices", s.handleCreateDevice)
	mux.HandleFunc("GET /api/v1/devices/{id}", s.handleGetDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/detect", s.handleDetectDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/trust", s.handleSetDeviceTrust)

	// Monitoring read models.
	mux.HandleFunc("GET /api/v1/links", s.handleListLinks)
	mux.HandleFunc("GET /api/v1/alerts", s.handleListAlerts)
	mux.HandleFunc("GET /api/v1/dns/servers", s.handleListDNSServers)

	// Stateless planning/validation tools (preview before apply).
	mux.HandleFunc("POST /api/v1/tools/config-diff", s.handleConfigDiff)
	mux.HandleFunc("POST /api/v1/tools/routing/validate", s.handleRoutingValidate)
	mux.HandleFunc("POST /api/v1/tools/link-score", s.handleLinkScore)

	mws := []func(http.Handler) http.Handler{
		recoverer(s.logger),
		requestID,
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
