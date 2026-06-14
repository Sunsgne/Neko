"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Plus, Loader2 } from "lucide-react";
import { registerDevice, ApiError } from "@/lib/api";
import { currentToken } from "@/lib/session";

export function RegisterDeviceButton() {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [name, setName] = React.useState("");
  const [addr, setAddr] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await registerDevice(name.trim(), addr.trim(), currentToken());
      setOpen(false);
      setName("");
      setAddr("");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "登记失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90"
      >
        <Plus className="h-4 w-4" /> 登记设备
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={() => setOpen(false)}>
          <form onClick={(e) => e.stopPropagation()} onSubmit={submit} className="card w-full max-w-sm space-y-4">
            <h3 className="text-sm font-semibold">登记设备（进入纳管流程）</h3>
            <div>
              <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">名称</label>
              <input
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="例如：edge-sh-03"
                className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary"
                required
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">管理地址</label>
              <input
                value={addr}
                onChange={(e) => setAddr(e.target.value)}
                placeholder="例如：10.10.1.3"
                className="w-full rounded-lg border border-border bg-elevated px-3 py-2 font-mono text-sm outline-none focus:border-primary"
                required
              />
            </div>
            {error && <p className="text-sm text-danger">{error}</p>}
            <div className="flex justify-end gap-2">
              <button type="button" onClick={() => setOpen(false)} className="rounded-lg border border-border px-3 py-2 text-sm text-muted hover:text-foreground">
                取消
              </button>
              <button type="submit" disabled={loading} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
                {loading && <Loader2 className="h-4 w-4 animate-spin" />} 登记
              </button>
            </div>
          </form>
        </div>
      )}
    </>
  );
}
