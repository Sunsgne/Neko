"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Camera, Loader2, ChevronRight, Search } from "lucide-react";
import {
  takeSnapshot, getSnapshot, ApiError,
  type ConfigSnapshot, type ConfigSnapshotDetail, type ConfigStatement,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

export function ConfigBackup({ deviceId, snapshots }: { deviceId: string; snapshots: ConfigSnapshot[] }) {
  const router = useRouter();
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [openId, setOpenId] = React.useState<string | null>(null);
  const [detail, setDetail] = React.useState<ConfigSnapshotDetail | null>(null);
  const [loadingDetail, setLoadingDetail] = React.useState(false);
  const [filter, setFilter] = React.useState("");

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

  async function view(id: string) {
    if (openId === id) {
      setOpenId(null);
      setDetail(null);
      return;
    }
    setOpenId(id);
    setDetail(null);
    setFilter("");
    setLoadingDetail(true);
    setError(null);
    try {
      setDetail(await getSnapshot(deviceId, id, currentToken()));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "无法读取快照内容");
      setOpenId(null);
    } finally {
      setLoadingDetail(false);
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
            <li key={s.id}>
              <button onClick={() => view(s.id)}
                className="flex w-full items-center justify-between py-2 text-left text-sm hover:text-primary">
                <span className="flex items-center gap-1.5">
                  <ChevronRight className={`h-3.5 w-3.5 text-muted transition-transform ${openId === s.id ? "rotate-90" : ""}`} />
                  <span className="font-mono text-xs text-muted">{new Date(s.taken_at).toLocaleString()}</span>
                </span>
                <span className="text-muted">{s.source} · {s.statement_count} 条配置</span>
              </button>
              {openId === s.id && (
                <div className="mb-2">
                  {loadingDetail && (
                    <div className="flex items-center gap-2 py-3 text-sm text-muted">
                      <Loader2 className="h-3.5 w-3.5 animate-spin" /> 读取配置内容…
                    </div>
                  )}
                  {detail && <SnapshotViewer detail={detail} filter={filter} setFilter={setFilter} />}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function SnapshotViewer({ detail, filter, setFilter }: { detail: ConfigSnapshotDetail; filter: string; setFilter: (s: string) => void }) {
  const statements = detail.state?.statements ?? [];
  const q = filter.trim().toLowerCase();
  const shown = q
    ? statements.filter((st) =>
        st.path.toLowerCase().includes(q) ||
        st.key.toLowerCase().includes(q) ||
        Object.entries(st.attributes).some(([k, v]) => `${k} ${v}`.toLowerCase().includes(q)),
      )
    : statements;

  // Group by RouterOS section path for readability.
  const groups = new Map<string, ConfigStatement[]>();
  for (const st of shown) {
    const arr = groups.get(st.path) ?? [];
    arr.push(st);
    groups.set(st.path, arr);
  }

  if (statements.length === 0) {
    return <p className="py-3 text-sm text-muted">该快照未捕获到任何配置项。</p>;
  }

  return (
    <div className="rounded-lg border border-border bg-elevated/40 p-3">
      <div className="mb-2 flex items-center gap-2">
        <Search className="h-3.5 w-3.5 text-muted" />
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="过滤：路径 / 键 / 属性，如 /ip/route 或 gateway"
          className="w-full rounded-md border border-border bg-surface px-2 py-1 text-xs outline-none focus:border-primary"
        />
        <span className="shrink-0 text-xs text-muted">{shown.length}/{statements.length}</span>
      </div>
      <div className="max-h-[420px] space-y-3 overflow-auto">
        {[...groups.entries()].map(([path, items]) => (
          <div key={path}>
            <div className="sticky top-0 bg-elevated/90 py-0.5 font-mono text-xs font-semibold text-primary">{path}</div>
            <ul className="space-y-1 pl-2">
              {items.map((st) => (
                <li key={st.path + st.key} className="rounded border border-border/50 bg-surface/50 p-1.5">
                  <div className="font-mono text-[11px] text-muted">{st.key}</div>
                  <div className="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 font-mono text-[11px] text-foreground/80">
                    {Object.entries(st.attributes).map(([k, v]) => (
                      <span key={k}><span className="text-muted">{k}</span>=<span>{v}</span></span>
                    ))}
                  </div>
                </li>
              ))}
            </ul>
          </div>
        ))}
        {shown.length === 0 && <p className="py-2 text-xs text-muted">无匹配项。</p>}
      </div>
    </div>
  );
}
