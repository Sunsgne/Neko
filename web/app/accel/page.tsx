"use client";

import * as React from "react";
import { Rocket, Loader2 } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import { previewAccel, type AccelPreview, ApiError } from "@/lib/api";
import { currentToken } from "@/lib/session";

const MODES = [
  { id: "overseas_direct", title: "海外运营（直连）", desc: "全量流量直接走海外出口 IP，不做分流" },
  { id: "smart_split", title: "智能分流", desc: "国内直连本地出口，海外走 SD-WAN 隧道" },
  { id: "domestic_direct", title: "国内直连", desc: "全量走本地出口" },
];

const riskTone: Record<string, "success" | "primary" | "warning" | "danger"> = {
  low: "success", medium: "primary", high: "warning", critical: "danger",
};

export default function AccelPage() {
  const [mode, setMode] = React.useState("overseas_direct");
  const [tunnel, setTunnel] = React.useState("wg-hk");
  const [gw, setGw] = React.useState("100.64.0.1");
  const [exitIP, setExitIP] = React.useState("203.0.113.9");
  const [wan, setWan] = React.useState("192.168.1.1");
  const [overseasDNS, setOverseasDNS] = React.useState("8.8.8.8,1.1.1.1");
  const [domesticDNS, setDomesticDNS] = React.useState("223.5.5.5,114.114.114.114");
  const [preview, setPreview] = React.useState<AccelPreview | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(false);

  async function run() {
    setError(null);
    setLoading(true);
    try {
      const profile = {
        mode,
        tunnel_interface: tunnel,
        overseas_gateway: gw,
        overseas_exit_ip: exitIP,
        local_wan_gateway: wan,
        overseas_dns: overseasDNS.split(",").map((s) => s.trim()).filter(Boolean),
        domestic_dns: domesticDNS.split(",").map((s) => s.trim()).filter(Boolean),
      };
      setPreview(await previewAccel(profile, currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "预览失败");
      setPreview(null);
    } finally {
      setLoading(false);
    }
  }

  const overseas = mode === "overseas_direct";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">加速业务配置</h1>
        <p className="mt-1 text-sm text-muted">客户侧设备加速模式 · 生成 RouterOS 配置（预览后可下发到设备）</p>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        {MODES.map((m) => (
          <button
            key={m.id}
            onClick={() => setMode(m.id)}
            className={`card text-left transition-colors ${mode === m.id ? "border-primary shadow-glow" : "hover:border-border"}`}
          >
            <div className="flex items-center gap-2 text-sm font-semibold">
              <Rocket className="h-4 w-4 text-primary" /> {m.title}
            </div>
            <p className="mt-1 text-xs text-muted">{m.desc}</p>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader title="参数" subtitle={overseas ? "海外直连（不分流）" : "分流 / 国内"} />
          <div className="space-y-3">
            <Field label="隧道接口" value={tunnel} onChange={setTunnel} mono />
            <Field label="海外网关（下一跳）" value={gw} onChange={setGw} mono />
            {overseas && <Field label="海外出口 IP" value={exitIP} onChange={setExitIP} mono />}
            {!overseas && <Field label="本地 WAN 网关" value={wan} onChange={setWan} mono />}
            {overseas ? (
              <Field label="海外 DNS（逗号分隔）" value={overseasDNS} onChange={setOverseasDNS} mono />
            ) : (
              <Field label="国内 DNS（逗号分隔）" value={domesticDNS} onChange={setDomesticDNS} mono />
            )}
            <button
              onClick={run}
              disabled={loading}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60"
            >
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />} 生成预览
            </button>
            {error && <p className="text-sm text-danger">{error}</p>}
          </div>
        </Card>

        <Card>
          <CardHeader
            title="生成的 RouterOS 配置"
            subtitle={preview ? preview.desc : "点击生成预览"}
            action={preview ? <Badge tone={riskTone[preview.plan.aggregate_risk] ?? "neutral"}>风险 {preview.plan.aggregate_risk}</Badge> : undefined}
          />
          {preview ? (
            <pre className="max-h-[420px] overflow-auto rounded-lg border border-border bg-elevated/50 p-3 text-xs leading-relaxed">
              {preview.state.statements.map((st) => (
                <div key={st.path + st.key}>
                  <span className="text-primary">{st.path}</span>{" "}
                  <span className="text-muted">{st.key}</span>
                  {"\n"}
                  {Object.entries(st.attributes).map(([k, v]) => (
                    <div key={k} className="pl-4 text-foreground/80">
                      {k} = {v}
                    </div>
                  ))}
                </div>
              ))}
            </pre>
          ) : (
            <p className="py-12 text-center text-sm text-muted">尚无预览</p>
          )}
        </Card>
      </div>
    </div>
  );
}

function Field({ label, value, onChange, mono }: { label: string; value: string; onChange: (v: string) => void; mono?: boolean }) {
  return (
    <div>
      <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">{label}</label>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={`w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary ${mono ? "font-mono" : ""}`}
      />
    </div>
  );
}
