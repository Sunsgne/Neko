"use client";

import * as React from "react";
import { Workflow, Plus, Trash2, Eye, Send, Loader2 } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import { listDevices, orchestrate, type Device, type OrchestrateResult, ApiError } from "@/lib/api";
import { currentToken } from "@/lib/session";

type Uplink = { name: string; gateway: string; interface: string; priority: number; weight: number };

const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

export default function OrchestratePage() {
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [deviceId, setDeviceId] = React.useState("");
  const [strategy, setStrategy] = React.useState<"failover" | "loadbalance">("failover");
  const [uplinks, setUplinks] = React.useState<Uplink[]>([
    { name: "电信", gateway: "192.168.1.1", interface: "ether1", priority: 1, weight: 1 },
    { name: "联通", gateway: "192.168.2.1", interface: "ether2", priority: 2, weight: 1 },
  ]);
  const [accelOn, setAccelOn] = React.useState(false);
  const [accelMode, setAccelMode] = React.useState("overseas_direct");
  const [tunnel, setTunnel] = React.useState("wg-hk");
  const [overseasGw, setOverseasGw] = React.useState("100.64.0.1");

  const [result, setResult] = React.useState<OrchestrateResult | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    listDevices(currentToken())
      .then((d) => {
        const managed = d.filter((x) => x.trust_state === "managed");
        setDevices(managed);
        if (managed[0]) setDeviceId(managed[0].id);
      })
      .catch(() => setDevices([]));
  }, []);

  function buildBody(dryRun: boolean, creds?: { u: string; p: string }) {
    const body: Record<string, unknown> = {
      dry_run: dryRun,
      link_policy: { strategy, uplinks },
    };
    if (accelOn) {
      body.accel = {
        mode: accelMode,
        tunnel_interface: tunnel,
        overseas_gateway: overseasGw,
        overseas_dns: ["8.8.8.8", "1.1.1.1"],
      };
    }
    if (creds) {
      body.username = creds.u;
      body.password = creds.p;
      body.confirm_timeout_sec = 90;
    }
    return body;
  }

  async function run(dryRun: boolean) {
    setError(null);
    if (!deviceId) {
      setError("请先选择一台已托管设备");
      return;
    }
    let creds: { u: string; p: string } | undefined;
    if (!dryRun) {
      const u = window.prompt("设备登录用户名（用于通过 RouterOS REST 下发，不会被保存）", "admin");
      if (u === null) return;
      const p = window.prompt("设备登录密码") ?? "";
      creds = { u, p };
    }
    setBusy(true);
    try {
      setResult(await orchestrate(deviceId, buildBody(dryRun, creds), currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "请求失败");
      setResult(null);
    } finally {
      setBusy(false);
    }
  }

  function updUplink(i: number, patch: Partial<Uplink>) {
    setUplinks((u) => u.map((x, idx) => (idx === i ? { ...x, ...patch } : x)));
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">站点编排 · 一键下发</h1>
        <p className="mt-1 text-sm text-muted">选择设备与链路策略，预览生成的 RouterOS 配置后一键下发（无需登录设备）</p>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="space-y-4">
          <CardHeader title="1 · 目标设备" subtitle="仅列出已托管 (managed) 设备" />
          <select
            value={deviceId}
            onChange={(e) => setDeviceId(e.target.value)}
            className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary"
          >
            {devices.length === 0 && <option value="">（无已托管设备）</option>}
            {devices.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name} · {d.mgmt_address} · {d.role}
              </option>
            ))}
          </select>

          <CardHeader title="2 · 链路选择" subtitle="主备自动切换 / ECMP 负载均衡（含 check-gateway 健康检测）" />
          <div className="flex gap-2">
            {(["failover", "loadbalance"] as const).map((s) => (
              <button
                key={s}
                onClick={() => setStrategy(s)}
                className={`rounded-lg border px-3 py-1.5 text-sm ${strategy === s ? "border-primary bg-primary/15 text-primary" : "border-border text-muted"}`}
              >
                {s === "failover" ? "主备切换" : "负载均衡"}
              </button>
            ))}
          </div>
          <div className="space-y-2">
            {uplinks.map((u, i) => (
              <div key={i} className="grid grid-cols-[1fr_1fr_0.8fr_auto] items-center gap-2">
                <input value={u.name} onChange={(e) => updUplink(i, { name: e.target.value })} placeholder="名称"
                  className="rounded-lg border border-border bg-elevated px-2 py-1.5 text-sm outline-none focus:border-primary" />
                <input value={u.gateway} onChange={(e) => updUplink(i, { gateway: e.target.value })} placeholder="网关"
                  className="rounded-lg border border-border bg-elevated px-2 py-1.5 font-mono text-xs outline-none focus:border-primary" />
                <input value={u.interface} onChange={(e) => updUplink(i, { interface: e.target.value })} placeholder="接口"
                  className="rounded-lg border border-border bg-elevated px-2 py-1.5 font-mono text-xs outline-none focus:border-primary" />
                <button onClick={() => setUplinks((x) => x.filter((_, idx) => idx !== i))} className="rounded-lg p-1.5 text-muted hover:text-danger">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
            <button
              onClick={() => setUplinks((u) => [...u, { name: "", gateway: "", interface: "", priority: u.length + 1, weight: 1 }])}
              className="flex items-center gap-1 text-xs text-primary"
            >
              <Plus className="h-3 w-3" /> 添加上行
            </button>
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={accelOn} onChange={(e) => setAccelOn(e.target.checked)} />
            叠加加速策略
          </label>
          {accelOn && (
            <div className="space-y-2 rounded-lg border border-border bg-elevated/40 p-3">
              <select value={accelMode} onChange={(e) => setAccelMode(e.target.value)}
                className="w-full rounded-lg border border-border bg-elevated px-2 py-1.5 text-sm outline-none focus:border-primary">
                <option value="overseas_direct">海外运营（直连，不分流）</option>
                <option value="smart_split">智能分流</option>
              </select>
              <input value={tunnel} onChange={(e) => setTunnel(e.target.value)} placeholder="隧道接口"
                className="w-full rounded-lg border border-border bg-elevated px-2 py-1.5 font-mono text-xs outline-none focus:border-primary" />
              <input value={overseasGw} onChange={(e) => setOverseasGw(e.target.value)} placeholder="海外网关"
                className="w-full rounded-lg border border-border bg-elevated px-2 py-1.5 font-mono text-xs outline-none focus:border-primary" />
            </div>
          )}

          <div className="flex gap-2">
            <button onClick={() => run(true)} disabled={busy}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览
            </button>
            <button onClick={() => run(false)} disabled={busy}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} 下发到设备
            </button>
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
        </Card>

        <Card>
          <CardHeader
            title="3 · 生成配置 / 下发结果"
            subtitle="预览 = 不触碰设备；下发 = 通过 RouterOS REST 应用（commit-confirm 保护）"
            action={result?.plan ? <Badge tone={riskTone[result.plan.aggregate_risk] ?? "neutral"}>风险 {result.plan.aggregate_risk}</Badge> : undefined}
          />
          {!result && <p className="py-12 text-center text-sm text-muted">点击「预览」生成配置</p>}
          {result?.result && (
            <div className={`mb-3 rounded-lg border p-3 text-sm ${result.result.status === "committed" ? "border-success/40 bg-success/10 text-success" : "border-danger/40 bg-danger/10 text-danger"}`}>
              下发结果：{result.result.status}{result.result.reason ? ` · ${result.result.reason}` : ""}
            </div>
          )}
          {result?.error && <p className="mb-2 text-sm text-danger">{result.error}</p>}
          {(result?.desired || result?.plan) && (
            <pre className="max-h-[460px] overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">
              {(result.desired?.statements ?? []).map((st) => (
                <div key={st.path + st.key}>
                  <span className="text-primary">{st.path}</span> <span className="text-muted">{st.key}</span>
                  {Object.entries(st.attributes).map(([k, v]) => (
                    <div key={k} className="pl-4 text-foreground/80">{k} = {v}</div>
                  ))}
                </div>
              ))}
            </pre>
          )}
        </Card>
      </div>
    </div>
  );
}
