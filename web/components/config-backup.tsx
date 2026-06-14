"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Camera, Loader2 } from "lucide-react";
import { takeSnapshot, ApiError, type ConfigSnapshot } from "@/lib/api";
import { currentToken } from "@/lib/session";

export function ConfigBackup({ deviceId, snapshots }: { deviceId: string; snapshots: ConfigSnapshot[] }) {
  const router = useRouter();
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  async function snap() {
    setBusy(true);
    setError(null);
    try {
      await takeSnapshot(deviceId, currentToken());
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "快照失败");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <span className="text-xs text-muted">{snapshots.length} 个快照</span>
        <button onClick={snap} disabled={busy}
          className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm hover:border-primary disabled:opacity-60">
          {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Camera className="h-3.5 w-3.5" />} 立即快照
        </button>
      </div>
      {error && <p className="mb-2 text-sm text-danger">{error}</p>}
      {snapshots.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted">暂无配置快照。点「立即快照」备份当前运行配置。</p>
      ) : (
        <ul className="divide-y divide-border/60">
          {snapshots.map((s) => (
            <li key={s.id} className="flex items-center justify-between py-2 text-sm">
              <span className="font-mono text-xs text-muted">{new Date(s.taken_at).toLocaleString()}</span>
              <span className="text-muted">{s.source} · {s.statement_count} 条配置</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
