import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import { listDevices, type Device } from "@/lib/api";
import { serverToken } from "@/lib/server-session";
import { RegisterDeviceButton } from "@/components/register-device";

export const dynamic = "force-dynamic";

const demo: Device[] = [
  { id: "1", tenant_id: "", name: "pop-sh-core", mgmt_address: "10.10.0.1", role: "backbone", region: "cn-east", platform: "routerboard", model: "CCR2216-1G-12XS-2XQ", serial: "HEXBB1", trust_state: "managed", created_at: "", updated_at: "" },
  { id: "2", tenant_id: "", name: "pop-gz-core", mgmt_address: "10.30.0.1", role: "backbone", region: "cn-south", platform: "chr", model: "CHR", serial: "", trust_state: "managed", created_at: "", updated_at: "" },
  { id: "3", tenant_id: "", name: "gw-hk-exit", mgmt_address: "10.200.0.1", role: "gateway", region: "overseas-hk", platform: "chr", model: "CHR", serial: "", trust_state: "managed", created_at: "", updated_at: "" },
];

const platformTone: Record<string, "primary" | "success" | "warning" | "neutral"> = {
  routerboard: "primary", chr: "success", x86: "warning", unknown: "neutral",
};

export default async function BackbonePage() {
  let nodes: Device[] = [];
  let live = false;
  try {
    const [bb, gw] = await Promise.all([
      listDevices(serverToken(), "backbone"),
      listDevices(serverToken(), "gateway"),
    ]);
    nodes = [...bb, ...gw];
    live = true;
  } catch {
    nodes = [];
  }
  if (nodes.length === 0 && !live) nodes = demo;

  const backbone = nodes.filter((n) => n.role === "backbone");
  const gateways = nodes.filter((n) => n.role === "gateway");

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">骨干节点管理</h1>
          <p className="mt-1 text-sm text-muted">
            SD-WAN 骨干节点 / POP 与出口网关（均为 RouterOS，统一纳管）— {backbone.length} 骨干 · {gateways.length} 出口
          </p>
        </div>
        <div className="flex gap-2">
          <RegisterDeviceButton role="backbone" label="登记骨干节点" />
          <RegisterDeviceButton role="gateway" label="登记出口网关" />
        </div>
      </div>

      <Card className="p-0">
        <CardHeader title="节点清单" subtitle={`${nodes.length} 个节点`} />
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                <th className="px-5 py-3 font-medium">名称</th>
                <th className="px-5 py-3 font-medium">角色</th>
                <th className="px-5 py-3 font-medium">地域</th>
                <th className="px-5 py-3 font-medium">管理地址</th>
                <th className="px-5 py-3 font-medium">平台 / 型号</th>
                <th className="px-5 py-3 font-medium">状态</th>
              </tr>
            </thead>
            <tbody>
              {nodes.map((n) => (
                <tr key={n.id} className="border-b border-border/60 transition-colors hover:bg-elevated/40">
                  <td className="px-5 py-3 font-medium">{n.name}</td>
                  <td className="px-5 py-3">
                    <Badge tone={n.role === "backbone" ? "primary" : "warning"}>
                      {n.role === "backbone" ? "骨干 POP" : "出口网关"}
                    </Badge>
                  </td>
                  <td className="px-5 py-3 text-muted">{n.region || "—"}</td>
                  <td className="px-5 py-3 font-mono text-xs text-muted">{n.mgmt_address}</td>
                  <td className="px-5 py-3">
                    <Badge tone={platformTone[n.platform]}>{n.platform}</Badge>
                    <span className="ml-2 text-muted">{n.model || "—"}</span>
                  </td>
                  <td className="px-5 py-3">
                    <span className="inline-flex items-center gap-2">
                      <StatusDot tone={n.trust_state === "managed" ? "success" : "warning"} />
                      {n.trust_state}
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
