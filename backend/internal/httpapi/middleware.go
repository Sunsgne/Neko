package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neko/sdwan/backend/internal/auth"
	"github.com/neko/sdwan/backend/internal/metrics"
)

// instrument records request counts and latency into the metrics registry.
func instrument(reg *metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			labels := map[string]string{"method": r.Method, "status": strconv.Itoa(sw.status)}
			reg.IncCounter("neko_http_requests_total", labels)
			reg.AddCounter("neko_http_request_seconds_sum", map[string]string{"method": r.Method}, time.Since(start).Seconds())
		})
	}
}

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyTenantID  ctxKey = "tenant_id"
	ctxKeyPrincipal ctxKey = "principal"
)

// authenticate validates the bearer token and derives the tenant scope from
// the resulting principal. Operators may target any tenant via X-Tenant-Id;
// tenant principals are locked to their own tenant.
func authenticate(a auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Health, metrics and login/logout endpoints are always public.
			switch r.URL.Path {
			case "/healthz", "/readyz", "/metrics", "/api/v1/auth/login", "/api/v1/auth/logout":
				next.ServeHTTP(w, r)
				return
			}
			token := bearerToken(r)
			p, err := a.Authenticate(r.Context(), token)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			scope := p.Scope()
			if p.IsOperator {
				// Operator may scope to a specific tenant via header.
				scope = r.Header.Get("X-Tenant-Id")
			}
			ctx := context.WithValue(r.Context(), ctxKeyPrincipal, p)
			ctx = context.WithValue(ctx, ctxKeyTenantID, scope)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	// Fallback for SSE (EventSource cannot set headers): ?token=...
	return r.URL.Query().Get("token")
}

// requestID assigns a unique id to each request and exposes it via header.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// tenantScope extracts the tenant scope from the X-Tenant-Id header.
//
// Bootstrap behaviour: operator/admin callers omit the header (cross-tenant);
// tenant callers send their tenant id. Epic 1 replaces this with token-derived
// scoping and enforces authorization.
func tenantScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKeyTenantID, r.Header.Get("X-Tenant-Id"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func tenantFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTenantID).(string); ok {
		return v
	}
	return ""
}

func principalFrom(ctx context.Context) (auth.Principal, bool) {
	if v, ok := ctx.Value(ctxKeyPrincipal).(auth.Principal); ok {
		return v, true
	}
	return auth.Principal{}, false
}

// logging records method, path, status and duration for each request.
func logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", r.Context().Value(ctxKeyRequestID),
			)
		})
	}
}

// recoverer converts panics into 500 responses instead of crashing the server.
func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic", "recover", rec, "path", r.URL.Path)
					respondError(w, http.StatusInternalServerError, "internal", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// cors applies permissive CORS suitable for local development.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Id, X-Request-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func newID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "req"
	}
	return hex.EncodeToString(b)
}
