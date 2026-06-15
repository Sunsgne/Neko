"use client";

import * as React from "react";
import Link from "next/link";
import { Rocket, Loader2, Eye, Send, Server, Router as RouterIcon, RefreshCw, KeyRound } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import {
  listDevices, proposeAccel, orchestrate,
  type Device, type AccelProposal, type OrchestrateResult, ApiError,
} from "@/lib/api";
import { hostFromMgmt, popPeerOf, tunnelNameForPop } from "@/lib/tunnel";
import { currentToken } from "@/lib/session";

const MODES = [
  { id: "overseas_direct", title: "海外运营（直连）", desc: "全量流量经 POP 出口直连，不做分流" },
  { id: "smart_split", title: "智能分流", desc: "国内直连本地 WAN，海外经 POP 隧道出口" },
  { id: "domestic_direct", title: "国内直连", desc: "全量走本地出口（无需 POP 隧道）" },
] as const;

type Mode = (typeof MODES)[number]["id"];

const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

export default function AccelPage() {
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [cpeId, setCpeId] = React.useState("");
  const [popId, setPopId] = React.useState("");
  const [mode, setMode] = React.useState<Mode>("overseas_direct");
  const [localWan, setLocalWan] = React.useState("192.168.1.1");
  const [cpeOverlay, setCpeOverlay] = React.useState("");
  const [popPeer, setPopPeer] = React.useState("");
  const [popKey, setPopKey] = React.useState("");
  const [cpePriv, setCpePriv] = React.useState("");
  const [cpePub, setCpePub] = React.useState("");
  const [tunnel, setTunnel] = React.useState("");
  const [popEndpoint, setPopEndpoint] = React.useState("");

  const [proposal, setProposal] = React.useState<AccelProposal | null>(null);
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
  const cpe = devices.find((d) => d.id === cpeId);
  const pop = devices.find((d) => d.id === popId);
  const needsPop = mode !== "domestic_direct";

  React.useEffect(() => {
    if (cpeOverlay) setPopPeer(popPeerOf(cpeOverlay));
  }, [cpeOverlay]);

  React.useEffect(() => {
    if (pop) {
      setTunnel(tunnelNameForPop(pop.name));
      setPopEndpoint(hostFromMgmt(pop.mgmt_address));
    }
  }, [pop]);

  function applyProposal(p: AccelProposal) {
    setProposal(p);
    setCpeOverlay(p.cpe_overlay);
    setPopPeer(p.pop_peer);
    setTunnel(p.tunnel_interface);
    setPopEndpoint(p.pop_endpoint);
    setCpePriv(p.cpe_private_key);
    setCpePub(p.cpe_public_key);
    if (p.pop_public_key_hint) setPopKey(p.pop_public_key_hint);
    if (p.tunnel.public_key) setPopKey(p.tunnel.public_key);
  }

  async function generateWG() {
    setError(null);
    if (!cpeId) return setError("请选择客户侧设备 (CPE)");
    if (needsPop && !popId) return setError("请选择接入的骨干 / 出口节点 (POP)");
    setBusy(true);
    try {
      const res = await proposeAccel({
        cpe_device_id: cpeId,
        pop_device_id: popId || undefined,
        mode,
        local_wan_gateway: localWan,
        cpe_overlay: cpeOverlay || undefined,
        pop_public_key: popKey || undefined,
      }, currentToken());
      applyProposal(res.proposal);
      setResult({ dry_run: true, desired: res.desired, plan: res.plan });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "WG 协商生成失败");
    } finally { setBusy(false); }
  }

  function buildOrchestrateBody(dryRun: boolean, creds?: { u: string; p: string }) {
    const body: Record<string, unknown> = { dry_run: dryRun };
    if (needsPop && proposal) {
      body.tunnel = {
        ...proposal.tunnel,
        tunnel_addr: cpeOverlay || proposal.cpe_overlay,
        public_key: popKey || proposal.tunnel.public_key || undefined,
        private_key: cpePriv || proposal.cpe_private_key,
        remote_ip: popEndpoint || proposal.pop_endpoint,
        name: tunnel || proposal.tunnel_interface,
      };
      body.accel = {
        ...proposal.accel,
        tunnel_interface: tunnel || proposal.tunnel_interface,
        overseas_gateway: popPeer || proposal.pop_peer,
        local_wan_gateway: localWan,
      };
    } else if (mode === "domestic_direct") {
      body.accel = { mode: "domestic_direct", local_wan_gateway: localWan, domestic_dns: ["223.5.5.5", "114.114.114.114"] };
    }
    if (creds) { body.username = creds.u; body.password = creds.p; body.confirm_timeout_sec = 90; }
    return body;
  }

  async function run(dryRun: boolean) {
    setError(null);
    if (!cpeId) return setError("请选择客户侧设备 (CPE)");
    if (needsPop && !popId) return setError("请选择 POP");
    if (needsPop && !proposal && dryRun) return setError("请先点「生成 WG 协商参数」");
    let creds: { u: string; p: string } | undefined;
    if (!dryRun) {
      const u = window.prompt("CPE 设备登录用户名（用于下发，不保存）", "admin");
      if (u === null) return;
      creds = { u, p: window.prompt("CPE 设备登录密码") ?? "" };
    }
    setBusy(true);
    try {
      setResult(await orchestrate(cpeId, buildOrchestrateBody(dryRun, creds), currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "请求失败");
      setResult(null);
    } finally { setBusy(false); }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">加速业务配置</h1>
        <p className="mt-1 text-sm text-muted">
          选择 CPE 与骨干 POP，自动生成 WireGuard 隧道协商参数，预览后一键下发加速策略
        </p>
        <p className="mt-1 text-xs text-muted">
          含国内外分流(chnroutes)请使用
          <Link href="/orchestrate" className="mx-1 text-primary hover:underline">站点编排</Link>
          页
        </p>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        {MODES.map((m) => (
          <button key={m.id} onClick={() => { setMode(m.id); setProposal(null); setResult(null); }}
            className={`card text-left transition-colors ${mode === m.id ? "border-primary shadow-glow" : "hover:border-border"}`}>
            <div className="flex items-center gap-2 text-sm font-semibold">
              <Rocket className="h-4 w-4 text-primary" /> {m.title}
            </div>
            <p className="mt-1 text-xs text-muted">{m.desc}</p>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="space-y-4">
          <CardHeader title="1 · 设备与 POP" subtitle="从已托管设备中选择，勿手填管理地址" />
          <SelectField icon={RouterIcon} label="客户侧设备 (CPE)" value={cpeId} onChange={setCpeId}
            options={cpes.map((d) => ({ value: d.id, label: `${d.name} · ${d.mgmt_address}` }))} />
          {needsPop && (
            <SelectField icon={Server} label="接入骨干 / 出口 (POP)" value={popId} onChange={setPopId}
              options={pops.map((d) => ({ value: d.id, label: `${d.name} · ${d.region || d.role} · ${d.mgmt_address}` }))}
              emptyHint="暂无 POP，请先在「骨干节点」登记" />
          )}

          {needsPop && (
            <>
              <CardHeader title="2 · WireGuard 隧道（到 POP）" subtitle="点下方按钮自动生成密钥与 overlay 地址" />
              <div className="grid grid-cols-2 gap-3">
                <ReadonlyField label="隧道接口（自动）" value={tunnel || (pop ? tunnelNameForPop(pop.name) : "—")} />
                <ReadonlyField label="POP 端点（自动）" value={popEndpoint || (pop ? hostFromMgmt(pop.mgmt_address) : "—")} />
                <Field label="CPE 隧道地址" value={cpeOverlay} onChange={setCpeOverlay} mono placeholder="生成后自动填充" />
                <Field label="POP 对端地址" value={popPeer} onChange={setPopPeer} mono placeholder="生成后自动填充" />
                <div className="col-span-2">
                  <Field label="POP WireGuard 公钥" value={popKey} onChange={setPopKey} mono
                    placeholder="可自动从 POP 读取，或下发后补充" />
                </div>
                {cpePub && (
                  <div className="col-span-2 rounded-lg border border-border/60 bg-elevated/40 p-3 text-xs">
                    <div className="mb-1 flex items-center gap-1.5 font-medium text-muted"><KeyRound className="h-3.5 w-3.5" /> CPE 密钥（已生成）</div>
                    <div className="font-mono break-all text-foreground/80">公钥 {cpePub}</div>
                    <div className="mt-1 font-mono break-all text-muted">私钥 {cpePriv.slice(0, 16)}…（下发时写入设备）</div>
                  </div>
                )}
              </div>
              <button onClick={generateWG} disabled={busy}
                className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-primary/40 bg-primary/10 px-3 py-2 text-sm text-primary hover:bg-primary/20 disabled:opacity-60">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                生成 WG 协商参数
              </button>
            </>
          )}

          {(mode === "smart_split" || mode === "domestic_direct") && (
            <Field label="本地 WAN 网关（国内直连）" value={localWan} onChange={setLocalWan} mono />
          )}

          <div className="flex gap-2 pt-1">
            <button onClick={() => run(true)} disabled={busy}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览完整配置
            </button>
            <button onClick={() => run(false)} disabled={busy}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} 下发到 CPE
            </button>
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
        </Card>

        <Card>
          <CardHeader
            title="生成的 RouterOS 配置"
            subtitle={cpe && pop && needsPop ? `${cpe.name} ⇄ ${pop.name}` : cpe ? cpe.name : "选择设备后生成"}
            action={result?.plan ? <Badge tone={riskTone[result.plan.aggregate_risk] ?? "neutral"}>风险 {result.plan.aggregate_risk}</Badge> : undefined}
          />
          {result?.result && (
            <div className={`mb-3 rounded-lg border p-3 text-sm ${result.result.status === "committed" ? "border-success/40 bg-success/10 text-success" : "border-danger/40 bg-danger/10 text-danger"}`}>
              下发结果：{result.result.status}{result.result.reason ? ` · ${result.result.reason}` : ""}
            </div>
          )}
          {result?.error && <p className="mb-2 text-sm text-danger">{result.error}</p>}
          {result?.desired ? (
            <pre className="max-h-[520px] overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">
              {result.desired.statements.map((st) => (
                <div key={st.path + st.key} className="mb-2">
                  <span className="text-primary">{st.path}</span> <span className="text-muted">{st.key}</span>
                  {Object.entries(st.attributes).map(([k, v]) => (
                    <div key={k} className="pl-4 text-foreground/80">
                      {k} = {k === "private-key" ? `${String(v).slice(0, 12)}…` : v}
                    </div>
                  ))}
                </div>
              ))}
            </pre>
          ) : (
            <p className="py-16 text-center text-sm text-muted">
              {needsPop ? "选择 CPE 与 POP 后，点「生成 WG 协商参数」再预览" : "选择 CPE 后点「预览完整配置」"}
            </p>
          )}
        </Card>
      </div>
    </div>
  );
}

function SelectField({ icon: Icon, label, value, onChange, options, emptyHint }: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: Array<{ value: string; label: string }>;
  emptyHint?: string;
}) {
  return (
    <div>
      <label className="mb-1.5 flex items-center gap-1.5 text-xs uppercase tracking-wide text-muted">
        <Icon className="h-3.5 w-3.5" /> {label}
      </label>
      <select value={value} onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
        <option value="">请选择…</option>
        {options.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>
      {options.length === 0 && emptyHint && <p className="mt-1 text-xs text-warning">{emptyHint}</p>}
    </div>
  );
}

function Field({ label, value, onChange, mono, placeholder }: {
  label: string; value: string; onChange: (v: string) => void; mono?: boolean; placeholder?: string;
}) {
  return (
    <div>
      <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">{label}</label>
      <input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className={`w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary ${mono ? "font-mono" : ""}`} />
    </div>
  );
}

function ReadonlyField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">{label}</label>
      <div className="rounded-lg border border-border/60 bg-surface/50 px-3 py-2 font-mono text-sm text-muted">{value}</div>
    </div>
  );
}
