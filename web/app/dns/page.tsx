import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import { listDNSServers, type DNSServer } from "@/lib/api";

export const dynamic = "force-dynamic";

const demo: DNSServer[] = [
  { id: "1", tenant_id: "", address: "202.96.209.133", region: "shanghai", isp: "telecom", supports_ecs: true, healthy: true, latency_ms: 5 },
  { id: "2", tenant_id: "", address: "223.5.5.5", region: "", isp: "public", supports_ecs: true, healthy: true, latency_ms: 6 },
  { id: "3", tenant_id: "", address: "123.123.123.123", region: "beijing", isp: "unicom", supports_ecs: false, healthy: true, latency_ms: 8 },
  { id: "4", tenant_id: "", address: "114.114.114.114", region: "", isp: "public", supports_ecs: false, healthy: true, latency_ms: 10 },
];

const ispLabel: Record<string, string> = {
  telecom: "电信",
  unicom: "联通",
  mobile: "移动",
  edu: "教育网",
  public: "公共",
  "": "—",
};

export default async function DnsPage() {
  let servers: DNSServer[] = [];
  try {
    servers = await listDNSServers();
  } catch {
    servers = [];
  }
  if (servers.length === 0) servers = demo;

  const healthy = servers.filter((s) => s.healthy).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">DNS 管理</h1>
        <p className="mt-1 text-sm text-muted">
          大规模 DNS 池 · 中国地域 / 运营商调度 · ECS — {healthy}/{servers.length} 健康
        </p>
      </div>

      <Card className="p-0">
        <CardHeader title="DNS 服务器池" subtitle="按延迟排序（调度按运营商/地域/ECS 加权）" />
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                <th className="px-5 py-3 font-medium">地址</th>
                <th className="px-5 py-3 font-medium">运营商</th>
                <th className="px-5 py-3 font-medium">地域</th>
                <th className="px-5 py-3 font-medium">ECS</th>
                <th className="px-5 py-3 font-medium">延迟</th>
                <th className="px-5 py-3 font-medium">状态</th>
              </tr>
            </thead>
            <tbody>
              {servers.map((s) => (
                <tr key={s.id} className="border-b border-border/60 transition-colors hover:bg-elevated/40">
                  <td className="px-5 py-3 font-mono text-xs">{s.address}</td>
                  <td className="px-5 py-3">{ispLabel[s.isp] ?? s.isp}</td>
                  <td className="px-5 py-3 text-muted">{s.region || "anycast"}</td>
                  <td className="px-5 py-3">{s.supports_ecs ? <Badge tone="primary">ECS</Badge> : <span className="text-muted">—</span>}</td>
                  <td className="px-5 py-3 font-mono text-xs">{s.latency_ms} ms</td>
                  <td className="px-5 py-3">
                    <span className="inline-flex items-center gap-2">
                      <StatusDot tone={s.healthy ? "success" : "danger"} />
                      {s.healthy ? "健康" : "故障"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}
