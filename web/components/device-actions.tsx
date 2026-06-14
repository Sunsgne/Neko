"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { LinkIcon, RefreshCw, Loader2 } from "lucide-react";
import { enrollDevice, pollDevice, ApiError } from "@/lib/api";
import { currentToken } from "@/lib/session";

export function DeviceActions({ deviceId, enrolled, compact }: { deviceId: string; enrolled: boolean; compact?: boolean }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [user, setUser] = React.useState("admin");
  const [pass, setPass] = React.useState("");
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  async function doEnroll(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await enrollDevice(deviceId, user.trim(), pass, currentToken());
      setOpen(false);
      setPass("");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "纳管失败");
    } finally {
      setBusy(false);
    }
  }

  async function doPoll() {
    setBusy(true);
    setError(null);
    try {
      await pollDevice(deviceId, currentToken());
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "轮询失败");
    } finally {
      setBusy(false);
    }
  }

  const btn = compact
    ? "rounded-md border border-border px-2 py-1 text-xs hover:border-primary"
    : "flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm";

  return (
    <span className="inline-flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
      {enrolled ? (
        <button onClick={doPoll} disabled={busy} className={`${btn} ${compact ? "" : "border border-border hover:border-primary"}`} title="立即轮询状态">
          {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />} {compact ? "" : "轮询"}
        </button>
      ) : (
        <button onClick={() => setOpen(true)} className={`${btn} ${compact ? "border-primary/60 text-primary" : "bg-primary text-background font-medium hover:opacity-90"}`} title="录入凭据并纳管">
          <LinkIcon className="h-3.5 w-3.5" /> 纳管
        </button>
      )}
      {error && <span className="text-xs text-danger">{error}</span>}

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={() => setOpen(false)}>
          <form onClick={(e) => e.stopPropagation()} onSubmit={doEnroll} className="card w-full max-w-sm space-y-4">
            <h3 className="text-sm font-semibold">纳管设备</h3>
            <p className="text-xs text-muted">平台将通过 RouterOS REST 连接设备、读取能力并加密保存凭据，之后无需登录设备即可管理。</p>
            <div>
              <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">用户名</label>
              <input value={user} onChange={(e) => setUser(e.target.value)} autoFocus
                className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary" required />
            </div>
            <div>
              <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">密码</label>
              <input type="password" value={pass} onChange={(e) => setPass(e.target.value)}
                className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary" />
            </div>
            {error && <p className="text-sm text-danger">{error}</p>}
            <div className="flex justify-end gap-2">
              <button type="button" onClick={() => setOpen(false)} className="rounded-lg border border-border px-3 py-2 text-sm text-muted">取消</button>
              <button type="submit" disabled={busy} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
                {busy && <Loader2 className="h-4 w-4 animate-spin" />} 连接并纳管
              </button>
            </div>
          </form>
        </div>
      )}
    </span>
  );
}
