"use client";

import * as React from "react";
import { Workflow, Eye, Send, Loader2, Server, Router as RouterIcon } from "lucide-react";
import { Card, CardHeader, Badge, EmptyState, PreviewPanelHeader } from "@/components/ui";
import {
  listDevices, deployFabric,
  type Device, type FabricDeployResult, ApiError,
} from "@/lib/api";
import { hostFromMgmt, popPeerOf, tunnelNameForPop } from "@/lib/tunnel";
import { currentToken } from "@/lib/session";

const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

type PreviewSide = "cpe" | "pop";

export default function OrchestratePage() {
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [cpeId, setCpeId] = React.useState("");
  const [popId, setPopId] = React.useState("");
  const [cpeOverlay, setCpeOverlay] = React.useState("100.64.0.2/30");
  const [popPeer, setPopPeer] = React.useState("100.64.0.1");
  const [popKey, setPopKey] = React.useState("");
  const [internalCidr, setInternalCidr] = React.useState("10.0.0.0/8");

  const [fabricResult, setFabricResult] = React.useState<FabricDeployResult | null>(null);
  const [previewSide, setPreviewSide] = React.useState<PreviewSide>("cpe");
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    listDevices(currentToken()).then((d) => {
      setDevices(d.filter((x) => x.trust_state === "managed"));
    }).catch(() => setDevices([]));
  }, []);

  const cpes = devices.filter((d) => d.role !== "backbone" && d.role !== "gateway");
  const pops = devices.filter((d) => d.role === "backbone" || d.role === "gateway");
  const pop = devices.find((d) => d.id === popId);
  const cpe = devices.find((d) => d.id === cpeId);

  React.useEffect(() => { setPopPeer(popPeerOf(cpeOverlay)); }, [cpeOverlay]);

  function fabricBody(dryRun: boolean): Parameters<typeof deployFabric>[0] {
    return {
      cpe_device_id: cpeId,
      pop_device_id: popId,
      mode: "mesh",
      cpe_overlay: cpeOverlay,
      pop_public_key: popKey || undefined,
      overlay_routes: internalCidr.split(",").map((s) => s.trim()).filter(Boolean),
      dry_run: dryRun,
      confirm_timeout_sec: 90,
    };
  }

  async function run(dryRun: boolean) {
    setError(null);
    if (!cpeId) return setError("请选择接入设备 (CPE)");
    if (!popId) return setError("请选择接入的骨干 / 出口节点 (POP)");
    setBusy(true);
    try {
      setFabricResult(await deployFabric(fabricBody(dryRun), currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "请求失败");
      setFabricResult(null);
    } finally { setBusy(false); }
  }

  const previewDesired = fabricResult
    ? (previewSide === "cpe" ? fabricResult.cpe_desired : fabricResult.pop_desired)
    : undefined;
  const previewPlan = fabricResult
    ? (previewSide === "cpe" ? fabricResult.cpe_plan : fabricResult.pop_plan)
    : undefined;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">组网下发 · 内网打通</h1>
        <p className="mt-1 text-sm text-muted">
          将客户站点 (CPE) 经 WireGuard 隧道接入骨干 POP，宣告内网网段实现站点间互通。加速与分流请前往「加速」页配置。
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="space-y-4">
          <CardHeader title="1 · 站点与接入节点" />
          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-xs uppercase tracking-wide text-muted"><RouterIcon className="h-3.5 w-3.5" /> 接入设备 (CPE)</label>
            <select value={cpeId} onChange={(e) => setCpeId(e.target.value)} className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              <option value="">请选择…</option>
              {cpes.map((d) => <option key={d.id} value={d.id}>{d.name} · {d.mgmt_address}</option>)}
            </select>
          </div>
          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-xs uppercase tracking-wide text-muted"><Server className="h-3.5 w-3.5" /> 接入骨干节点 (POP)</label>
            <select value={popId} onChange={(e) => setPopId(e.target.value)} className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
              <option value="">请选择…</option>
              {pops.map((d) => <option key={d.id} value={d.id}>{d.name} · {d.region || d.role} · {d.mgmt_address}</option>)}
            </select>
            {pops.length === 0 && <p className="mt-1 text-xs text-warning">暂无骨干节点，请先在「骨干节点」登记。</p>}
          </div>

          <CardHeader title="2 · 内网网段" subtitle="经隧道可达的客户内网前缀，逗号分隔" />
          <F label="内网网段（CIDR）" v={internalCidr} on={setInternalCidr} mono placeholder="10.0.0.0/8, 192.168.0.0/16" />

          <CardHeader title="3 · 隧道参数" subtitle="WireGuard 隧道，端点取自所选 POP" />
          <div className="grid grid-cols-2 gap-3">
            <F label="CPE 隧道地址" v={cpeOverlay} on={setCpeOverlay} mono />
            <F label="POP 对端地址" v={popPeer} on={setPopPeer} mono />
            <F label="POP 端点(自动)" v={pop ? hostFromMgmt(pop.mgmt_address) : "—"} on={() => {}} mono disabled />
            <F label="CPE 隧道接口(自动)" v={pop ? tunnelNameForPop(pop.name) : "—"} on={() => {}} mono disabled />
            <div className="col-span-2"><F label="POP WireGuard 公钥（可选）" v={popKey} on={setPopKey} mono placeholder="留空则下发后在 POP 端补充" /></div>
          </div>

          <div className="flex gap-2">
            <button onClick={() => run(true)} disabled={busy} className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览
            </button>
            <button onClick={() => run(false)} disabled={busy} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
              双向下发 CPE + POP
            </button>
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
        </Card>

        <Card className="flex min-h-[520px] flex-col overflow-hidden p-0">
          <PreviewPanelHeader
            title="配置预览"
            context={fabricResult && cpe && pop ? `${cpe.name} ⇄ ${pop.name}` : undefined}
            action={previewPlan ? <Badge tone={riskTone[previewPlan.aggregate_risk] ?? "neutral"}>风险 {previewPlan.aggregate_risk}</Badge> : undefined}
          />
          {fabricResult && (
            <div className="flex gap-2 px-4 pb-3">
              <button onClick={() => setPreviewSide("cpe")}
                className={`rounded-md border px-2.5 py-1 text-xs ${previewSide === "cpe" ? "border-primary bg-primary/10 text-primary" : "border-border text-muted"}`}>
                CPE · {cpe?.name ?? "—"}
              </button>
              <button onClick={() => setPreviewSide("pop")}
                className={`rounded-md border px-2.5 py-1 text-xs ${previewSide === "pop" ? "border-primary bg-primary/10 text-primary" : "border-border text-muted"}`}>
                POP · {pop?.name ?? "—"}
              </button>
            </div>
          )}
          {!fabricResult ? (
            <EmptyState
              icon={<Workflow />}
              title="暂无预览"
              description="选择 CPE 与 POP 后，点「预览」生成隧道与内网路由配置"
            />
          ) : (
            <>
          {fabricResult.deploy && (
            <div className="mb-3 space-y-2 px-4 text-sm">
              <div className={`rounded-lg border p-2 ${fabricResult.deploy.pop_result.status === "committed" ? "border-success/40 bg-success/10 text-success" : "border-danger/40 bg-danger/10 text-danger"}`}>
                POP：{fabricResult.deploy.pop_result.status}{fabricResult.deploy.pop_result.reason ? ` · ${fabricResult.deploy.pop_result.reason}` : ""}
              </div>
              <div className={`rounded-lg border p-2 ${fabricResult.deploy.cpe_result.status === "committed" ? "border-success/40 bg-success/10 text-success" : "border-danger/40 bg-danger/10 text-danger"}`}>
                CPE：{fabricResult.deploy.cpe_result.status}{fabricResult.deploy.cpe_result.reason ? ` · ${fabricResult.deploy.cpe_result.reason}` : ""}
              </div>
            </div>
          )}
          {fabricResult.error && <p className="mb-2 px-4 text-sm text-danger">{fabricResult.error}</p>}
          {previewDesired && (
            <pre className="mx-4 mb-4 max-h-[460px] overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">
              {previewDesired.statements.map((st) => (
                <div key={st.path + st.key} className="mb-1">
                  <span className="text-primary">{st.path}</span> <span className="text-muted">{st.key}</span>
                  {Object.entries(st.attributes).map(([k, v]) => <div key={k} className="pl-4 text-foreground/80">{k} = {v}</div>)}
                </div>
              ))}
            </pre>
          )}
            </>
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
