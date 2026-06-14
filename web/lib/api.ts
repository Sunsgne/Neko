// API client and shared types for the Neko control plane.
// Mirrors backend/internal/store/models.go and docs/API.md.

// Browser-facing API base (inlined at build for client bundles).
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

// Resolve the base URL per execution context: the browser must use the public
// URL, while server-side (RSC, route handlers) prefers the internal service
// URL (e.g. http://api:8080) to avoid NAT hairpin and reduce latency.
function baseURL(): string {
  if (typeof window === "undefined") {
    return process.env.NEKO_API_INTERNAL_URL || API_BASE_URL;
  }
  return API_BASE_URL;
}

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

export type DeviceRole = "cpe" | "backbone" | "gateway";

export interface DeviceStatus {
  online: boolean;
  version?: string;
  uptime?: string;
  cpu_load_percent: number;
  free_memory_bytes: number;
  total_memory_bytes: number;
  board_temp_c?: number;
  interfaces_up: number;
  interfaces_total: number;
  last_polled_at?: string | null;
  last_error?: string;
}

export interface InterfaceCapability {
  name: string;
  type: string;
  speed_mbps: number;
  features: string[];
}

export interface CapabilityMatrix {
  routeros_version: string;
  architecture: string;
  board_name: string;
  packages: string[];
  license_level: number;
  device_mode: string;
  interfaces: InterfaceCapability[];
  supports_bgp: boolean;
  supports_ospf: boolean;
  supports_wireguard: boolean;
  supports_container: boolean;
}

export interface Device {
  id: string;
  tenant_id: string;
  name: string;
  mgmt_address: string;
  role: DeviceRole;
  region?: string;
  platform: DevicePlatform;
  model: string;
  serial: string;
  trust_state: TrustState;
  enrolled: boolean;
  capabilities?: CapabilityMatrix | null;
  status?: DeviceStatus | null;
  last_seen_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface AccelMode {
  mode: string;
  desc: string;
}

export interface AccelPreview {
  mode: string;
  desc: string;
  state: { statements: Array<{ path: string; key: string; attributes: Record<string, string> }> };
  plan: { changes: Array<{ type: string; path: string; key: string; risk: string }>; aggregate_risk: string };
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

export interface AuthUser {
  id: string;
  email: string;
  display_name: string;
  tenant_id: string;
  is_operator: boolean;
}

export interface Envelope<T> {
  data: T;
  meta?: { page: number; page_size: number; total: number };
}

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

function authHeaders(token?: string): Record<string, string> {
  const h: Record<string, string> = { "Content-Type": "application/json" };
  if (token) h["Authorization"] = `Bearer ${token}`;
  return h;
}

async function request<T>(
  method: string,
  path: string,
  opts: { token?: string; body?: unknown } = {},
): Promise<Envelope<T>> {
  const res = await fetch(`${baseURL()}${path}`, {
    method,
    headers: authHeaders(opts.token),
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    cache: "no-store",
  });
  const text = await res.text();
  const json = text ? JSON.parse(text) : {};
  if (!res.ok) {
    const err = json?.error ?? {};
    // Stale/expired token: clear the client session and bounce to login so the
    // user re-authenticates instead of seeing "missing or invalid token".
    if (res.status === 401 && typeof window !== "undefined") {
      for (const c of ["neko_token", "neko_email", "neko_role"]) {
        document.cookie = `${c}=; path=/; max-age=0`;
      }
      const next = encodeURIComponent(window.location.pathname);
      window.location.href = `/login?next=${next}`;
    }
    throw new ApiError(res.status, err.code ?? "error", err.message ?? `request failed: ${res.status}`);
  }
  return json as Envelope<T>;
}

export async function login(email: string, password: string): Promise<{ token: string; user: AuthUser }> {
  const env = await request<{ token: string; user: AuthUser }>("POST", "/api/v1/auth/login", {
    body: { email, password },
  });
  return env.data;
}

export async function logout(token: string): Promise<void> {
  try {
    await request("POST", "/api/v1/auth/logout", { token });
  } catch {
    /* ignore */
  }
}

export async function listTenants(token?: string): Promise<Tenant[]> {
  const env = await request<Tenant[]>("GET", "/api/v1/tenants", { token });
  return env.data ?? [];
}

export async function createTenant(name: string, token?: string): Promise<Tenant> {
  const env = await request<Tenant>("POST", "/api/v1/tenants", { token, body: { name } });
  return env.data;
}

export async function listDevices(token?: string, role?: DeviceRole): Promise<Device[]> {
  const q = role ? `?role=${role}` : "";
  const env = await request<Device[]>("GET", `/api/v1/devices${q}`, { token });
  return env.data ?? [];
}

export async function registerDevice(
  input: { name: string; mgmt_address: string; role?: DeviceRole; region?: string },
  token?: string,
): Promise<Device> {
  const env = await request<Device>("POST", "/api/v1/devices", { token, body: input });
  return env.data;
}

export async function listAccelModes(token?: string): Promise<AccelMode[]> {
  const env = await request<AccelMode[]>("GET", "/api/v1/accel/modes", { token });
  return env.data ?? [];
}

export async function previewAccel(profile: Record<string, unknown>, token?: string): Promise<AccelPreview> {
  const env = await request<AccelPreview>("POST", "/api/v1/accel/preview", { token, body: profile });
  return env.data;
}

export async function listConfigSections(token?: string): Promise<string[]> {
  const env = await request<string[]>("GET", "/api/v1/config/sections", { token });
  return env.data ?? [];
}

export interface Uplink {
  name: string;
  gateway: string;
  interface?: string;
  priority?: number;
  weight?: number;
}

export interface OrchestrateResult {
  dry_run?: boolean;
  desired?: { statements: Array<{ path: string; key: string; attributes: Record<string, string> }> };
  plan?: { changes: Array<{ type: string; path: string; key: string; risk: string }>; aggregate_risk: string };
  result?: { status: string; rolled_back?: boolean; reason?: string };
  error?: string;
}

export async function orchestrate(
  deviceId: string,
  body: Record<string, unknown>,
  token?: string,
): Promise<OrchestrateResult> {
  const env = await request<OrchestrateResult>("POST", `/api/v1/devices/${deviceId}/orchestrate`, { token, body });
  return env.data;
}

export async function getDevice(id: string, token?: string): Promise<Device> {
  const env = await request<Device>("GET", `/api/v1/devices/${id}`, { token });
  return env.data;
}

export async function enrollDevice(id: string, username: string, password: string, token?: string): Promise<Device> {
  const env = await request<Device>("POST", `/api/v1/devices/${id}/enroll`, {
    token, body: { username, password },
  });
  return env.data;
}

export async function pollDevice(id: string, token?: string): Promise<Device> {
  const env = await request<Device>("POST", `/api/v1/devices/${id}/poll`, { token });
  return env.data;
}

export interface MetricSeries {
  name: string;
  points: Array<{ t: number; v: number }>;
}

export interface DeviceMetrics {
  enabled: boolean;
  series: MetricSeries[];
}

export async function getDeviceMetrics(id: string, token?: string): Promise<DeviceMetrics> {
  const env = await request<DeviceMetrics>("GET", `/api/v1/devices/${id}/metrics`, { token });
  return env.data;
}

export interface ConfigSnapshot {
  id: string;
  device_id: string;
  source: string;
  statement_count: number;
  taken_at: string;
}

export async function listSnapshots(id: string, token?: string): Promise<ConfigSnapshot[]> {
  const env = await request<ConfigSnapshot[]>("GET", `/api/v1/devices/${id}/snapshots`, { token });
  return env.data ?? [];
}

export async function takeSnapshot(id: string, token?: string): Promise<ConfigSnapshot> {
  const env = await request<ConfigSnapshot>("POST", `/api/v1/devices/${id}/snapshot`, { token });
  return env.data;
}

export interface Candidate {
  address: string;
  board: string;
  version: string;
  arch: string;
}

export async function discover(
  body: { cidr: string; port: number; username: string; password: string },
  token?: string,
): Promise<Candidate[]> {
  const env = await request<Candidate[]>("POST", "/api/v1/discover", { token, body });
  return env.data ?? [];
}

export interface BatchResult {
  created: number;
  enrolled: number;
  results: Array<{ name: string; device_id?: string; enrolled: boolean; error?: string }>;
}

export async function batchOnboard(
  body: { devices: Array<{ name: string; mgmt_address: string; role?: DeviceRole; region?: string }>; username?: string; password?: string },
  token?: string,
): Promise<BatchResult> {
  const env = await request<BatchResult>("POST", "/api/v1/devices/batch", { token, body });
  return env.data;
}

export interface AuditEntry {
  id: string;
  tenant_id: string;
  actor_id: string;
  action: string;
  object_type: string;
  object_id: string;
  at: string;
}

export async function listAudit(token?: string): Promise<AuditEntry[]> {
  const env = await request<AuditEntry[]>("GET", "/api/v1/audit", { token });
  return env.data ?? [];
}

export async function listLinks(token?: string): Promise<Link[]> {
  const env = await request<Link[]>("GET", "/api/v1/links", { token });
  return env.data ?? [];
}

export async function listAlerts(token?: string): Promise<Alert[]> {
  const env = await request<Alert[]>("GET", "/api/v1/alerts", { token });
  return env.data ?? [];
}

export async function listDNSServers(token?: string): Promise<DNSServer[]> {
  const env = await request<DNSServer[]>("GET", "/api/v1/dns/servers", { token });
  return env.data ?? [];
}
