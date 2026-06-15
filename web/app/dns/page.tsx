"use client";

import * as React from "react";
import { Plus, Trash2, Loader2, Globe, Eye, Send } from "lucide-react";
import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import {
  listDNSServers, createDNSServer, deleteDNSServer, listDevices, applyDNS,
  type DNSServer, type Device, type DNSApplyResult, ApiError,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

const ispLabel: Record<string, string> = {
  telecom: "电信", unicom: "联通", mobile: "移动", edu: "教育网", public: "公共", "": "—",
};
const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

export default function DnsPage() {
  const [servers, setServers] = React.useState<DNSServer[]>([]);
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [deviceId, setDeviceId] = React.useState("");
  const [sel, setSel] = React.useState<Record<string, boolean>>({});
  const [result, setResult] = React.useState<DNSApplyResult | null>(null);
  const [verifyDoh, setVerifyDoh] = React.useState<"auto" | "on" | "off">("auto");
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  // add form
  const [kind, setKind] = React.useState<"udp" | "doh">("udp");
  const [addr, setAddr] = React.useState("");
  const [isp, setIsp] = React.useState("public");
  const [region, setRegion] = React.useState("");
  const [ecs, setEcs] = React.useState(false);

  async function reload() {
    const [s, d] = await Promise.all([
      listDNSServers(currentToken()).catch(() => []),
      listDevices(currentToken()).catch(() => []),
    ]);
    setServers(s);
    const managed = d.filter((x) => x.trust_state === "managed");
    setDevices(managed);
    if (!deviceId && managed[0]) setDeviceId(managed[0].id);
  }
  React.useEffect(() => { reload(); /* eslint-disable-next-line */ }, []);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await createDNSServer({ kind, address: addr.trim(), isp, region: region.trim(), supports_ecs: ecs }, currentToken());
      setAddr(""); setRegion(""); setEcs(false);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "添加失败");
    } finally { setBusy(false); }
  }

  async function remove(id: string) {
    if (!window.confirm("删除该 DNS 服务器？")) return;
    await deleteDNSServer(id, currentToken()).catch(() => {});
    await reload();
  }

  async function deliver(dryRun: boolean) {
    setError(null);
    const ids = servers.filter((s) => sel[s.id]).map((s) => s.id);
    if (!deviceId) { setError("请选择目标设备"); return; }
    if (ids.length === 0) { setError("请勾选要下发的 DNS 服务器"); return; }
    setBusy(true);
    try {
      const verify = verifyDoh === "auto" ? undefined : verifyDoh === "on";
      setResult(await applyDNS(deviceId, { server_ids: ids, verify_doh_cert: verify, dry_run: dryRun }, currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "下发失败");
    } finally { setBusy(false); }
  }

  const healthy = servers.filter((s) => s.healthy).length;
  const dohSelected = servers.some((s) => sel[s.id] && s.kind === "doh");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">DNS 管理</h1>
        <p className="mt-1 text-sm text-muted">维护 DNS 服务器池，按需选择并下发到设备（/ip/dns）— {healthy}/{servers.length} 健康</p>
      </div>

      <Card className="space-y-4">
        <CardHeader title="添加 DNS 服务器" />
        <form onSubmit={add} className="grid grid-cols-1 gap-3 sm:grid-cols-6">
          <select value={kind} onChange={(e) => setKind(e.target.value as "udp" | "doh")} className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
            <option value="udp">普通 UDP</option><option value="doh">DoH (HTTPS)</option>
          </select>
          <input value={addr} onChange={(e) => setAddr(e.target.value)} required
            placeholder={kind === "doh" ? "DoH URL，如 https://dns.alidns.com/dns-query" : "地址，如 223.5.5.5"}
            className="rounded-lg border border-border bg-elevated px-3 py-2 font-mono text-sm outline-none focus:border-primary sm:col-span-2" />
          <select value={isp} onChange={(e) => setIsp(e.target.value)} className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
            <option value="public">公共</option><option value="telecom">电信</option><option value="unicom">联通</option><option value="mobile">移动</option><option value="edu">教育网</option>
          </select>
          <input value={region} onChange={(e) => setRegion(e.target.value)} placeholder="地域（可选）"
            className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary" />
          <button type="submit" disabled={busy} className="flex items-center justify-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} 添加
          </button>
          <label className="flex items-center gap-2 text-sm sm:col-span-6"><input type="checkbox" checked={ecs} onChange={(e) => setEcs(e.target.checked)} /> 支持 ECS（EDNS Client Subnet）</label>
        </form>
        {error && <p className="text-sm text-danger">{error}</p>}
      </Card>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="p-0">
          <CardHeader title="DNS 服务器池" inset border />
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                  <th className="w-10 px-5 py-3"></th>
                  <th className="px-3 py-3 font-medium">类型</th>
                  <th className="px-3 py-3 font-medium">地址</th>
                  <th className="px-3 py-3 font-medium">运营商</th>
                  <th className="px-3 py-3 font-medium">地域</th>
                  <th className="px-3 py-3 font-medium">ECS</th>
                  <th className="px-3 py-3 font-medium">状态</th>
                  <th className="px-3 py-3 text-right font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {servers.map((s) => (
                  <tr key={s.id} className="border-b border-border/50 hover:bg-elevated/40">
                    <td className="px-5 py-3"><input type="checkbox" checked={!!sel[s.id]} onChange={(e) => setSel((x) => ({ ...x, [s.id]: e.target.checked }))} /></td>
                    <td className="px-3 py-3">{s.kind === "doh" ? <Badge tone="primary">DoH</Badge> : <span className="text-xs text-muted">UDP</span>}</td>
                    <td className="px-3 py-3 max-w-[260px] truncate font-mono text-xs">{s.address}</td>
                    <td className="px-3 py-3">{ispLabel[s.isp] ?? s.isp}</td>
                    <td className="px-3 py-3 text-muted">{s.region || "anycast"}</td>
                    <td className="px-3 py-3">{s.supports_ecs ? <Badge tone="primary">ECS</Badge> : <span className="text-muted">—</span>}</td>
                    <td className="px-3 py-3"><span className="inline-flex items-center gap-1.5 text-xs"><StatusDot tone={s.healthy ? "success" : "danger"} />{s.healthy ? "健康" : "故障"}</span></td>
                    <td className="px-3 py-3 text-right">
                      <button onClick={() => remove(s.id)} title="删除" className="rounded-md p-1.5 text-muted hover:bg-danger/10 hover:text-danger"><Trash2 className="h-4 w-4" /></button>
                    </td>
                  </tr>
                ))}
                {servers.length === 0 && (
                  <tr><td colSpan={8} className="px-5 py-10 text-center text-sm text-muted">DNS 池为空，请在上方添加</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </Card>

        <Card>
          <CardHeader title="下发到设备" subtitle="将勾选的 DNS 写入设备 /ip/dns（commit-confirm 保护）" />
          <div className="space-y-3">
            <select value={deviceId} onChange={(e) => setDeviceId(e.target.value)} className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              {devices.length === 0 && <option value="">（无已托管设备）</option>}
              {devices.map((d) => <option key={d.id} value={d.id}>{d.name} · {d.mgmt_address}</option>)}
            </select>
            {dohSelected && (
              <div className="rounded-lg border border-border bg-elevated/40 p-3 text-sm">
                <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">验证 DoH 证书 (verify-doh-cert)</label>
                <select value={verifyDoh} onChange={(e) => setVerifyDoh(e.target.value as "auto" | "on" | "off")}
                  className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
                  <option value="auto">自动（IP 端点关闭 / 域名端点开启）</option>
                  <option value="off">关闭（IP 形式的 DoH 必须关闭）</option>
                  <option value="on">开启（需设备已导入 CA 证书）</option>
                </select>
                <p className="mt-1 text-xs text-muted">IP 形式的 DoH（如 https://202.101.51.194:9291/dns-query）无法校验证书，需关闭。</p>
              </div>
            )}
            <div className="flex gap-2">
              <button onClick={() => deliver(true)} disabled={busy} className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览
              </button>
              <button onClick={() => deliver(false)} disabled={busy} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} 下发
              </button>
              {result?.plan && <Badge tone={riskTone[result.plan.aggregate_risk] ?? "neutral"}>风险 {result.plan.aggregate_risk}</Badge>}
            </div>
            {result?.result && (
              <div className={`rounded-lg border p-3 text-sm ${result.result.status === "committed" ? "border-success/40 bg-success/10 text-success" : "border-danger/40 bg-danger/10 text-danger"}`}>
                下发结果：{result.result.status}{result.result.reason ? ` · ${result.result.reason}` : ""}
              </div>
            )}
            {result?.error && <p className="text-sm text-danger">{result.error}</p>}
            {result?.desired && (
              <pre className="max-h-72 overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs">
                {result.desired.statements.map((st) => (
                  <div key={st.path + st.key}>
                    <span className="text-primary">{st.path}</span> <span className="text-muted">{st.key}</span>
                    {Object.entries(st.attributes).map(([k, v]) => <div key={k} className="pl-4 text-foreground/80">{k} = {v}</div>)}
                  </div>
                ))}
              </pre>
            )}
            <p className="flex items-center gap-1 text-xs text-muted"><Globe className="h-3 w-3" /> 已勾选 {servers.filter((s) => sel[s.id]).length} 个 DNS</p>
          </div>
        </Card>
      </div>
    </div>
  );
}
