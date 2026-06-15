"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Camera, Loader2, Search, Eye, Pencil, RotateCcw, X } from "lucide-react";
import {
  takeSnapshot, getSnapshot, restoreSnapshot, applySnapshotConfig, ApiError,
  type ConfigSnapshot, type ConfigSnapshotDetail, type ConfigStatement,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

type ModalMode = "view" | "edit" | null;

export function ConfigBackup({ deviceId, snapshots }: { deviceId: string; snapshots: ConfigSnapshot[] }) {
  const router = useRouter();
  const [busy, setBusy] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [modal, setModal] = React.useState<ModalMode>(null);
  const [activeId, setActiveId] = React.useState<string | null>(null);
  const [detail, setDetail] = React.useState<ConfigSnapshotDetail | null>(null);
  const [loadingDetail, setLoadingDetail] = React.useState(false);
  const [filter, setFilter] = React.useState("");
  const [editJson, setEditJson] = React.useState("");
  const [editError, setEditError] = React.useState<string | null>(null);
  const [applyResult, setApplyResult] = React.useState<string | null>(null);

  async function snap() {
    setBusy("snap");
    setError(null);
    try {
      await takeSnapshot(deviceId, currentToken());
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "快照失败");
    } finally {
      setBusy(null);
    }
  }

  function closeModal() {
    setModal(null);
    setActiveId(null);
    setDetail(null);
    setFilter("");
    setEditJson("");
    setEditError(null);
    setApplyResult(null);
  }

  async function loadDetail(id: string, mode: ModalMode) {
    setActiveId(id);
    setModal(mode);
    setDetail(null);
    setFilter("");
    setEditJson("");
    setEditError(null);
    setApplyResult(null);
    setLoadingDetail(true);
    setError(null);
    try {
      const d = await getSnapshot(deviceId, id, currentToken());
      setDetail(d);
      if (mode === "edit") {
        setEditJson(JSON.stringify(d.state?.statements ?? [], null, 2));
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "无法读取快照内容");
      closeModal();
    } finally {
      setLoadingDetail(false);
    }
  }

  async function doRestore(id: string) {
    const snap = snapshots.find((s) => s.id === id);
    const label = snap ? new Date(snap.taken_at).toLocaleString() : id;
    if (!window.confirm(`确认将设备配置还原到快照「${label}」？\n\n此操作会覆盖设备当前运行配置。`)) return;
    setBusy(id);
    setError(null);
    try {
      const res = await restoreSnapshot(deviceId, id, currentToken());
      if (res.error || res.result?.status === "rolledback" || res.result?.status === "blocked") {
        setError(res.error || res.result?.reason || `还原失败（${res.result?.status}）`);
      } else {
        router.refresh();
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "还原失败");
    } finally {
      setBusy(null);
    }
  }

  async function doApply() {
    if (!activeId || !detail) return;
    setEditError(null);
    setApplyResult(null);
    let statements: ConfigStatement[];
    try {
      statements = JSON.parse(editJson);
      if (!Array.isArray(statements)) throw new Error("必须是 JSON 数组");
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "JSON 格式无效");
      return;
    }
    if (!window.confirm(`确认将修改后的 ${statements.length} 条配置下发到设备？`)) return;
    setBusy(activeId);
    try {
      const res = await applySnapshotConfig(deviceId, activeId, { statements }, currentToken());
      const changes = res.plan?.changes?.length ?? 0;
      if (res.error || res.result?.status === "rolledback" || res.result?.status === "blocked") {
        setEditError(res.error || res.result?.reason || `下发失败（${res.result?.status}）`);
      } else {
        setApplyResult(`已下发 · ${res.result?.status} · ${changes} 项变更`);
        router.refresh();
      }
    } catch (err) {
      setEditError(err instanceof ApiError ? err.message : "下发失败");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <div className="mb-3 flex items-center justify-between px-1">
        <span className="text-xs text-muted">{snapshots.length} 个快照</span>
        <button onClick={snap} disabled={busy === "snap"}
          className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm hover:border-primary disabled:opacity-60">
          {busy === "snap" ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Camera className="h-3.5 w-3.5" />} 立即快照
        </button>
      </div>
      {error && <p className="mb-2 text-sm text-danger">{error}</p>}
      {snapshots.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted">暂无配置快照。点「立即快照」备份当前运行配置。</p>
      ) : (
        <div className="data-table-wrap rounded-lg border border-border/60">
          <table className="data-table">
            <thead>
              <tr>
                <th className="w-[200px]">时间</th>
                <th className="w-[100px]">来源</th>
                <th className="w-[100px]">条目</th>
                <th className="text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {snapshots.map((s) => (
                <tr key={s.id}>
                  <td className="font-mono text-xs">{new Date(s.taken_at).toLocaleString()}</td>
                  <td className="text-xs text-muted">{s.source}</td>
                  <td className="font-mono text-xs text-muted">{s.statement_count}</td>
                  <td>
                    <div className="flex items-center justify-end gap-1.5">
                      <button onClick={() => loadDetail(s.id, "view")} disabled={!!busy}
                        className="btn-sm" title="查看配置">
                        <Eye className="h-3 w-3" /> 查看
                      </button>
                      <button onClick={() => loadDetail(s.id, "edit")} disabled={!!busy}
                        className="btn-sm" title="编辑并下发">
                        <Pencil className="h-3 w-3" /> 更改
                      </button>
                      <button onClick={() => doRestore(s.id)} disabled={!!busy}
                        className="btn-sm-danger" title="还原到此快照">
                        {busy === s.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <RotateCcw className="h-3 w-3" />}
                        还原
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {modal && (
        <div className="modal-backdrop" onClick={closeModal}>
          <div className="modal-panel" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div>
                <h3 className="text-sm font-semibold">
                  {modal === "view" ? "查看快照" : "更改配置"}
                </h3>
                {detail && (
                  <p className="mt-0.5 font-mono text-xs text-muted">
                    {new Date(detail.taken_at).toLocaleString()} · {detail.statement_count} 条
                  </p>
                )}
              </div>
              <button onClick={closeModal} className="rounded-md p-1.5 text-muted hover:bg-elevated">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="flex-1 overflow-auto p-5">
              {loadingDetail && (
                <div className="flex items-center gap-2 py-8 text-sm text-muted">
                  <Loader2 className="h-4 w-4 animate-spin" /> 读取配置内容…
                </div>
              )}
              {!loadingDetail && detail && modal === "view" && (
                <SnapshotViewer detail={detail} filter={filter} setFilter={setFilter} />
              )}
              {!loadingDetail && detail && modal === "edit" && (
                <div className="space-y-3">
                  <p className="text-xs text-muted">
                    编辑下方 JSON（配置项数组），保存后将 diff 并下发到设备。每条需包含 path、key、attributes 字段。
                  </p>
                  <textarea
                    value={editJson}
                    onChange={(e) => setEditJson(e.target.value)}
                    spellCheck={false}
                    className="h-[min(420px,50vh)] w-full resize-y rounded-lg border border-border bg-elevated/50 p-3 font-mono text-xs leading-relaxed outline-none focus:border-primary"
                  />
                  {editError && <p className="text-sm text-danger">{editError}</p>}
                  {applyResult && <p className="text-sm text-success">{applyResult}</p>}
                </div>
              )}
            </div>

            {modal === "edit" && !loadingDetail && detail && (
              <div className="flex justify-end gap-2 border-t border-border px-5 py-3">
                <button onClick={closeModal} className="btn-sm">取消</button>
                <button onClick={doApply} disabled={!!busy} className="btn-sm-primary">
                  {busy ? <Loader2 className="h-3 w-3 animate-spin" /> : null} 下发到设备
                </button>
              </div>
            )}
          </div>
        </div>
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
    <div>
      <div className="mb-3 flex items-center gap-2">
        <Search className="h-3.5 w-3.5 shrink-0 text-muted" />
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="过滤：路径 / 键 / 属性，如 /ip/route 或 gateway"
          className="w-full rounded-md border border-border bg-elevated px-2 py-1.5 text-xs outline-none focus:border-primary"
        />
        <span className="shrink-0 text-xs text-muted">{shown.length}/{statements.length}</span>
      </div>
      <div className="max-h-[min(480px,55vh)] space-y-3 overflow-auto rounded-lg border border-border/60 bg-elevated/30 p-3">
        {[...groups.entries()].map(([path, items]) => (
          <div key={path}>
            <div className="sticky top-0 bg-elevated/95 py-0.5 font-mono text-xs font-semibold text-primary">{path}</div>
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
