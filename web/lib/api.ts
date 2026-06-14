// API client and shared types for the Neko control plane.
// Mirrors backend/internal/store/models.go and docs/API.md.

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type TenantStatus = "active" | "suspended";

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: TenantStatus;
  created_at: string;
  updated_at: string;
}

export type DevicePlatform = "routerboard" | "chr" | "x86" | "unknown";

export type TrustState =
  | "untrusted"
  | "discovered"
  | "authenticated"
  | "enrolled"
  | "managed";

export interface Device {
  id: string;
  tenant_id: string;
  name: string;
  mgmt_address: string;
  platform: DevicePlatform;
  model: string;
  serial: string;
  trust_state: TrustState;
  last_seen_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Link {
  id: string;
  tenant_id: string;
  name: string;
  kind: "wan" | "overlay";
  isp: string;
  role: "primary" | "backup";
  status: string;
  latency_ms: number;
  jitter_ms: number;
  loss: number;
  score: number;
}

export interface Alert {
  id: string;
  tenant_id: string;
  device_id: string;
  severity: "info" | "warning" | "critical";
  title: string;
  detail: string;
  state: "firing" | "resolved";
  fired_at: string;
}

export interface DNSServer {
  id: string;
  tenant_id: string;
  address: string;
  region: string;
  isp: string;
  supports_ecs: boolean;
  healthy: boolean;
  latency_ms: number;
}

export interface Envelope<T> {
  data: T;
  meta?: { page: number; page_size: number; total: number };
}

async function getJSON<T>(path: string, tenantId?: string): Promise<Envelope<T>> {
  const headers: Record<string, string> = {};
  if (tenantId) headers["X-Tenant-Id"] = tenantId;
  const res = await fetch(`${API_BASE_URL}${path}`, {
    headers,
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`request failed: ${res.status}`);
  return res.json();
}

export async function listTenants(): Promise<Tenant[]> {
  const env = await getJSON<Tenant[]>("/api/v1/tenants");
  return env.data ?? [];
}

export async function listDevices(tenantId?: string): Promise<Device[]> {
  const env = await getJSON<Device[]>("/api/v1/devices", tenantId);
  return env.data ?? [];
}

export async function listLinks(tenantId?: string): Promise<Link[]> {
  const env = await getJSON<Link[]>("/api/v1/links", tenantId);
  return env.data ?? [];
}

export async function listAlerts(tenantId?: string): Promise<Alert[]> {
  const env = await getJSON<Alert[]>("/api/v1/alerts", tenantId);
  return env.data ?? [];
}

export async function listDNSServers(tenantId?: string): Promise<DNSServer[]> {
  const env = await getJSON<DNSServer[]>("/api/v1/dns/servers", tenantId);
  return env.data ?? [];
}
