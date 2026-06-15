"use client";

import * as React from "react";
import { Plus, Trash2, Loader2, Gauge, Eye, Send } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import {
  listQoSPolicies, createQoSPolicy, deleteQoSPolicy, listDevices, applyQoS,
  type QoSPolicy, type Device, type QoSApplyResult, ApiError,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

const PRESETS = ["1M", "5M", "10M", "20M", "50M", "100M"];
const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

export default function QoSPage() {
  const [policies, setPolicies] = React.useState<QoSPolicy[]>([]);
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [deviceId, setDeviceId] = React.useState("");
  const [sel, setSel] = React.useState<Record<string, boolean>>({});
  const [result, setResult] = React.useState<QoSApplyResult | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  const [name, setName] = React.useState("");
  const [target, setTarget] = React.useState("");
  const [maxLimit, setMaxLimit] = React.useState("10M");
  const [limitAt, setLimitAt] = React.useState("");
  const [priority, setPriority] = React.useState("8");

  async function reload() {
    const [p, d] = await Promise.all([
      listQoSPolicies(currentToken()).catch(() => []),
      listDevices(currentToken()).catch(() => []),
    ]);
    setPolicies(p);
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
      await createQoSPolicy({
        name: name.trim(),
        target: target.trim(),
        max_limit: maxLimit.trim(),
        limit_at: limitAt.trim() || undefined,
        priority: parseInt(priority, 10) || 8,
      }, currentToken());
      setName(""); setTarget("");
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "添加失败");
    } finally { setBusy(false); }
  }

  async function remove(id: string) {
    if (!window.confirm("删除该限速策略？")) return;
    await deleteQoSPolicy(id, currentToken()).catch(() => {});
    await reload();
  }

  async function deliver(dryRun: boolean) {
    setError(null);
    const ids = policies.filter((p) => sel[p.id]).map((p) => p.id);
    if (!deviceId) { setError("请选择目标设备"); return; }
    if (ids.length === 0) { setError("请勾选要下发的限速策略"); return; }
    setBusy(true);
    try {
      setResult(await applyQoS(deviceId, { policy_ids: ids, dry_run: dryRun }, currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "下发失败");
    } finally { setBusy(false); }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">限速配置</h1>
        <p className="mt-1 text-sm text-muted">
          维护 Simple Queue 策略池，下发到设备 /queue/simple（上行/下行 max-limit，支持网段、主机或接口 target）
        </p>
      </div>

      <Card className="space-y-4">
        <CardHeader title="添加快限速策略" subtitle="对应 RouterOS Simple Queues" />
        <form onSubmit={add} className="grid grid-cols-1 gap-3 sm:grid-cols-6">
          <input value={name} onChange={(e) => setName(e.target.value)} required placeholder="名称，如 lan-10m"
            className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary sm:col-span-2" />
          <input value={target} onChange={(e) => setTarget(e.target.value)} required placeholder="目标：192.168.0.0/24 或 ether1"
            className="rounded-lg border border-border bg-elevated px-3 py-2 font-mono text-sm outline-none focus:border-primary sm:col-span-2" />
          <div className="flex gap-1 sm:col-span-2">
            <input value={maxLimit} onChange={(e) => setMaxLimit(e.target.value)} required placeholder="10M 或 10M/5M"
              className="w-full rounded-lg border border-border bg-elevated px-3 py-2 font-mono text-sm outline-none focus:border-primary" />
            <select value={maxLimit} onChange={(e) => setMaxLimit(e.target.value)}
              className="rounded-lg border border-border bg-elevated px-2 text-xs outline-none">
              {PRESETS.map((p) => <option key={p} value={p}>{p}</option>)}
            </select>
          </div>
          <input value={limitAt} onChange={(e) => setLimitAt(e.target.value)} placeholder="保证速率 limit-at（可选）"
            className="rounded-lg border border-border bg-elevated px-3 py-2 font-mono text-sm outline-none focus:border-primary sm:col-span-2" />
          <select value={priority} onChange={(e) => setPriority(e.target.value)}
            className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
            {[1, 2, 3, 4, 5, 6, 7, 8].map((n) => (
              <option key={n} value={n}>优先级 {n}{n === 1 ? "（最高）" : n === 8 ? "（默认）" : ""}</option>
            ))}
          </select>
          <button type="submit" disabled={busy}
            className="flex items-center justify-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60 sm:col-span-3">
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} 添加
          </button>
        </form>
        {error && <p className="text-sm text-danger">{error}</p>}
      </Card>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="p-0">
          <CardHeader title="策略池" inset border />
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                  <th className="w-10 px-5 py-3"></th>
                  <th className="px-3 py-3 font-medium">名称</th>
                  <th className="px-3 py-3 font-medium">目标</th>
                  <th className="px-3 py-3 font-medium">max-limit</th>
                  <th className="px-3 py-3 font-medium">优先级</th>
                  <th className="px-3 py-3 text-right font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {policies.map((p) => (
                  <tr key={p.id} className="border-b border-border/50 hover:bg-elevated/40">
                    <td className="px-5 py-3">
                      <input type="checkbox" checked={!!sel[p.id]} onChange={(e) => setSel((x) => ({ ...x, [p.id]: e.target.checked }))} />
                    </td>
                    <td className="px-3 py-3 font-medium">{p.name}</td>
                    <td className="px-3 py-3 max-w-[180px] truncate font-mono text-xs">{p.target}</td>
                    <td className="px-3 py-3 font-mono text-xs">{p.max_limit}{p.limit_at ? <span className="text-muted"> · at {p.limit_at}</span> : null}</td>
                    <td className="px-3 py-3 text-muted">{p.priority || 8}</td>
                    <td className="px-3 py-3 text-right">
                      <button onClick={() => remove(p.id)} title="删除" className="rounded-md p-1.5 text-muted hover:bg-danger/10 hover:text-danger">
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
                {policies.length === 0 && (
                  <tr><td colSpan={6} className="px-5 py-10 text-center text-sm text-muted">策略池为空，请在上方添加</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </Card>

        <Card>
          <CardHeader title="下发到设备" subtitle="写入 /queue/simple（commit-confirm 保护）" />
          <div className="space-y-3">
            <select value={deviceId} onChange={(e) => setDeviceId(e.target.value)}
              className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              {devices.length === 0 && <option value="">（无已托管设备）</option>}
              {devices.map((d) => <option key={d.id} value={d.id}>{d.name} · {d.mgmt_address}</option>)}
            </select>
            <div className="flex flex-wrap gap-2">
              <button onClick={() => deliver(true)} disabled={busy}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览
              </button>
              <button onClick={() => deliver(false)} disabled={busy}
                className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} 下发
              </button>
              {result?.plan && <Badge tone={riskTone[result.plan.aggregate_risk] ?? "neutral"}>风险 {result.plan.aggregate_risk}</Badge>}
              {result?.rule_count != null && <Badge tone="primary">{result.rule_count} 条队列</Badge>}
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
                    {Object.entries(st.attributes).map(([k, v]) => (
                      <div key={k} className="pl-4 text-foreground/80">{k} = {v}</div>
                    ))}
                  </div>
                ))}
              </pre>
            )}
            <p className="flex items-center gap-1 text-xs text-muted">
              <Gauge className="h-3 w-3" /> 已勾选 {policies.filter((p) => sel[p.id]).length} 条策略
            </p>
          </div>
        </Card>
      </div>
    </div>
  );
}
