import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import { getDevice, type Device } from "@/lib/api";
import { serverToken } from "@/lib/server-session";
import { DeviceActions } from "@/components/device-actions";

export const dynamic = "force-dynamic";

function bytes(n?: number): string {
  if (!n) return "—";
  const mb = n / (1024 * 1024);
  return mb >= 1024 ? `${(mb / 1024).toFixed(1)} GiB` : `${Math.round(mb)} MiB`;
}

export default async function DeviceDetailPage({ params }: { params: { id: string } }) {
  let d: Device | null = null;
  let err: string | null = null;
  try {
    d = await getDevice(params.id, serverToken());
  } catch (e) {
    err = e instanceof Error ? e.message : "加载失败";
  }

  if (!d) {
    return (
      <div className="space-y-4">
        <Link href="/devices" className="inline-flex items-center gap-1 text-sm text-muted hover:text-foreground"><ArrowLeft className="h-4 w-4" /> 返回设备</Link>
        <Card><p className="text-sm text-danger">无法加载设备：{err}</p></Card>
      </div>
    );
  }

  const st = d.status;
  const cap = d.capabilities;
  const memPct = st && st.total_memory_bytes > 0 ? Math.round((1 - st.free_memory_bytes / st.total_memory_bytes) * 100) : null;

  return (
    <div className="space-y-6">
      <Link href="/devices" className="inline-flex items-center gap-1 text-sm text-muted hover:text-foreground"><ArrowLeft className="h-4 w-4" /> 返回设备</Link>

      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{d.name}</h1>
          <p className="mt-1 flex items-center gap-2 text-sm text-muted">
            <span className="font-mono">{d.mgmt_address}</span>
            <Badge tone="primary">{d.role}</Badge>
            {d.region && <span>{d.region}</span>}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {d.enrolled ? (
            <Badge tone={st?.online ? "success" : "danger"}><StatusDot tone={st?.online ? "success" : "danger"} />{st?.online ? "在线" : "离线"}</Badge>
          ) : (
            <Badge tone="warning">未纳管</Badge>
          )}
          <DeviceActions deviceId={d.id} enrolled={d.enrolled} />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader title="身份" />
          <dl className="space-y-2 text-sm">
            <Row k="平台" v={<Badge tone="primary">{d.platform}</Badge>} />
            <Row k="型号" v={d.model || "—"} />
            <Row k="序列号" v={<span className="font-mono text-xs">{d.serial || "—"}</span>} />
            <Row k="信任状态" v={d.trust_state} />
            <Row k="托管" v={d.enrolled ? "是（凭据已加密保存）" : "否"} />
          </dl>
        </Card>

        <Card>
          <CardHeader title="实时状态" subtitle={st?.last_polled_at ? `最近轮询 ${new Date(st.last_polled_at).toLocaleString()}` : "尚未轮询"} />
          {st ? (
            <dl className="space-y-2 text-sm">
              <Row k="在线" v={st.online ? "在线" : `离线${st.last_error ? "（" + st.last_error + "）" : ""}`} />
              <Row k="版本" v={<span className="font-mono text-xs">{st.version || "—"}</span>} />
              <Row k="运行时长" v={st.uptime || "—"} />
              <Row k="CPU 负载" v={`${st.cpu_load_percent}%`} />
              <Row k="内存使用" v={memPct != null ? `${memPct}%（空闲 ${bytes(st.free_memory_bytes)}）` : "—"} />
              <Row k="接口" v={`${st.interfaces_up}/${st.interfaces_total} up`} />
              {st.board_temp_c ? <Row k="温度" v={`${st.board_temp_c}°C`} /> : null}
            </dl>
          ) : (
            <p className="text-sm text-muted">纳管并轮询后显示实时状态。</p>
          )}
        </Card>

        <Card>
          <CardHeader title="能力矩阵" />
          {cap ? (
            <dl className="space-y-2 text-sm">
              <Row k="RouterOS" v={<span className="font-mono text-xs">{cap.routeros_version}</span>} />
              <Row k="架构" v={cap.architecture} />
              <Row k="License" v={`L${cap.license_level}`} />
              <Row k="Device Mode" v={cap.device_mode || "—"} />
              <Row k="路由" v={[cap.supports_bgp && "BGP", cap.supports_ospf && "OSPF"].filter(Boolean).join(" / ") || "—"} />
              <Row k="WireGuard" v={cap.supports_wireguard ? "支持" : "—"} />
              <Row k="容器" v={cap.supports_container ? "支持" : "—"} />
            </dl>
          ) : (
            <p className="text-sm text-muted">纳管后自动识别能力。</p>
          )}
        </Card>
      </div>

      {cap?.interfaces && cap.interfaces.length > 0 && (
        <Card className="p-0">
          <CardHeader title="接口" subtitle={`${cap.interfaces.length} 个接口`} />
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                  <th className="px-5 py-3 font-medium">名称</th>
                  <th className="px-5 py-3 font-medium">类型</th>
                  <th className="px-5 py-3 font-medium">速率</th>
                  <th className="px-5 py-3 font-medium">能力</th>
                </tr>
              </thead>
              <tbody>
                {cap.interfaces.map((i) => (
                  <tr key={i.name} className="border-b border-border/60">
                    <td className="px-5 py-3 font-mono text-xs">{i.name}</td>
                    <td className="px-5 py-3 text-muted">{i.type}</td>
                    <td className="px-5 py-3 font-mono text-xs text-muted">{i.speed_mbps ? `${i.speed_mbps} Mbps` : "—"}</td>
                    <td className="px-5 py-3 text-muted">{(i.features || []).join(", ") || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  );
}

function Row({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="text-muted">{k}</dt>
      <dd className="text-right">{v}</dd>
    </div>
  );
}
