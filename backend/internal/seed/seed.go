// Package seed populates the in-memory store and catalog with realistic demo
// data so the console is fully interactive without real devices or a database.
// Enabled via NEKO_SEED=true.
package seed

import (
	"context"
	"time"

	"github.com/neko/sdwan/backend/internal/catalog"
	"github.com/neko/sdwan/backend/internal/store"
)

// Demo seeds tenants and devices into the store and links/alerts/DNS into the
// catalog. It is idempotent enough for a single startup call.
func Demo(ctx context.Context, st store.Store, cat *catalog.Catalog) error {
	now := time.Date(2026, 6, 14, 4, 0, 0, 0, time.UTC)

	tenants := []store.Tenant{
		{ID: "ten_acme", Name: "Acme Corp", Slug: "acme-corp", Status: store.TenantActive, CreatedAt: now, UpdatedAt: now},
		{ID: "ten_globex", Name: "Globex 物流", Slug: "globex", Status: store.TenantActive, CreatedAt: now, UpdatedAt: now},
		{ID: "ten_chuxin", Name: "初心科技", Slug: "chuxin-tech", Status: store.TenantActive, CreatedAt: now, UpdatedAt: now},
	}
	for i := range tenants {
		_ = st.Tenants().Create(ctx, &tenants[i])
	}

	seen := now.Add(-2 * time.Minute)
	devices := []store.Device{
		{
			ID: "dev_sh01", TenantID: "ten_acme", Name: "edge-sh-01", MgmtAddress: "10.10.1.1",
			Platform: store.PlatformRouterBOARD, Model: "CCR2004-1G-12S+2XS", Serial: "HEX0A1", TrustState: store.TrustManaged,
			Capabilities: &store.CapabilityMatrix{RouterOSVersion: "7.14.3", Architecture: "arm64", BoardName: "CCR2004-1G-12S+2XS", LicenseLevel: 6, DeviceMode: "enterprise", SupportsBGP: true, SupportsOSPF: true, SupportsWireGuard: true, SupportsContainer: true, Packages: []string{"routeros", "container"}},
			LastSeenAt:   &seen, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "dev_bj02", TenantID: "ten_acme", Name: "edge-bj-02", MgmtAddress: "10.20.1.1",
			Platform: store.PlatformX86, Model: "x86_64", Serial: "", TrustState: store.TrustManaged,
			Capabilities: &store.CapabilityMatrix{RouterOSVersion: "7.13", Architecture: "x86_64", BoardName: "x86", LicenseLevel: 0, DeviceMode: "enterprise", SupportsBGP: true, SupportsOSPF: true, SupportsWireGuard: true},
			LastSeenAt:   &seen, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "dev_gzcore", TenantID: "ten_globex", Name: "pop-gz-core", MgmtAddress: "10.30.0.1",
			Role: store.RoleBackbone, Region: "cn-south",
			Platform: store.PlatformCHR, Model: "CHR", Serial: "", TrustState: store.TrustManaged,
			Capabilities: &store.CapabilityMatrix{RouterOSVersion: "7.14", Architecture: "x86_64", BoardName: "CHR", LicenseLevel: 4, DeviceMode: "enterprise", SupportsBGP: true, SupportsOSPF: true, SupportsWireGuard: true},
			LastSeenAt:   &seen, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "dev_popsh", TenantID: "ten_acme", Name: "pop-sh-core", MgmtAddress: "10.10.0.1",
			Role: store.RoleBackbone, Region: "cn-east",
			Platform: store.PlatformRouterBOARD, Model: "CCR2216-1G-12XS-2XQ", Serial: "HEXBB1", TrustState: store.TrustManaged,
			Capabilities: &store.CapabilityMatrix{RouterOSVersion: "7.14.3", Architecture: "arm64", BoardName: "CCR2216-1G-12XS-2XQ", LicenseLevel: 6, DeviceMode: "enterprise", SupportsBGP: true, SupportsOSPF: true, SupportsWireGuard: true, SupportsContainer: true},
			LastSeenAt:   &seen, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "dev_gwhk", TenantID: "ten_acme", Name: "gw-hk-exit", MgmtAddress: "10.200.0.1",
			Role: store.RoleGateway, Region: "overseas-hk",
			Platform: store.PlatformCHR, Model: "CHR", Serial: "", TrustState: store.TrustManaged,
			Capabilities: &store.CapabilityMatrix{RouterOSVersion: "7.14", Architecture: "x86_64", BoardName: "CHR", LicenseLevel: 6, DeviceMode: "enterprise", SupportsBGP: true, SupportsOSPF: true, SupportsWireGuard: true},
			LastSeenAt:   &seen, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "dev_cpe01", TenantID: "ten_acme", Name: "cpe-acme-001", MgmtAddress: "192.168.88.1",
			Platform: store.PlatformRouterBOARD, Model: "hAP ax3", Serial: "HEX9Z2", TrustState: store.TrustAuthenticated,
			Capabilities: &store.CapabilityMatrix{RouterOSVersion: "7.14.2", Architecture: "arm", BoardName: "hAP ax3", LicenseLevel: 4, DeviceMode: "home", SupportsBGP: true, SupportsOSPF: true, SupportsWireGuard: true},
			LastSeenAt:   &seen, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "dev_cpe14", TenantID: "ten_chuxin", Name: "cpe-cx-014", MgmtAddress: "192.168.88.14",
			Platform: store.PlatformRouterBOARD, Model: "RB5009UG+S+IN", Serial: "HEX7Q8", TrustState: store.TrustDiscovered,
			LastSeenAt: &seen, CreatedAt: now, UpdatedAt: now,
		},
		{
			// Points at the bundled RouterOS simulator (compose service "rosim").
			// Click 纳管 in the UI (any username/password) to manage it live.
			ID: "dev_sim01", TenantID: "ten_acme", Name: "sim-edge-01", MgmtAddress: "rosim:8729",
			Role: store.RoleCPE, Region: "lab",
			Platform: store.PlatformUnknown, TrustState: store.TrustDiscovered,
			CreatedAt: now, UpdatedAt: now,
		},
	}
	for i := range devices {
		_ = st.Devices().Create(ctx, &devices[i])
	}

	cat.ReplaceLinks([]catalog.Link{
		{ID: "lnk_sh_tel", TenantID: "ten_acme", Name: "上海-电信", Kind: "wan", ISP: "telecom", Role: "primary", Status: "up", LatencyMs: 8, JitterMs: 2, Loss: 0, Score: 98},
		{ID: "lnk_sh_uni", TenantID: "ten_acme", Name: "上海-联通", Kind: "wan", ISP: "unicom", Role: "backup", Status: "up", LatencyMs: 14, JitterMs: 4, Loss: 0.002, Score: 92},
		{ID: "lnk_bj_mob", TenantID: "ten_acme", Name: "北京-移动", Kind: "wan", ISP: "mobile", Role: "primary", Status: "degraded", LatencyMs: 46, JitterMs: 18, Loss: 0.012, Score: 71},
		{ID: "lnk_gz_tel", TenantID: "ten_globex", Name: "广州-电信", Kind: "wan", ISP: "telecom", Role: "primary", Status: "down", LatencyMs: 220, JitterMs: 80, Loss: 0.045, Score: 38},
		{ID: "lnk_ov_shbj", TenantID: "ten_acme", Name: "Overlay 上海↔北京", Kind: "overlay", ISP: "", Role: "primary", Status: "up", LatencyMs: 11, JitterMs: 3, Loss: 0, Score: 96},
	})

	// Seed persisted alerts (the live source after monitoring runs).
	for _, a := range []store.Alert{
		{ID: "al_seed1", TenantID: "ten_globex", DeviceID: "dev_gzcore", Code: "link_down", Severity: "critical", Title: "广州-电信 链路中断", Detail: "丢包 4.5%，已本地切换备线"},
		{ID: "al_seed2", TenantID: "ten_acme", DeviceID: "dev_bj02", Code: "iface_util", Severity: "warning", Title: "edge-bj-02 sfp1 利用率 > 85%", Detail: "持续 6 分钟"},
	} {
		_, _, _ = st.Alerts().Fire(ctx, a)
	}

	cat.ReplaceAlerts([]catalog.Alert{
		{ID: "al_1", TenantID: "ten_globex", DeviceID: "dev_gzcore", Severity: "critical", Title: "广州-电信 链路中断", Detail: "丢包 4.5%，评分 38，已本地切换备线", State: "firing", FiredAt: "2026-06-14T03:51:00Z"},
		{ID: "al_2", TenantID: "ten_acme", DeviceID: "dev_bj02", Severity: "warning", Title: "edge-bj-02 sfp1 利用率 > 85%", Detail: "持续 6 分钟", State: "firing", FiredAt: "2026-06-14T03:58:00Z"},
		{ID: "al_3", TenantID: "ten_acme", DeviceID: "dev_sh01", Severity: "info", Title: "BGP 策略 v7 已确认", Detail: "commit-confirm 成功", State: "resolved", FiredAt: "2026-06-14T03:33:00Z"},
	})

	// Persisted DNS pool (manageable + deliverable from the DNS page).
	for _, d := range []store.DNSServer{
		{ID: "dns_tel_sh", Address: "202.96.209.133", Region: "shanghai", ISP: "telecom", SupportsECS: true, Healthy: true, LatencyMs: 5},
		{ID: "dns_uni_bj", Address: "123.123.123.123", Region: "beijing", ISP: "unicom", Healthy: true, LatencyMs: 8},
		{ID: "dns_mob_gz", Address: "211.136.192.6", Region: "guangzhou", ISP: "mobile", Healthy: true, LatencyMs: 12},
		{ID: "dns_ali", Address: "223.5.5.5", Region: "", ISP: "public", SupportsECS: true, Healthy: true, LatencyMs: 6},
		{ID: "dns_114", Address: "114.114.114.114", Region: "", ISP: "public", Healthy: true, LatencyMs: 10},
	} {
		_ = st.Dns().Create(ctx, d)
	}

	return nil
}
