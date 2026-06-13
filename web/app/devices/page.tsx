import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import { listDevices, type Device, type TrustState, type DevicePlatform } from "@/lib/api";

export const dynamic = "force-dynamic";

const demo: Device[] = [
  mk("edge-sh-01", "10.10.1.1", "routerboard", "managed", "CCR2004-1G-12S+2XS"),
  mk("edge-bj-02", "10.20.1.1", "x86", "managed", "x86_64 CHR-host"),
  mk("pop-gz-core", "10.30.0.1", "chr", "enrolled", "CHR"),
  mk("cpe-acme-001", "192.168.88.1", "routerboard", "authenticated", "hAP ax3"),
  mk("cpe-acme-014", "192.168.88.14", "routerboard", "discovered", "RB5009UG+S+IN"),
];

function mk(name: string, addr: string, platform: DevicePlatform, trust: TrustState, model: string): Device {
  const now = new Date().toISOString();
  return {
    id: name,
    tenant_id: "acme-corp",
    name,
    mgmt_address: addr,
    platform,
    model,
    serial: "",
    trust_state: trust,
    created_at: now,
    updated_at: now,
  };
}

const platformTone: Record<DevicePlatform, "primary" | "success" | "warning" | "neutral"> = {
  routerboard: "primary",
  chr: "success",
  x86: "warning",
  unknown: "neutral",
};

const trustTone: Record<TrustState, "neutral" | "primary" | "warning" | "success"> = {
  untrusted: "neutral",
  discovered: "warning",
  authenticated: "primary",
  enrolled: "primary",
  managed: "success",
};

export default async function DevicesPage() {
  let devices: Device[] = [];
  try {
    devices = await listDevices();
  } catch {
    devices = [];
  }
  if (devices.length === 0) devices = demo;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">设备纳管</h1>
        <p className="mt-1 text-sm text-muted">
          自动识别 RouterBOARD / CHR / x86，建立能力矩阵 · 基于能力而非型号下发
        </p>
      </div>

      <Card className="p-0">
        <CardHeader title="设备清单" subtitle={`${devices.length} 台设备`} />
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                <th className="px-5 py-3 font-medium">名称</th>
                <th className="px-5 py-3 font-medium">管理地址</th>
                <th className="px-5 py-3 font-medium">平台</th>
                <th className="px-5 py-3 font-medium">型号</th>
                <th className="px-5 py-3 font-medium">信任状态</th>
              </tr>
            </thead>
            <tbody>
              {devices.map((d) => (
                <tr key={d.id} className="border-b border-border/60 transition-colors hover:bg-elevated/40">
                  <td className="px-5 py-3 font-medium">{d.name}</td>
                  <td className="px-5 py-3 font-mono text-xs text-muted">{d.mgmt_address}</td>
                  <td className="px-5 py-3">
                    <Badge tone={platformTone[d.platform]}>{d.platform}</Badge>
                  </td>
                  <td className="px-5 py-3 text-muted">{d.model || "—"}</td>
                  <td className="px-5 py-3">
                    <span className="inline-flex items-center gap-2">
                      <StatusDot tone={trustTone[d.trust_state]} />
                      {d.trust_state}
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
