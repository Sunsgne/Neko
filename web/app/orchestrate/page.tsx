"use client";

import * as React from "react";
import {
  Workflow, Eye, Send, Loader2, Server, Router as RouterIcon,
  Plus, Trash2, GitBranch, Network,
} from "lucide-react";
import { Card, CardHeader, Badge, EmptyState, PreviewPanelHeader } from "@/components/ui";
import {
  listDevices, deployMesh,
  type Device, type MeshDeployResult, type MeshTopology, type MeshSiteInput, ApiError,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

const TOPOLOGIES: { id: MeshTopology; title: string; desc: string; diagram: string }[] = [
  {
    id: "hub_spoke",
    title: "Hub-Spoke",
    desc: "各站点接入就近 POP，骨干间 iBGP 互联，适合多分支",
    diagram: "CPE₁→POP₁↔POP₂←CPE₂",
  },
  {
    id: "transit",
    title: "Transit 多跳",
    desc: "设备→骨干→骨干→设备，跨域站点经骨干链中转",
    diagram: "CPE₁→POP₁→POP₂→CPE₂",
  },
  {
    id: "full_mesh",
    title: "骨干 Full Mesh",
    desc: "所有骨干节点全互联 (WG + iBGP)，站点经 home POP 可达全网",
    diagram: "POP₁↔POP₂↔POP₃ …",
  },
];

const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

type SiteRow = { key: string; cpeId: string; popId: string; prefixes: string };

function newSite(): SiteRow {
  return { key: Math.random().toString(36).slice(2), cpeId: "", popId: "", prefixes: "10.0.0.0/24" };
}

export default function OrchestratePage() {
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [topology, setTopology] = React.useState<MeshTopology>("transit");
  const [localAS, setLocalAS] = React.useState("65001");
  const [sites, setSites] = React.useState<SiteRow[]>([newSite(), newSite()]);
  const [backbonePath, setBackbonePath] = React.useState<string[]>([]);

  const [meshResult, setMeshResult] = React.useState<MeshDeployResult | null>(null);
  const [previewNode, setPreviewNode] = React.useState<string>("");
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    listDevices(currentToken()).then((d) => {
      setDevices(d.filter((x) => x.trust_state === "managed"));
    }).catch(() => setDevices([]));
  }, []);

  const cpes = devices.filter((d) => d.role !== "backbone" && d.role !== "gateway");
  const pops = devices.filter((d) => d.role === "backbone" || d.role === "gateway");

  // Auto-build backbone path from unique POPs when transit/full_mesh
  React.useEffect(() => {
    const used = [...new Set(sites.map((s) => s.popId).filter(Boolean))];
    if (topology === "transit" && used.length >= 2) {
      setBackbonePath(used);
    } else if (topology === "full_mesh" && used.length >= 2) {
      setBackbonePath(used);
    }
  }, [sites, topology]);

  function updateSite(key: string, patch: Partial<SiteRow>) {
    setSites((rows) => rows.map((r) => (r.key === key ? { ...r, ...patch } : r)));
  }

  function buildSitesPayload(): MeshSiteInput[] {
    return sites
      .filter((s) => s.cpeId && s.popId)
      .map((s) => ({
        device_id: s.cpeId,
        pop_device_id: s.popId,
        prefixes: s.prefixes.split(",").map((p) => p.trim()).filter(Boolean),
      }));
  }

  async function run(dryRun: boolean) {
    setError(null);
    const payload = buildSitesPayload();
    if (payload.length < 2) {
      setError("至少配置 2 个有效站点（CPE + 归属 POP + 内网前缀）");
      return;
    }
    if (topology === "transit" && backbonePath.length < 2) {
      setError("Transit 拓扑需要至少 2 个骨干节点，请配置骨干路径");
      return;
    }
    setBusy(true);
    try {
      const res = await deployMesh({
        topology,
        local_as: parseInt(localAS, 10) || 65001,
        sites: payload,
        backbone_path: topology !== "hub_spoke" ? backbonePath : undefined,
        dry_run: dryRun,
        confirm_timeout_sec: 90,
      }, currentToken());
      setMeshResult(res);
      if (res.mesh?.nodes?.[0]) setPreviewNode(res.mesh.nodes[0].device_id);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "请求失败");
      setMeshResult(null);
    } finally { setBusy(false); }
  }

  const mesh = meshResult?.mesh;
  const activeNode = mesh?.nodes.find((n) => n.device_id === previewNode);
  const deviceName = (id: string) => devices.find((d) => d.id === id)?.name ?? id;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">组网下发 · SD-WAN 互联</h1>
        <p className="mt-1 text-sm text-muted">
          多站点经骨干组网：支持 Hub-Spoke、Transit（设备→骨干→骨干→设备）、骨干 Full Mesh。加速策略请前往「加速」页。
        </p>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        {TOPOLOGIES.map((t) => (
          <button key={t.id} onClick={() => { setTopology(t.id); setMeshResult(null); }}
            className={`card text-left transition-colors ${topology === t.id ? "border-primary shadow-glow" : "hover:border-border"}`}>
            <div className="flex items-center gap-2 text-sm font-semibold">
              <Network className="h-4 w-4 text-primary" /> {t.title}
            </div>
            <p className="mt-1 font-mono text-[10px] text-primary/80">{t.diagram}</p>
            <p className="mt-1 text-xs text-muted">{t.desc}</p>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="space-y-4">
          <Card className="space-y-3">
            <CardHeader title="站点列表" subtitle="每个站点：CPE 设备 + 归属 POP + 要互通的内网前缀" />
            {sites.map((row, idx) => (
              <div key={row.key} className="rounded-lg border border-border bg-elevated/30 p-3 space-y-2">
                <div className="flex items-center justify-between text-xs font-medium text-muted">
                  <span>站点 {idx + 1}</span>
                  {sites.length > 2 && (
                    <button onClick={() => setSites((s) => s.filter((r) => r.key !== row.key))}
                      className="text-danger hover:opacity-80"><Trash2 className="h-3.5 w-3.5" /></button>
                  )}
                </div>
                <Select label="CPE 设备" icon={RouterIcon} value={row.cpeId} onChange={(v) => updateSite(row.key, { cpeId: v })}
                  options={cpes.map((d) => ({ value: d.id, label: `${d.name} · ${d.mgmt_address}` }))} />
                <Select label="归属 POP" icon={Server} value={row.popId} onChange={(v) => updateSite(row.key, { popId: v })}
                  options={pops.map((d) => ({ value: d.id, label: `${d.name} · ${d.region || d.role}` }))} />
                <Field label="内网前缀（逗号分隔）" value={row.prefixes} onChange={(v) => updateSite(row.key, { prefixes: v })} mono />
              </div>
            ))}
            <button onClick={() => setSites((s) => [...s, newSite()])}
              className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-dashed border-border py-2 text-sm text-muted hover:border-primary hover:text-primary">
              <Plus className="h-4 w-4" /> 添加站点
            </button>
          </Card>

          {(topology === "transit" || topology === "full_mesh") && (
            <Card className="space-y-3">
              <CardHeader
                title="骨干路径"
                subtitle={topology === "transit" ? "Transit 顺序：POP₁ → POP₂ → …" : "参与 Full Mesh 的骨干节点"}
              />
              <div className="flex flex-wrap gap-2">
                {backbonePath.map((id, i) => (
                  <div key={id} className="flex items-center gap-1 rounded-md border border-border bg-elevated px-2 py-1 text-xs">
                    {topology === "transit" && i > 0 && <GitBranch className="h-3 w-3 text-muted" />}
                    <span>{deviceName(id)}</span>
                    <button onClick={() => setBackbonePath((p) => p.filter((x) => x !== id))}
                      className="ml-1 text-muted hover:text-danger">×</button>
                  </div>
                ))}
              </div>
              <select
                value=""
                onChange={(e) => {
                  const id = e.target.value;
                  if (id && !backbonePath.includes(id)) setBackbonePath((p) => [...p, id]);
                }}
                className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
                <option value="">+ 添加骨干节点到路径…</option>
                {pops.filter((p) => !backbonePath.includes(p.id)).map((d) => (
                  <option key={d.id} value={d.id}>{d.name}</option>
                ))}
              </select>
            </Card>
          )}

          <Card className="space-y-3">
            <Field label="BGP AS 号" value={localAS} onChange={setLocalAS} mono />
            <div className="flex gap-2">
              <button onClick={() => run(true)} disabled={busy}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary disabled:opacity-60">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} 预览
              </button>
              <button onClick={() => run(false)} disabled={busy}
                className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                按序下发全网
              </button>
            </div>
            {error && <p className="text-sm text-danger">{error}</p>}
          </Card>
        </div>

        <Card className="flex min-h-[560px] flex-col overflow-hidden p-0">
          <PreviewPanelHeader
            title="组网预览"
            context={mesh?.summary}
            action={activeNode?.plan
              ? <Badge tone={riskTone[activeNode.plan.aggregate_risk] ?? "neutral"}>风险 {activeNode.plan.aggregate_risk}</Badge>
              : undefined}
          />

          {mesh && (
            <>
              <div className="border-b border-border px-4 py-2">
                <p className="mb-2 text-xs uppercase tracking-wide text-muted">Overlay 链路</p>
                <div className="space-y-1 text-xs">
                  {mesh.links.map((l, i) => (
                    <div key={i} className="font-mono text-foreground/80">
                      <Badge tone={l.kind === "pop_pop" ? "primary" : "neutral"}>{l.kind === "pop_pop" ? "骨干" : "站点"}</Badge>
                      {" "}{deviceName(l.from_device_id)} → {deviceName(l.to_device_id)}
                      {l.to_gateway && <span className="text-muted"> · gw {l.to_gateway}</span>}
                    </div>
                  ))}
                </div>
              </div>
              <div className="flex flex-wrap gap-2 px-4 py-3">
                {mesh.nodes.map((n) => (
                  <button key={n.device_id} onClick={() => setPreviewNode(n.device_id)}
                    className={`rounded-md border px-2.5 py-1 text-xs ${previewNode === n.device_id ? "border-primary bg-primary/10 text-primary" : "border-border text-muted"}`}>
                    {n.role === "backbone" || n.role === "gateway" ? "POP" : "CPE"} · {n.device_name}
                  </button>
                ))}
              </div>
            </>
          )}

          {meshResult?.deploy && (
            <div className="space-y-1 px-4 pb-2 text-xs">
              {meshResult.deploy.results.map((r) => (
                <div key={r.device_id}
                  className={`rounded border px-2 py-1 ${r.result.status === "committed" ? "border-success/40 text-success" : "border-danger/40 text-danger"}`}>
                  {deviceName(r.device_id)}：{r.result.status}{r.result.reason ? ` · ${r.result.reason}` : ""}
                </div>
              ))}
            </div>
          )}

          {!mesh ? (
            <EmptyState icon={<Workflow />} title="暂无预览" description="配置至少 2 个站点后点「预览」" />
          ) : activeNode ? (
            <pre className="mx-4 mb-4 max-h-[360px] flex-1 overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">
              {activeNode.desired.statements.map((st) => (
                <div key={st.path + st.key} className="mb-1">
                  <span className="text-primary">{st.path}</span> <span className="text-muted">{st.key}</span>
                  {Object.entries(st.attributes).map(([k, v]) => (
                    <div key={k} className="pl-4 text-foreground/80">{k} = {k === "private-key" ? `${String(v).slice(0, 12)}…` : v}</div>
                  ))}
                </div>
              ))}
            </pre>
          ) : null}
        </Card>
      </div>
    </div>
  );
}

function Select({ label, icon: Icon, value, onChange, options }: {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  value: string;
  onChange: (v: string) => void;
  options: Array<{ value: string; label: string }>;
}) {
  return (
    <div>
      <label className="mb-1 flex items-center gap-1 text-[11px] uppercase tracking-wide text-muted">
        <Icon className="h-3 w-3" /> {label}
      </label>
      <select value={value} onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-md border border-border bg-elevated px-2 py-1.5 text-sm outline-none focus:border-primary">
        <option value="">请选择…</option>
        {options.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>
    </div>
  );
}

function Field({ label, value, onChange, mono }: { label: string; value: string; onChange: (v: string) => void; mono?: boolean }) {
  return (
    <div>
      <label className="mb-1 block text-[11px] uppercase tracking-wide text-muted">{label}</label>
      <input value={value} onChange={(e) => onChange(e.target.value)}
        className={`w-full rounded-md border border-border bg-elevated px-2 py-1.5 text-sm outline-none focus:border-primary ${mono ? "font-mono" : ""}`} />
    </div>
  );
}
