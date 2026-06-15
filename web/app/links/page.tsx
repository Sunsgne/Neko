"use client";

import * as React from "react";
import { Plus, Trash2, Loader2, Activity, RefreshCw, Gauge } from "lucide-react";
import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import {
  listLinks, listDevices, createLink, deleteLink, probeLink,
  type Link, type Device, ApiError,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

type Tone = "success" | "primary" | "warning" | "danger";

function scoreTone(score: number): Tone {
  if (score >= 90) return "success";
  if (score >= 75) return "primary";
  if (score >= 60) return "warning";
  return "danger";
}
const scoreText: Record<Tone, string> = { success: "text-success", primary: "text-primary", warning: "text-warning", danger: "text-danger" };
const scoreBar: Record<Tone, string> = { success: "bg-success", primary: "bg-primary", warning: "bg-warning", danger: "bg-danger" };

function statusTone(status: string): "success" | "warning" | "danger" | "neutral" {
  if (status === "up") return "success";
  if (status === "degraded") return "warning";
  if (status === "down") return "danger";
  return "neutral";
}
const statusLabel: Record<string, string> = { up: "正常", degraded: "降级", down: "中断", unknown: "未探测" };

export default function LinksPage() {
  const [links, setLinks] = React.useState<Link[]>([]);
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [error, setError] = React.useState<string | null>(null);
  const [probing, setProbing] = React.useState<Record<string, boolean>>({});
  const [busy, setBusy] = React.useState(false);
  const [showAdd, setShowAdd] = React.useState(false);

  const [deviceId, setDeviceId] = React.useState("");
  const [name, setName] = React.useState("");
  const [kind, setKind] = React.useState("wan");
  const [isp, setIsp] = React.useState("telecom");
  const [role, setRole] = React.useState("primary");
  const [target, setTarget] = React.useState("");

  async function reload() {
    const [l, d] = await Promise.all([
      listLinks(currentToken()).catch(() => []),
      listDevices(currentToken()).catch(() => []),
    ]);
    setLinks(l);
    const managed = d.filter((x) => x.trust_state === "managed");
    setDevices(managed);
    if (!deviceId && managed[0]) setDeviceId(managed[0].id);
  }
  React.useEffect(() => { reload(); /* eslint-disable-next-line */ }, []);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!deviceId) return setError("请选择设备");
    setBusy(true);
    try {
      await createLink({ device_id: deviceId, name: name.trim(), kind, isp, role, target: target.trim() }, currentToken());
      setName(""); setTarget(""); setShowAdd(false);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "添加失败");
    } finally { setBusy(false); }
  }

  async function remove(id: string) {
    if (!window.confirm("删除该链路监控？")) return;
    await deleteLink(id, currentToken()).catch(() => {});
    await reload();
  }

  async function probe(id: string) {
    setProbing((p) => ({ ...p, [id]: true }));
    setError(null);
    try {
      const res = await probeLink(id, currentToken());
      setLinks((ls) => ls.map((l) => (l.id === id ? res.link : l)));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "探测失败");
    } finally {
      setProbing((p) => ({ ...p, [id]: false }));
    }
  }

  async function probeAll() {
    for (const l of links) await probe(l.id);
  }

  const deviceName = (id?: string) => devices.find((d) => d.id === id)?.name ?? id ?? "—";

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">链路质量</h1>
          <p className="mt-1 text-sm text-muted">从设备主动 Ping 目标测量 · 延迟 / 抖动 / 丢包 · 评分 · 实时探测</p>
        </div>
        <div className="flex gap-2">
          <button onClick={probeAll} disabled={links.length === 0}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-50">
            <RefreshCw className="h-4 w-4" /> 全部探测
          </button>
          <button onClick={() => setShowAdd((v) => !v)}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90">
            <Plus className="h-4 w-4" /> 添加链路
          </button>
        </div>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {showAdd && (
        <Card>
          <CardHeader title="添加监控链路" subtitle="平台将定时从所选设备 Ping 目标，测量真实链路质量" />
          <form onSubmit={add} className="grid grid-cols-1 gap-3 sm:grid-cols-6">
            <select value={deviceId} onChange={(e) => setDeviceId(e.target.value)} className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary sm:col-span-2">
              {devices.length === 0 && <option value="">（无已托管设备）</option>}
              {devices.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
            </select>
            <input value={name} onChange={(e) => setName(e.target.value)} required placeholder="名称，如 上海-电信"
              className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary sm:col-span-2" />
            <input value={target} onChange={(e) => setTarget(e.target.value)} required placeholder="探测目标 IP，如 202.96.209.133"
              className="rounded-lg border border-border bg-elevated px-3 py-2 font-mono text-sm outline-none focus:border-primary sm:col-span-2" />
            <select value={kind} onChange={(e) => setKind(e.target.value)} className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              <option value="wan">WAN</option><option value="overlay">Overlay</option>
            </select>
            <select value={isp} onChange={(e) => setIsp(e.target.value)} className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              <option value="telecom">电信</option><option value="unicom">联通</option><option value="mobile">移动</option><option value="edu">教育网</option><option value="overlay">Overlay</option>
            </select>
            <select value={role} onChange={(e) => setRole(e.target.value)} className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              <option value="primary">主用</option><option value="backup">备用</option>
            </select>
            <button type="submit" disabled={busy} className="flex items-center justify-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} 添加
            </button>
          </form>
        </Card>
      )}

      {links.length === 0 ? (
        <Card>
          <div className="flex flex-col items-center gap-2 py-16 text-center text-sm text-muted">
            <Activity className="h-6 w-6 text-primary" />
            还没有监控链路。点「添加链路」选择设备与探测目标，平台会从设备实测质量。
          </div>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {links.map((l) => (
            <Card key={l.id}>
              <CardHeader
                title={l.name}
                subtitle={`${deviceName(l.device_id)} · ${l.kind} · ${l.isp || "overlay"} · ${l.role === "backup" ? "备用" : "主用"}`}
                action={<Badge tone={statusTone(l.status)}><StatusDot tone={statusTone(l.status)} />{statusLabel[l.status] ?? l.status}</Badge>}
              />
              <div className="mb-1 flex items-end justify-between">
                <span className="text-xs uppercase tracking-wide text-muted">评分</span>
                <span className={`font-mono text-2xl font-semibold ${scoreText[scoreTone(l.score)]}`}>{l.score.toFixed(0)}</span>
              </div>
              <div className="mb-3 h-2 overflow-hidden rounded-full bg-border/60">
                <div className={`h-full rounded-full ${scoreBar[scoreTone(l.score)]}`} style={{ width: `${Math.max(2, l.score)}%` }} />
              </div>
              <div className="mb-3 grid grid-cols-3 gap-2 text-center">
                <Metric label="延迟" value={`${l.latency_ms.toFixed(0)} ms`} />
                <Metric label="抖动" value={`${l.jitter_ms.toFixed(0)} ms`} />
                <Metric label="丢包" value={`${(l.loss * 100).toFixed(1)}%`} />
              </div>
              <div className="flex items-center justify-between text-xs text-muted">
                <span className="font-mono">→ {l.target || "—"}</span>
                <span>{l.measured_at ? new Date(l.measured_at).toLocaleTimeString() : "未探测"}</span>
              </div>
              <div className="mt-3 flex gap-2">
                <button onClick={() => probe(l.id)} disabled={probing[l.id]}
                  className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm hover:border-primary disabled:opacity-60">
                  {probing[l.id] ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Gauge className="h-3.5 w-3.5" />} 立即探测
                </button>
                <button onClick={() => remove(l.id)} title="删除"
                  className="rounded-lg border border-border p-1.5 text-muted hover:border-danger/50 hover:bg-danger/10 hover:text-danger">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-elevated/50 py-2">
      <div className="font-mono text-sm font-medium">{value}</div>
      <div className="text-[10px] uppercase tracking-wide text-muted">{label}</div>
    </div>
  );
}
