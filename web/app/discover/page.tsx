"use client";

import * as React from "react";
import { Radar, Loader2, LinkIcon } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import { discover, batchOnboard, type Candidate, ApiError } from "@/lib/api";
import { currentToken } from "@/lib/session";

export default function DiscoverPage() {
  const [cidr, setCidr] = React.useState("");
  const [port, setPort] = React.useState(443);
  const [user, setUser] = React.useState("admin");
  const [pass, setPass] = React.useState("");
  const [cands, setCands] = React.useState<Candidate[]>([]);
  const [sel, setSel] = React.useState<Record<string, boolean>>({});
  const [scanning, setScanning] = React.useState(false);
  const [onboarding, setOnboarding] = React.useState(false);
  const [msg, setMsg] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  async function scan() {
    setError(null);
    setMsg(null);
    setScanning(true);
    setCands([]);
    try {
      const c = await discover({ cidr: cidr.trim(), port, username: user.trim(), password: pass }, currentToken());
      setCands(c);
      setSel(Object.fromEntries(c.map((x) => [x.address, true])));
      setMsg(`发现 ${c.length} 台可达设备`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "扫描失败");
    } finally {
      setScanning(false);
    }
  }

  async function onboard() {
    const chosen = cands.filter((c) => sel[c.address]);
    if (chosen.length === 0) {
      setError("请至少选择一台设备");
      return;
    }
    setError(null);
    setOnboarding(true);
    try {
      const devices = chosen.map((c) => ({
        name: `disc-${c.address.replace(/[.:]/g, "-")}`,
        mgmt_address: c.address,
        role: "cpe" as const,
        region: "",
      }));
      const res = await batchOnboard({ devices, username: user.trim(), password: pass }, currentToken());
      setMsg(`批量纳管完成：创建 ${res.created}，纳管 ${res.enrolled}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "纳管失败");
    } finally {
      setOnboarding(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">发现 · 批量纳管</h1>
        <p className="mt-1 text-sm text-muted">扫描网段发现 RouterOS 设备，选择后用统一凭据批量纳管</p>
      </div>

      <Card className="space-y-3">
        <CardHeader title="网段扫描" subtitle="对该网段每个地址探测 RouterOS REST（HTTPS）" />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-4">
          <Field label="网段 CIDR" value={cidr} onChange={setCidr} placeholder="10.0.0.0/29" mono />
          <Field label="端口" value={String(port)} onChange={(v) => setPort(parseInt(v) || 443)} mono />
          <Field label="用户名" value={user} onChange={setUser} />
          <Field label="密码" value={pass} onChange={setPass} type="password" />
        </div>
        <div className="flex items-center gap-3">
          <button onClick={scan} disabled={scanning || !cidr}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
            {scanning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Radar className="h-4 w-4" />} 扫描
          </button>
          {msg && <span className="text-sm text-success">{msg}</span>}
          {error && <span className="text-sm text-danger">{error}</span>}
        </div>
      </Card>

      {cands.length > 0 && (
        <Card className="p-0">
          <CardHeader title="发现结果" subtitle={`${cands.length} 台`} inset border action={
            <button onClick={onboard} disabled={onboarding}
              className="mr-5 flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {onboarding ? <Loader2 className="h-4 w-4 animate-spin" /> : <LinkIcon className="h-4 w-4" />} 批量纳管
            </button>
          } />
          <table className="w-full text-sm">
            <thead>
              <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                <th className="px-5 py-3 font-medium">选择</th>
                <th className="px-5 py-3 font-medium">地址</th>
                <th className="px-5 py-3 font-medium">型号</th>
                <th className="px-5 py-3 font-medium">版本</th>
                <th className="px-5 py-3 font-medium">架构</th>
              </tr>
            </thead>
            <tbody>
              {cands.map((c) => (
                <tr key={c.address} className="border-b border-border/60">
                  <td className="px-5 py-3">
                    <input type="checkbox" checked={!!sel[c.address]} onChange={(e) => setSel((s) => ({ ...s, [c.address]: e.target.checked }))} />
                  </td>
                  <td className="px-5 py-3 font-mono text-xs">{c.address}</td>
                  <td className="px-5 py-3">{c.board || "—"}</td>
                  <td className="px-5 py-3 font-mono text-xs text-muted">{c.version || "—"}</td>
                  <td className="px-5 py-3 text-muted">{c.arch || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
    </div>
  );
}

function Field({ label, value, onChange, placeholder, mono, type }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string; mono?: boolean; type?: string }) {
  return (
    <div>
      <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">{label}</label>
      <input type={type} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className={`w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary ${mono ? "font-mono" : ""}`} />
    </div>
  );
}
