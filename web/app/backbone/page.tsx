import { Card } from "@/components/ui";
import { listDevices, type Device } from "@/lib/api";
import { serverToken } from "@/lib/server-session";
import { RegisterDeviceButton } from "@/components/register-device";
import { DeviceTable } from "@/components/device-table";

export const dynamic = "force-dynamic";

export default async function BackbonePage() {
  let nodes: Device[] = [];
  try {
    const [bb, gw] = await Promise.all([
      listDevices(serverToken(), "backbone"),
      listDevices(serverToken(), "gateway"),
    ]);
    nodes = [...bb, ...gw];
  } catch {
    nodes = [];
  }

  const backbone = nodes.filter((n) => n.role === "backbone").length;
  const gateways = nodes.filter((n) => n.role === "gateway").length;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">骨干节点管理</h1>
          <p className="mt-1 text-sm text-muted">
            SD-WAN 骨干节点 / POP 与出口网关（均为 RouterOS，统一纳管）— {backbone} 骨干 · {gateways} 出口
          </p>
        </div>
        <div className="flex gap-2">
          <RegisterDeviceButton role="backbone" label="登记骨干节点" />
          <RegisterDeviceButton role="gateway" label="登记出口网关" />
        </div>
      </div>

      <Card className="p-0">
        {nodes.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 py-16 text-center">
            <p className="text-sm text-muted">还没有骨干节点 / 出口网关</p>
            <p className="text-xs text-muted">点击右上角登记，或在「发现纳管」中批量添加</p>
          </div>
        ) : (
          <DeviceTable devices={nodes} />
        )}
      </Card>
    </div>
  );
}
