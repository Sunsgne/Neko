"use client";

import * as React from "react";
import { Loader2, Plus, Trash2, Save, RefreshCw, SlidersHorizontal, X, Search } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import {
  listDevices, getConfigCatalog, restList, restCreate, restUpdate, restDelete,
  type Device, type ConfigSectionGroup, type ConfigSection, type RestItem, ApiError,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

// Keys RouterOS returns that aren't user-editable attributes.
const READONLY_KEYS = new Set([".id", ".nextid", "dynamic", "invalid", "default"]);

export default function ConfigPage() {
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [deviceId, setDeviceId] = React.useState("");
  const [catalog, setCatalog] = React.useState<ConfigSectionGroup[]>([]);
  const [section, setSection] = React.useState<ConfigSection | null>(null);
  const [items, setItems] = React.useState<RestItem[]>([]);
  const [filter, setFilter] = React.useState("");
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [editing, setEditing] = React.useState<RestItem | null>(null);
  const [creating, setCreating] = React.useState(false);

  React.useEffect(() => {
    getConfigCatalog(currentToken()).then(setCatalog).catch(() => setCatalog([]));
    listDevices(currentToken()).then((d) => {
      const managed = d.filter((x) => x.trust_state === "managed");
      setDevices(managed);
      if (managed[0]) setDeviceId(managed[0].id);
    }).catch(() => setDevices([]));
  }, []);

  const load = React.useCallback(async (sec: ConfigSection | null) => {
    if (!deviceId || !sec) return;
    setLoading(true);
    setError(null);
    setEditing(null);
    setCreating(false);
    try {
      const res = await restList(deviceId, sec.path, currentToken());
      setItems(res.items ?? []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "读取失败");
      setItems([]);
    } finally { setLoading(false); }
  }, [deviceId]);

  function pickSection(sec: ConfigSection) {
    setSection(sec);
    setFilter("");
    load(sec);
  }

  async function onDelete(item: RestItem) {
    if (!section || !window.confirm("删除该配置项？")) return;
    setError(null);
    try {
      await restDelete(deviceId, section.path, String(item[".id"]), currentToken());
      await load(section);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "删除失败");
    }
  }

  const columns = React.useMemo(() => {
    const set = new Set<string>();
    for (const it of items) for (const k of Object.keys(it)) if (k !== ".id") set.add(k);
    return [...set].slice(0, 6);
  }, [items]);

  const shown = React.useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return items;
    return items.filter((it) => JSON.stringify(it).toLowerCase().includes(q));
  }, [items, filter]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">远程配置</h1>
        <p className="mt-1 text-sm text-muted">无需登录设备，远程读写 RouterOS 全部菜单的配置（使用已托管设备的保存凭据）</p>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <label className="text-xs uppercase tracking-wide text-muted">目标设备</label>
        <select value={deviceId} onChange={(e) => { setDeviceId(e.target.value); setSection(null); setItems([]); }}
          className="rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary">
          {devices.length === 0 && <option value="">（无已托管设备）</option>}
          {devices.map((d) => <option key={d.id} value={d.id}>{d.name} · {d.mgmt_address}</option>)}
        </select>
        {section && (
          <button onClick={() => load(section)} className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm hover:border-primary">
            <RefreshCw className="h-3.5 w-3.5" /> 刷新
          </button>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[260px_1fr]">
        {/* Menu tree */}
        <Card className="max-h-[70vh] overflow-auto p-0">
          <CardHeader title="菜单" subtitle="选择配置段" />
          <div className="px-2 pb-3">
            {catalog.map((g) => (
              <div key={g.menu} className="mb-2">
                <div className="px-2 py-1 text-[11px] font-semibold uppercase tracking-wide text-muted">{g.menu}</div>
                {g.sections.map((sec) => (
                  <button key={sec.path} onClick={() => pickSection(sec)}
                    className={`flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm transition-colors ${section?.path === sec.path ? "bg-primary/15 text-primary" : "text-muted hover:bg-elevated hover:text-foreground"}`}>
                    <span className="truncate">{sec.label}</span>
                    {sec.singleton && <span className="ml-1 shrink-0 text-[10px] text-muted">单例</span>}
                  </button>
                ))}
              </div>
            ))}
          </div>
        </Card>

        {/* Items */}
        <Card className="p-0">
          <CardHeader
            title={section ? section.label : "请选择配置段"}
            subtitle={section ? section.path : "左侧菜单覆盖 RouterOS 全部功能"}
            action={section ? (
              section.singleton ? (
                <button onClick={() => { setCreating(false); setEditing(items[0] ?? {}); }}
                  className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-background hover:opacity-90">
                  <Save className="h-3.5 w-3.5" /> 编辑设置
                </button>
              ) : (
                <button onClick={() => { setCreating(true); setEditing({}); }}
                  className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-background hover:opacity-90">
                  <Plus className="h-3.5 w-3.5" /> 新增
                </button>
              )
            ) : undefined}
          />

          {error && <div className="mx-4 mb-3 rounded-lg border border-danger/40 bg-danger/10 p-2 text-sm text-danger">{error}</div>}

          {!section ? (
            <div className="flex flex-col items-center gap-2 py-20 text-center text-sm text-muted">
              <SlidersHorizontal className="h-6 w-6 text-primary" /> 从左侧选择一个配置段
            </div>
          ) : loading ? (
            <div className="flex items-center justify-center gap-2 py-20 text-sm text-muted"><Loader2 className="h-4 w-4 animate-spin" /> 读取中…</div>
          ) : (
            <div className="px-4 pb-4">
              {items.length > 1 && (
                <div className="mb-3 flex items-center gap-2">
                  <Search className="h-3.5 w-3.5 text-muted" />
                  <input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="过滤…"
                    className="w-full rounded-md border border-border bg-elevated px-2 py-1 text-xs outline-none focus:border-primary" />
                  <span className="shrink-0 text-xs text-muted">{shown.length}/{items.length}</span>
                </div>
              )}
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-y border-border text-left text-xs uppercase tracking-wide text-muted">
                      {columns.map((c) => <th key={c} className="px-3 py-2 font-medium">{c}</th>)}
                      <th className="px-3 py-2 text-right font-medium">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {shown.map((it, i) => (
                      <tr key={String(it[".id"] ?? i)} className="border-b border-border/50 hover:bg-elevated/40">
                        {columns.map((c) => (
                          <td key={c} className="max-w-[220px] truncate px-3 py-2 font-mono text-xs">{fmt(it[c])}</td>
                        ))}
                        <td className="px-3 py-2 text-right">
                          <button onClick={() => { setCreating(false); setEditing(it); }} title="编辑"
                            className="rounded-md p-1.5 text-muted hover:bg-primary/10 hover:text-primary"><Save className="h-4 w-4" /></button>
                          {it[".id"] !== undefined && (
                            <button onClick={() => onDelete(it)} title="删除"
                              className="rounded-md p-1.5 text-muted hover:bg-danger/10 hover:text-danger"><Trash2 className="h-4 w-4" /></button>
                          )}
                        </td>
                      </tr>
                    ))}
                    {items.length === 0 && (
                      <tr><td colSpan={columns.length + 1} className="px-3 py-10 text-center text-sm text-muted">该配置段暂无条目。{!section.singleton && "点「新增」添加。"}</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </Card>
      </div>

      {editing && section && (
        <ItemEditor
          section={section}
          item={editing}
          creating={creating}
          deviceId={deviceId}
          onClose={() => { setEditing(null); setCreating(false); }}
          onSaved={() => { setEditing(null); setCreating(false); load(section); }}
        />
      )}
    </div>
  );
}

function fmt(v: unknown): string {
  if (v === undefined || v === null) return "—";
  return String(v);
}

function ItemEditor({ section, item, creating, deviceId, onClose, onSaved }: {
  section: ConfigSection;
  item: RestItem;
  creating: boolean;
  deviceId: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const initialRows = React.useMemo(() => {
    const rows: Array<{ k: string; v: string }> = [];
    for (const [k, v] of Object.entries(item)) {
      if (READONLY_KEYS.has(k)) continue;
      rows.push({ k, v: String(v) });
    }
    if (rows.length === 0) rows.push({ k: "", v: "" });
    return rows;
  }, [item]);

  const [rows, setRows] = React.useState(initialRows);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const id = String(item[".id"] ?? "");

  function setRow(i: number, patch: Partial<{ k: string; v: string }>) {
    setRows((r) => r.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));
  }

  async function save() {
    setBusy(true);
    setError(null);
    const attributes: Record<string, string> = {};
    for (const { k, v } of rows) if (k.trim()) attributes[k.trim()] = v;
    try {
      if (section.singleton) {
        await restCreate(deviceId, { path: section.path, attributes, singleton: true }, currentToken());
      } else if (creating || !id) {
        await restCreate(deviceId, { path: section.path, attributes }, currentToken());
      } else {
        await restUpdate(deviceId, { path: section.path, item_id: id, attributes }, currentToken());
      }
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "保存失败");
    } finally { setBusy(false); }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div className="w-full max-w-lg rounded-xl border border-border bg-surface p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-3 flex items-center justify-between">
          <div>
            <div className="text-sm font-semibold">{creating ? "新增" : "编辑"} · {section.label}</div>
            <div className="font-mono text-xs text-muted">{section.path}{id && ` · ${id}`}</div>
          </div>
          <button onClick={onClose} className="rounded-md p-1.5 text-muted hover:bg-elevated"><X className="h-4 w-4" /></button>
        </div>

        <div className="max-h-[55vh] space-y-2 overflow-auto">
          {rows.map((row, i) => (
            <div key={i} className="flex items-center gap-2">
              <input value={row.k} onChange={(e) => setRow(i, { k: e.target.value })} placeholder="参数名 (如 address)"
                className="w-2/5 rounded-md border border-border bg-elevated px-2 py-1.5 font-mono text-xs outline-none focus:border-primary" />
              <span className="text-muted">=</span>
              <input value={row.v} onChange={(e) => setRow(i, { v: e.target.value })} placeholder="值"
                className="flex-1 rounded-md border border-border bg-elevated px-2 py-1.5 font-mono text-xs outline-none focus:border-primary" />
              <button onClick={() => setRows((r) => r.filter((_, idx) => idx !== i))}
                className="rounded-md p-1.5 text-muted hover:bg-danger/10 hover:text-danger"><Trash2 className="h-3.5 w-3.5" /></button>
            </div>
          ))}
          <button onClick={() => setRows((r) => [...r, { k: "", v: "" }])}
            className="flex items-center gap-1.5 rounded-md border border-dashed border-border px-2 py-1.5 text-xs text-muted hover:border-primary hover:text-primary">
            <Plus className="h-3.5 w-3.5" /> 添加参数
          </button>
        </div>

        {error && <p className="mt-2 text-sm text-danger">{error}</p>}

        <div className="mt-4 flex items-center justify-end gap-2">
          <Badge tone="warning">写入将立即下发到设备</Badge>
          <button onClick={onClose} className="rounded-lg border border-border px-3 py-1.5 text-sm hover:border-primary">取消</button>
          <button onClick={save} disabled={busy} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} 保存并下发
          </button>
        </div>
      </div>
    </div>
  );
}
