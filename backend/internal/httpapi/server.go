// Package httpapi wires HTTP routes, middleware and handlers for the Neko API.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/tenant"
)

// Server holds dependencies for the HTTP API.
type Server struct {
	logger    *slog.Logger
	tenants   *tenant.Service
	inventory *inventory.Service
	storeKind string
}

// Deps are the dependencies required to build the server.
type Deps struct {
	Logger    *slog.Logger
	Tenants   *tenant.Service
	Inventory *inventory.Service
	StoreKind string
}

// New builds a Server.
func New(d Deps) *Server {
	return &Server{
		logger:    d.Logger,
		tenants:   d.Tenants,
		inventory: d.Inventory,
		storeKind: d.StoreKind,
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

	return chain(mux,
		recoverer(s.logger),
		requestID,
		logging(s.logger),
		cors,
		tenantScope,
	)
}
