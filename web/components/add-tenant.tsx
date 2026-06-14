"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Plus, Loader2 } from "lucide-react";
import { createTenant, ApiError } from "@/lib/api";
import { currentToken } from "@/lib/session";

export function AddTenantButton() {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [name, setName] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await createTenant(name.trim(), currentToken());
      setOpen(false);
      setName("");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "创建失败");
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
        <Plus className="h-4 w-4" /> 新建租户
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={() => setOpen(false)}>
          <form
            onClick={(e) => e.stopPropagation()}
            onSubmit={submit}
            className="card w-full max-w-sm space-y-4"
          >
            <h3 className="text-sm font-semibold">新建租户</h3>
            <div>
              <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">名称</label>
              <input
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="例如：Acme Corp"
                className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary"
                required
              />
            </div>
            {error && <p className="text-sm text-danger">{error}</p>}
            <div className="flex justify-end gap-2">
              <button type="button" onClick={() => setOpen(false)} className="rounded-lg border border-border px-3 py-2 text-sm text-muted hover:text-foreground">
                取消
              </button>
              <button type="submit" disabled={loading} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
                {loading && <Loader2 className="h-4 w-4 animate-spin" />} 创建
              </button>
            </div>
          </form>
        </div>
      )}
    </>
  );
}
