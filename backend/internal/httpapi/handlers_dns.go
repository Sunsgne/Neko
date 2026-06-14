package httpapi

import (
	"net/http"
	"strings"

	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/dns"
	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/store"
)

type createDNSRequest struct {
	Address     string `json:"address"`
	Region      string `json:"region"`
	ISP         string `json:"isp"`
	SupportsECS bool   `json:"supports_ecs"`
	LatencyMs   int    `json:"latency_ms"`
}

// handleCreateDNSServer adds a DNS server to the pool.
func (s *Server) handleCreateDNSServer(w http.ResponseWriter, r *http.Request) {
	var req createDNSRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if strings.TrimSpace(req.Address) == "" {
		respondError(w, http.StatusBadRequest, "invalid_input", "address is required")
		return
	}
	srv := store.DNSServer{
		ID:          s.idgen("dns"),
		TenantID:    tenantFrom(r.Context()),
		Address:     strings.TrimSpace(req.Address),
		Region:      strings.TrimSpace(req.Region),
		ISP:         strings.TrimSpace(req.ISP),
		SupportsECS: req.SupportsECS,
		Healthy:     true,
		LatencyMs:   req.LatencyMs,
	}
	if err := s.dns.Create(r.Context(), srv); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "create", "dns_server", srv.ID, map[string]string{"address": srv.Address})
	respondData(w, http.StatusCreated, srv)
}

// handleDeleteDNSServer removes a DNS server from the pool.
func (s *Server) handleDeleteDNSServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.dns.Delete(r.Context(), tenantFrom(r.Context()), id); err != nil {
		respondServiceError(w, err)
		return
	}
	s.record(r.Context(), "delete", "dns_server", id, nil)
	respondData(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

type dnsApplyRequest struct {
	ServerAddresses []string `json:"server_addresses"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	DryRun          bool     `json:"dry_run"`
}

// handleDNSApply generates the /ip/dns config from the selected servers and
// either previews it (dry_run) or delivers it to the device over REST.
func (s *Server) handleDNSApply(w http.ResponseWriter, r *http.Request) {
	dev, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	var req dnsApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if len(req.ServerAddresses) == 0 {
		respondError(w, http.StatusBadRequest, "empty", "请至少选择一个 DNS 服务器")
		return
	}
	primary := make([]dns.Server, 0, len(req.ServerAddresses))
	for _, a := range req.ServerAddresses {
		primary = append(primary, dns.Server{Address: a})
	}
	desired := dns.BuildConfig(primary, nil)

	if req.DryRun {
		plan := configengine.ComputeDiff(configengine.State{}, desired, configengine.RiskOptions{})
		respondData(w, http.StatusOK, map[string]any{"dry_run": true, "desired": desired, "plan": plan})
		return
	}

	applier := routeros.NewApplier(routeros.Target{Address: dev.MgmtAddress, Username: req.Username, Secret: req.Password}, nil)
	res, plan, err := configengine.Execute(r.Context(), applier, nil, desired, configengine.ApplyOptions{})
	if err != nil {
		respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan, "error": err.Error()})
		return
	}
	s.record(r.Context(), "dns_apply", "device", dev.ID, map[string]string{"servers": strings.Join(req.ServerAddresses, ",")})
	respondData(w, http.StatusOK, map[string]any{"result": res, "plan": plan})
}
