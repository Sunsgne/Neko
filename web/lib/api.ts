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
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers: authHeaders(opts.token),
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    cache: "no-store",
  });
  const text = await res.text();
  const json = text ? JSON.parse(text) : {};
  if (!res.ok) {
    const err = json?.error ?? {};
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

export async function listDevices(token?: string): Promise<Device[]> {
  const env = await request<Device[]>("GET", "/api/v1/devices", { token });
  return env.data ?? [];
}

export async function registerDevice(name: string, mgmtAddress: string, token?: string): Promise<Device> {
  const env = await request<Device>("POST", "/api/v1/devices", {
    token,
    body: { name, mgmt_address: mgmtAddress },
  });
  return env.data;
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
