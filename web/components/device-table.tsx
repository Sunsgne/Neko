"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Trash2, Loader2, ChevronRight } from "lucide-react";
import { Badge, StatusDot } from "@/components/ui";
import { DeviceActions } from "@/components/device-actions";
import { deleteDevice, ApiError, type Device, type DevicePlatform, type TrustState } from "@/lib/api";
import { currentToken } from "@/lib/session";

const platformTone: Record<DevicePlatform, "primary" | "success" | "warning" | "neutral"> = {
  routerboard: "primary",
  chr: "success",
  x86: "warning",
  unknown: "neutral",
};

const trustLabel: Record<TrustState, string> = {
  untrusted: "未信任",
  discovered: "已发现",
  authenticated: "已认证",
  enrolled: "已登记",
  managed: "已托管",
};

const trustTone: Record<TrustState, "neutral" | "primary" | "warning" | "success"> = {
  untrusted: "neutral",
  discovered: "warning",
  authenticated: "primary",
  enrolled: "primary",
  managed: "success",
};

export function DeviceTable({ devices }: { devices: Device[] }) {
  const router = useRouter();
  const [sel, setSel] = React.useState<Record<string, boolean>>({});
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const selectedIds = Object.keys(sel).filter((id) => sel[id]);
  const allChecked = devices.length > 0 && selectedIds.length === devices.length;

  function toggleAll() {
    if (allChecked) setSel({});
    else setSel(Object.fromEntries(devices.map((d) => [d.id, true])));
  }

  async function removeOne(id: string, name: string) {
    if (!window.confirm(`确认删除设备「${name}」？此操作不可撤销。`)) return;
    setBusy(true);
    setError(null);
    try {
      await deleteDevice(id, currentToken());
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "删除失败");
    } finally {
      setBusy(false);
    }
  }

  async function removeSelected() {
    if (selectedIds.length === 0) return;
    if (!window.confirm(`确认删除选中的 ${selectedIds.length} 台设备？此操作不可撤销。`)) return;
    setBusy(true);
    setError(null);
    try {
      for (const id of selectedIds) {
        await deleteDevice(id, currentToken()).catch(() => {});
      }
      setSel({});
      router.refresh();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <div className="flex h-12 items-center justify-between border-b border-border px-5">
        <div className="text-xs text-muted">
          {selectedIds.length > 0 ? `已选 ${selectedIds.length} 台` : `共 ${devices.length} 台设备`}
        </div>
        {selectedIds.length > 0 && (
          <button
            onClick={removeSelected}
            disabled={busy}
            className="flex items-center gap-1.5 rounded-lg border border-danger/40 bg-danger/10 px-3 py-1.5 text-sm text-danger hover:bg-danger/20 disabled:opacity-60"
          >
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />} 删除选中
          </button>
        )}
      </div>
      {error && <p className="px-5 py-2 text-sm text-danger">{error}</p>}
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left text-xs uppercase tracking-wide text-muted">
              <th className="w-10 px-5 py-3">
                <input type="checkbox" checked={allChecked} onChange={toggleAll} aria-label="全选" />
              </th>
              <th className="px-3 py-3 font-medium">名称</th>
              <th className="px-3 py-3 font-medium">在线</th>
              <th className="px-3 py-3 font-medium">管理地址</th>
              <th className="px-3 py-3 font-medium">平台 / 型号</th>
              <th className="px-3 py-3 font-medium">版本</th>
              <th className="px-3 py-3 font-medium">CPU / 内存</th>
              <th className="px-3 py-3 font-medium">状态</th>
              <th className="px-3 py-3 text-right font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {devices.map((d) => {
              const st = d.status;
              const memPct =
                st && st.total_memory_bytes > 0
                  ? Math.round((1 - st.free_memory_bytes / st.total_memory_bytes) * 100)
                  : null;
              return (
                <tr key={d.id} className="border-b border-border/50 transition-colors hover:bg-elevated/40">
                  <td className="px-5 py-3">
                    <input
                      type="checkbox"
                      checked={!!sel[d.id]}
                      onChange={(e) => setSel((s) => ({ ...s, [d.id]: e.target.checked }))}
                      aria-label={`选择 ${d.name}`}
                    />
                  </td>
                  <td className="px-3 py-3">
                    <Link href={`/devices/${d.id}`} className="group inline-flex items-center gap-1 font-medium hover:text-primary">
                      <span className="max-w-[200px] truncate">{d.name}</span>
                      <ChevronRight className="h-3.5 w-3.5 opacity-0 transition-opacity group-hover:opacity-100" />
                    </Link>
                  </td>
                  <td className="px-3 py-3">
                    {d.enrolled ? (
                      <span className="inline-flex items-center gap-1.5 text-xs">
                        <StatusDot tone={st?.online ? "success" : "danger"} />
                        {st?.online ? "在线" : "离线"}
                      </span>
                    ) : (
                      <span className="text-xs text-muted">—</span>
                    )}
                  </td>
                  <td className="px-3 py-3 font-mono text-xs text-muted">{d.mgmt_address}</td>
                  <td className="px-3 py-3">
                    <span className="inline-flex items-center gap-2">
                      <Badge tone={platformTone[d.platform]}>{d.platform}</Badge>
                      <span className="max-w-[180px] truncate text-muted">{d.model || "—"}</span>
                    </span>
                  </td>
                  <td className="px-3 py-3 font-mono text-xs text-muted">{st?.version || "—"}</td>
                  <td className="px-3 py-3 font-mono text-xs text-muted">
                    {st?.online ? `${st.cpu_load_percent}% / ${memPct ?? "—"}%` : "—"}
                  </td>
                  <td className="px-3 py-3">
                    <span className="inline-flex items-center gap-2 text-xs">
                      <StatusDot tone={trustTone[d.trust_state]} />
                      {trustLabel[d.trust_state]}
                    </span>
                  </td>
                  <td className="px-3 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <DeviceActions deviceId={d.id} enrolled={d.enrolled} compact />
                      <button
                        onClick={() => removeOne(d.id, d.name)}
                        disabled={busy}
                        title="删除设备"
                        className="rounded-md p-1.5 text-muted hover:bg-danger/10 hover:text-danger disabled:opacity-60"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
