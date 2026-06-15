"use client";

import * as React from "react";
import { Rocket, Loader2, Eye, Send, Server, Router as RouterIcon, RefreshCw, KeyRound, ListTree } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import {
  listDevices, proposeAccel, deployFabric, orchestrate,
  getChnroutes, refreshChnroutes, chinaSplit,
  type Device, type AccelProposal, type FabricDeployResult, type ConfigState, type ConfigPlan,
  type ChnroutesStatus, type ChinaSplitResult, ApiError,
} from "@/lib/api";
import { hostFromMgmt, popPeerOf, tunnelNameForPop } from "@/lib/tunnel";
import { currentToken } from "@/lib/session";

const MODES = [
  { id: "overseas_direct", title: "海外运营（直连）", desc: "全量流量经 POP 出口直连，不做分流" },
  { id: "china_split", title: "国内外分流（chnroutes）", desc: "国内走路由表本地 WAN，海外 0/1+128/1 经 POP 隧道" },
  { id: "domestic_direct", title: "国内直连", desc: "全量走本地出口（无需 POP 隧道）" },
] as const;

type Mode = (typeof MODES)[number]["id"];
type PreviewSide = "cpe" | "pop" | "routes";

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
  const [popIface, setPopIface] = React.useState("");
  const [tunnel, setTunnel] = React.useState("");
  const [popEndpoint, setPopEndpoint] = React.useState("");
  const [previewSide, setPreviewSide] = React.useState<PreviewSide>("cpe");

  const [proposal, setProposal] = React.useState<AccelProposal | null>(null);
  const [result, setResult] = React.useState<FabricDeployResult | null>(null);
  const [csResult, setCsResult] = React.useState<ChinaSplitResult | null>(null);
  const [chn, setChn] = React.useState<ChnroutesStatus | null>(null);
  const [refreshingChn, setRefreshingChn] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    listDevices(currentToken()).then((d) => {
      setDevices(d.filter((x) => x.trust_state === "managed"));
    }).catch(() => setDevices([]));
    getChnroutes(currentToken()).then(setChn).catch(() => setChn(null));
  }, []);

  const cpes = devices.filter((d) => d.role !== "backbone" && d.role !== "gateway");
  const pops = devices.filter((d) => d.role === "backbone" || d.role === "gateway");
  const cpe = devices.find((d) => d.id === cpeId);
  const pop = devices.find((d) => d.id === popId);
  const needsPop = mode !== "domestic_direct";
  const isChinaSplit = mode === "china_split";

  React.useEffect(() => {
    if (cpeOverlay) setPopPeer(popPeerOf(cpeOverlay));
  }, [cpeOverlay]);

  React.useEffect(() => {
    if (pop) {
      setTunnel(tunnelNameForPop(pop.name));
      setPopEndpoint(hostFromMgmt(pop.mgmt_address));
    }
  }, [pop]);

  async function doRefreshChn() {
    setRefreshingChn(true);
    setError(null);
    try {
      setChn(await refreshChnroutes(undefined, currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "刷新国内路由表失败");
    } finally { setRefreshingChn(false); }
  }

  function applyProposal(p: AccelProposal, fabric?: FabricDeployResult["fabric"]) {
    setProposal(p);
    setCpeOverlay(p.cpe_overlay);
    setPopPeer(p.pop_peer);
    setTunnel(p.tunnel_interface);
    setPopEndpoint(p.pop_endpoint);
    setCpePriv(p.cpe_private_key);
    setCpePub(p.cpe_public_key);
    if (fabric?.link.pop_interface) setPopIface(fabric.link.pop_interface);
    if (p.pop_public_key_hint) setPopKey(p.pop_public_key_hint);
    if (p.tunnel.public_key) setPopKey(p.tunnel.public_key);
  }

  function fabricBody(dryRun: boolean) {
    return {
      cpe_device_id: cpeId,
      pop_device_id: popId,
      mode,
      local_wan_gateway: localWan,
      cpe_overlay: cpeOverlay || undefined,
      pop_public_key: popKey || undefined,
      cpe_public_key: cpePub || undefined,
      dry_run: dryRun,
      confirm_timeout_sec: 90,
    };
  }

  function overseasGateway(): string {
    return popPeer || proposal?.pop_peer || "";
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
      applyProposal(res.proposal, res.fabric);
      setResult({
        dry_run: true,
        fabric: res.fabric,
        proposal: res.proposal,
        cpe_desired: res.cpe_desired,
        pop_desired: res.pop_desired,
        cpe_plan: res.cpe_plan,
        pop_plan: res.pop_plan,
      });
      setCsResult(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "WG 协商生成失败");
    } finally { setBusy(false); }
  }

  async function run(dryRun: boolean) {
    setError(null);
    if (!cpeId) return setError("请选择客户侧设备 (CPE)");
    if (needsPop && !popId) return setError("请选择 POP");
    if (needsPop && !overseasGateway()) return setError("请先点「生成 WG 协商参数」获取隧道对端地址");
    if (isChinaSplit && !chn?.loaded && dryRun) {
      await doRefreshChn();
    }
    setBusy(true);
    try {
      if (!needsPop) {
        const orch = await orchestrate(cpeId, {
          dry_run: dryRun,
          accel: { mode: "domestic_direct", local_wan_gateway: localWan, domestic_dns: ["223.5.5.5", "114.114.114.114"] },
          confirm_timeout_sec: 90,
        }, currentToken());
        setResult({
          dry_run: dryRun,
          cpe_desired: orch.desired,
          cpe_plan: orch.plan,
          deploy: orch.result ? {
            pop_result: { status: "skipped" },
            pop_plan: { changes: [], aggregate_risk: "low" },
            cpe_result: orch.result,
            cpe_plan: orch.plan ?? { changes: [], aggregate_risk: "low" },
            error: orch.error,
          } : undefined,
          error: orch.error,
        });
        setCsResult(null);
        return;
      }

      const fabric = await deployFabric(fabricBody(dryRun), currentToken());
      setResult(fabric);

      if (isChinaSplit) {
        const cs = await chinaSplit(cpeId, {
          dry_run: dryRun,
          wan_gateway: localWan,
          overseas_gateway: overseasGateway(),
        }, currentToken());
        setCsResult(cs);
        if (!dryRun && cs.status !== "delivered" && cs.error) {
          setError(cs.error);
        }
      } else {
        setCsResult(null);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "请求失败");
      setResult(null);
      setCsResult(null);
    } finally { setBusy(false); }
  }

  const previewDesired: ConfigState | undefined = needsPop && previewSide !== "routes"
    ? (previewSide === "cpe" ? result?.cpe_desired : result?.pop_desired)
    : !needsPop ? result?.cpe_desired : undefined;
  const previewPlan: ConfigPlan | undefined = needsPop && previewSide !== "routes"
    ? (previewSide === "cpe" ? result?.cpe_plan : result?.pop_plan)
    : result?.cpe_plan;

  const deployLabel = isChinaSplit
    ? "双向下发隧道 + 国内外分流"
    : needsPop ? "双向下发 CPE + POP" : "下发到 CPE";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">加速业务配置</h1>
        <p className="mt-1 text-sm text-muted">
          选择 CPE 与骨干 POP，自动生成 WireGuard 隧道，双侧预览后一键下发（托管凭据，无需手填密码）
        </p>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        {MODES.map((m) => (
          <button key={m.id} onClick={() => { setMode(m.id); setProposal(null); setResult(null); setCsResult(null); setPreviewSide("cpe"); }}
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
          <CardHeader title="1 · 设备与 POP" subtitle="从已托管设备中选择" />
          <SelectField icon={RouterIcon} label="客户侧设备 (CPE)" value={cpeId} onChange={setCpeId}
            options={cpes.map((d) => ({ value: d.id, label: `${d.name} · ${d.mgmt_address}` }))} />
          {needsPop && (
            <SelectField icon={Server} label="接入骨干 / 出口 (POP)" value={popId} onChange={setPopId}
              options={pops.map((d) => ({ value: d.id, label: `${d.name} · ${d.region || d.role} · ${d.mgmt_address}` }))}
              emptyHint="暂无 POP，请先在「骨干节点」登记" />
          )}

          {needsPop && (
            <>
              <CardHeader title="2 · WireGuard 隧道（CPE ↔ POP）" subtitle="自动生成双侧密钥与 overlay 地址" />
              <div className="grid grid-cols-2 gap-3">
                <ReadonlyField label="CPE 隧道接口" value={tunnel || (pop ? tunnelNameForPop(pop.name) : "—")} />
                <ReadonlyField label="POP 隧道接口" value={popIface || (cpe ? `wg-cpe-${cpe.name}`.slice(0, 24) : "—")} />
                <ReadonlyField label="POP 端点" value={popEndpoint || (pop ? hostFromMgmt(pop.mgmt_address) : "—")} />
                <ReadonlyField label="隧道对端（海外网关）" value={popPeer || "生成后自动填充"} />
                <Field label="CPE 隧道地址" value={cpeOverlay} onChange={setCpeOverlay} mono placeholder="生成后自动填充" />
                <div className="col-span-2">
                  <Field label="POP WireGuard 公钥" value={popKey} onChange={setPopKey} mono placeholder="可自动从 POP 读取" />
                </div>
                {cpePub && (
                  <div className="col-span-2 rounded-lg border border-border/60 bg-elevated/40 p-3 text-xs">
                    <div className="mb-1 flex items-center gap-1.5 font-medium text-muted"><KeyRound className="h-3.5 w-3.5" /> CPE 密钥（已生成）</div>
                    <div className="font-mono break-all text-foreground/80">公钥 {cpePub}</div>
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

          {(isChinaSplit || mode === "domestic_direct") && (
            <Field label="本地 WAN 网关（国内直连出口）" value={localWan} onChange={setLocalWan} mono />
          )}

          {isChinaSplit && (
            <div className="rounded-lg border border-border bg-elevated/40 p-3 text-sm">
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-2 font-medium">
                  <ListTree className="h-4 w-4 text-primary" /> 国内路由表 (chnroutes)
                </div>
                <button onClick={doRefreshChn} disabled={refreshingChn}
                  className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs hover:border-primary disabled:opacity-60">
                  {refreshingChn ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
                  刷新
                </button>
              </div>
              <p className="mt-2 text-xs text-muted">
                {chn?.loaded
                  ? <>已加载 <strong className="text-foreground">{chn.count.toLocaleString()}</strong> 条国内网段 · 海外经隧道走 0.0.0.0/1 + 128.0.0.0/1</>
                  : "尚未加载，点「刷新」从 chnroutes2 拉取国内网段路由表"}
              </p>
              {chn?.updated_at && (
                <p className="mt-1 text-xs text-muted">更新于 {new Date(chn.updated_at).toLocaleString()}</p>
              )}
            </div>
          )}

          <div className="flex flex-wrap gap-2 pt-1">
            <button onClick={() => run(true)} disabled={busy}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览
            </button>
            <button onClick={() => run(false)} disabled={busy}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
              {deployLabel}
            </button>
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
        </Card>

        <Card>
          <CardHeader
            title="配置预览 / 下发结果"
            subtitle={cpe && pop && needsPop ? `${cpe.name} ⇄ ${pop.name}` : cpe?.name ?? "选择设备后生成"}
            action={previewSide !== "routes" && previewPlan
              ? <Badge tone={riskTone[previewPlan.aggregate_risk] ?? "neutral"}>风险 {previewPlan.aggregate_risk}</Badge>
              : csResult?.route_count
                ? <Badge tone="primary">{csResult.route_count.toLocaleString()} 条路由</Badge>
                : undefined}
          />
          {needsPop && (
            <div className="mb-3 flex flex-wrap gap-2">
              <SideTab active={previewSide === "cpe"} onClick={() => setPreviewSide("cpe")} label={`CPE · ${cpe?.name ?? "—"}`} />
              <SideTab active={previewSide === "pop"} onClick={() => setPreviewSide("pop")} label={`POP · ${pop?.name ?? "—"}`} />
              {isChinaSplit && (
                <SideTab active={previewSide === "routes"} onClick={() => setPreviewSide("routes")} label="国内外分流路由" />
              )}
            </div>
          )}
          {result?.deploy && (
            <div className="mb-3 space-y-2 text-sm">
              {needsPop && <DeployStatus label="POP 隧道" res={result.deploy.pop_result} />}
              <DeployStatus label="CPE 隧道" res={result.deploy.cpe_result} />
              {csResult?.status && (
                <DeployStatus label="国内外分流" res={{ status: csResult.status === "delivered" ? "committed" : "failed", reason: csResult.error }} />
              )}
            </div>
          )}
          {result?.error && <p className="mb-2 text-sm text-danger">{result.error}</p>}
          {previewSide === "routes" && csResult?.script ? (
            <div className="space-y-2">
              <div className="flex flex-wrap gap-2 text-xs">
                <Badge tone="primary">总路由 {csResult.route_count ?? 0}</Badge>
                <Badge tone="success">国内 {csResult.domestic_count ?? 0}</Badge>
                <Badge tone="neutral">海外 {(csResult.overseas_halves ?? ["0.0.0.0/1", "128.0.0.0/1"]).join(" / ")}</Badge>
              </div>
              <pre className="max-h-[480px] overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">{csResult.script}</pre>
            </div>
          ) : previewDesired ? (
            <ConfigPreview state={previewDesired} />
          ) : (
            <p className="py-16 text-center text-sm text-muted">
              {needsPop ? "选择设备并生成 WG 参数后点「预览」" : "选择 CPE 后点「预览」"}
            </p>
          )}
        </Card>
      </div>
    </div>
  );
}

function SideTab({ active, onClick, label }: { active: boolean; onClick: () => void; label: string }) {
  return (
    <button onClick={onClick}
      className={`rounded-md border px-2.5 py-1 text-xs ${active ? "border-primary bg-primary/10 text-primary" : "border-border text-muted hover:border-primary/50"}`}>
      {label}
    </button>
  );
}

function DeployStatus({ label, res }: { label: string; res: { status: string; reason?: string } }) {
  const ok = res.status === "committed" || res.status === "skipped";
  return (
    <div className={`rounded-lg border p-2 ${ok ? "border-success/40 bg-success/10 text-success" : "border-danger/40 bg-danger/10 text-danger"}`}>
      {label}：{res.status}{res.reason ? ` · ${res.reason}` : ""}
    </div>
  );
}

function ConfigPreview({ state }: { state: ConfigState }) {
  return (
    <pre className="max-h-[520px] overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">
      {state.statements.map((st) => (
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
