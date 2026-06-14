import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import { listTenants, type Tenant } from "@/lib/api";
import { serverToken } from "@/lib/server-session";
import { AddTenantButton } from "@/components/add-tenant";

export const dynamic = "force-dynamic";

const demo: Tenant[] = [
  mk("Acme Corp", "acme-corp"),
  mk("Globex 物流", "globex"),
  mk("初心科技", "chuxin-tech"),
];

function mk(name: string, slug: string): Tenant {
  const now = new Date().toISOString();
  return { id: slug, name, slug, status: "active", created_at: now, updated_at: now };
}

export default async function TenantsPage() {
  let tenants: Tenant[] = [];
  let live = false;
  try {
    tenants = await listTenants(serverToken());
    live = true;
  } catch {
    tenants = [];
  }
  if (tenants.length === 0 && !live) tenants = demo;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">租户管理</h1>
          <p className="mt-1 text-sm text-muted">运营端 · 多租户严格隔离 (PostgreSQL RLS)</p>
        </div>
        <AddTenantButton />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {tenants.map((t) => (
          <Card key={t.id}>
            <CardHeader
              title={t.name}
              subtitle={`/${t.slug}`}
              action={
                <Badge tone={t.status === "active" ? "success" : "warning"}>
                  <StatusDot tone={t.status === "active" ? "success" : "warning"} />
                  {t.status}
                </Badge>
              }
            />
            <div className="grid grid-cols-3 gap-2 text-center">
              <Metric label="站点" value="—" />
              <Metric label="设备" value="—" />
              <Metric label="告警" value="0" />
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-elevated/50 py-2">
      <div className="text-lg font-semibold">{value}</div>
      <div className="text-[10px] uppercase tracking-wide text-muted">{label}</div>
    </div>
  );
}
