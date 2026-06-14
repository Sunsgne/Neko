"use client";

import * as React from "react";
import { Workflow, Eye, Send, Loader2, Server, Router as RouterIcon } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import { listDevices, orchestrate, type Device, type OrchestrateResult, ApiError } from "@/lib/api";
import { currentToken } from "@/lib/session";

type Mode = "mesh" | "overseas_direct" | "smart_split";

const MODES: { id: Mode; title: string; desc: string }[] = [
  { id: "mesh", title: "组网（站点入网）", desc: "CPE 经隧道接入骨干 POP，打通内网网段" },
  { id: "overseas_direct", title: "加速·海外直连", desc: "全量流量经该 POP 出口直连（不分流）" },
  { id: "smart_split", title: "加速·智能分流", desc: "海外走 POP 出口，国内本地直连" },
];

const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

function host(addr: string): string {
  // strip :port
  const i = addr.lastIndexOf(":");
  return i > 0 && addr.indexOf(".") < i ? addr.slice(0, i) : addr;
}
function peerOf(cidr: string): string {
  // 100.64.0.2/30 -> 100.64.0.1
  const ip = cidr.split("/")[0];
  const parts = ip.split(".");
  if (parts.length === 4) { parts[3] = "1"; return parts.join("."); }
  return ip;
}

export default function OrchestratePage() {
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [cpeId, setCpeId] = React.useState("");
  const [popId, setPopId] = React.useState("");
  const [mode, setMode] = React.useState<Mode>("mesh");
  const [cpeOverlay, setCpeOverlay] = React.useState("100.64.0.2/30");
  const [popPeer, setPopPeer] = React.useState("100.64.0.1");
  const [popKey, setPopKey] = React.useState("");
  const [internalCidr, setInternalCidr] = React.useState("10.0.0.0/8");
  const [localWan, setLocalWan] = React.useState("192.168.1.1");

  const [result, setResult] = React.useState<OrchestrateResult | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    listDevices(currentToken()).then((d) => {
      const managed = d.filter((x) => x.trust_state === "managed");
      setDevices(managed);
    }).catch(() => setDevices([]));
  }, []);

  const cpes = devices.filter((d) => d.role !== "backbone" && d.role !== "gateway");
  const pops = devices.filter((d) => d.role === "backbone" || d.role === "gateway");
  const pop = devices.find((d) => d.id === popId);
  const cpe = devices.find((d) => d.id === cpeId);

  React.useEffect(() => { setPopPeer(peerOf(cpeOverlay)); }, [cpeOverlay]);

  function tunnelName(): string {
    return pop ? `wg-${pop.name}`.replace(/[^a-zA-Z0-9-]/g, "-").slice(0, 24) : "wg-pop";
  }

  function buildBody(dryRun: boolean, creds?: { u: string; p: string }) {
    const popEndpoint = pop ? host(pop.mgmt_address) : "";
    const tn = tunnelName();
    const body: Record<string, unknown> = {
      dry_run: dryRun,
      tunnel: {
        name: tn,
        type: "wireguard",
        remote_ip: popEndpoint,
        tunnel_addr: cpeOverlay,
        public_key: popKey || undefined,
        listen_port: 13231,
      },
    };
    if (mode === "mesh") {
      body.overlay_routes = internalCidr.split(",").map((s) => s.trim()).filter(Boolean);
    } else if (mode === "overseas_direct") {
      body.accel = { mode: "overseas_direct", tunnel_interface: tn, overseas_gateway: popPeer, overseas_dns: ["8.8.8.8", "1.1.1.1"] };
    } else if (mode === "smart_split") {
      body.accel = { mode: "smart_split", tunnel_interface: tn, overseas_gateway: popPeer, local_wan_gateway: localWan, domestic_dns: ["223.5.5.5"] };
    }
    if (creds) { body.username = creds.u; body.password = creds.p; body.confirm_timeout_sec = 90; }
    return body;
  }

  async function run(dryRun: boolean) {
    setError(null);
    if (!cpeId) return setError("请选择接入设备 (CPE)");
    if (!popId) return setError("请选择接入的骨干 / 出口节点 (POP)");
    let creds: { u: string; p: string } | undefined;
    if (!dryRun) {
      const u = window.prompt("CPE 设备登录用户名（用于下发，不保存）", "admin");
      if (u === null) return;
      const p = window.prompt("CPE 设备登录密码") ?? "";
      creds = { u, p };
    }
    setBusy(true);
    try {
      setResult(await orchestrate(cpeId, buildBody(dryRun, creds), currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "请求失败");
      setResult(null);
    } finally { setBusy(false); }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">站点编排 · 一键入网</h1>
        <p className="mt-1 text-sm text-muted">将客户站点 (CPE) 经隧道接入骨干 POP，按业务选择组网或加速，预览后一键下发(无需登录设备)</p>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="space-y-4">
          <CardHeader title="1 · 站点与接入节点" subtitle="选择 CPE 与它要接入的骨干 / 出口节点" />
          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-xs uppercase tracking-wide text-muted"><RouterIcon className="h-3.5 w-3.5" /> 接入设备 (CPE)</label>
            <select value={cpeId} onChange={(e) => setCpeId(e.target.value)} className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              <option value="">请选择…</option>
              {cpes.map((d) => <option key={d.id} value={d.id}>{d.name} · {d.mgmt_address}</option>)}
            </select>
          </div>
          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-xs uppercase tracking-wide text-muted"><Server className="h-3.5 w-3.5" /> 接入骨干 / 出口节点 (POP)</label>
            <select value={popId} onChange={(e) => setPopId(e.target.value)} className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              <option value="">请选择…</option>
              {pops.map((d) => <option key={d.id} value={d.id}>{d.name} · {d.region || d.role} · {d.mgmt_address}</option>)}
            </select>
            {pops.length === 0 && <p className="mt-1 text-xs text-warning">暂无骨干/出口节点，请先在「骨干节点」登记。</p>}
          </div>

          <CardHeader title="2 · 业务模式" />
          <div className="grid grid-cols-1 gap-2">
            {MODES.map((m) => (
              <button key={m.id} onClick={() => setMode(m.id)} className={`rounded-lg border px-3 py-2 text-left text-sm transition-colors ${mode === m.id ? "border-primary bg-primary/10" : "border-border hover:border-border"}`}>
                <div className={mode === m.id ? "font-medium text-primary" : "font-medium"}>{m.title}</div>
                <div className="text-xs text-muted">{m.desc}</div>
              </button>
            ))}
          </div>

          <CardHeader title="3 · 隧道参数" subtitle="WireGuard 隧道，端点取自所选 POP" />
          <div className="grid grid-cols-2 gap-3">
            <F label="CPE 隧道地址" v={cpeOverlay} on={setCpeOverlay} mono />
            <F label="POP 对端地址" v={popPeer} on={setPopPeer} mono />
            <F label="POP 端点(自动)" v={pop ? host(pop.mgmt_address) : "—"} on={() => {}} mono disabled />
            <F label="隧道接口名(自动)" v={tunnelName()} on={() => {}} mono disabled />
            <div className="col-span-2"><F label="POP WireGuard 公钥（可选）" v={popKey} on={setPopKey} mono placeholder="留空则下发后在 POP 端补充" /></div>
            {mode === "mesh" && <div className="col-span-2"><F label="内网网段（经隧道可达，逗号分隔）" v={internalCidr} on={setInternalCidr} mono /></div>}
            {mode === "smart_split" && <div className="col-span-2"><F label="本地出口网关（国内直连）" v={localWan} on={setLocalWan} mono /></div>}
          </div>

          <div className="flex gap-2">
            <button onClick={() => run(true)} disabled={busy} className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览
            </button>
            <button onClick={() => run(false)} disabled={busy} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} 下发到 CPE
            </button>
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
        </Card>

        <Card>
          <CardHeader
            title="生成配置 / 下发结果"
            subtitle={cpe && pop ? `${cpe.name} ⇄ ${pop.name}（${MODES.find((m) => m.id === mode)?.title}）` : "选择 CPE 与 POP 后预览"}
            action={result?.plan ? <Badge tone={riskTone[result.plan.aggregate_risk] ?? "neutral"}>风险 {result.plan.aggregate_risk}</Badge> : undefined}
          />
          {!result && <div className="flex flex-col items-center gap-2 py-16 text-center text-sm text-muted"><Workflow className="h-6 w-6 text-primary" />点击「预览」生成 RouterOS 配置</div>}
          {result?.result && (
            <div className={`mb-3 rounded-lg border p-3 text-sm ${result.result.status === "committed" ? "border-success/40 bg-success/10 text-success" : "border-danger/40 bg-danger/10 text-danger"}`}>
              下发结果：{result.result.status}{result.result.reason ? ` · ${result.result.reason}` : ""}
            </div>
          )}
          {result?.error && <p className="mb-2 text-sm text-danger">{result.error}</p>}
          {result?.desired && (
            <pre className="max-h-[460px] overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">
              {result.desired.statements.map((st) => (
                <div key={st.path + st.key} className="mb-1">
                  <span className="text-primary">{st.path}</span> <span className="text-muted">{st.key}</span>
                  {Object.entries(st.attributes).map(([k, v]) => <div key={k} className="pl-4 text-foreground/80">{k} = {v}</div>)}
                </div>
              ))}
            </pre>
          )}
        </Card>
      </div>
    </div>
  );
}

function F({ label, v, on, mono, disabled, placeholder }: { label: string; v: string; on: (s: string) => void; mono?: boolean; disabled?: boolean; placeholder?: string }) {
  return (
    <div>
      <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">{label}</label>
      <input value={v} onChange={(e) => on(e.target.value)} disabled={disabled} placeholder={placeholder}
        className={`w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary disabled:opacity-60 ${mono ? "font-mono" : ""}`} />
    </div>
  );
}
