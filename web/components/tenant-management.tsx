"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  Plus, Loader2, Pencil, Trash2, PauseCircle, PlayCircle,
} from "lucide-react";
import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import {
  listTenants, createTenant, updateTenant, deleteTenant,
  type Tenant, ApiError,
} from "@/lib/api";
import { currentToken } from "@/lib/session";

export function TenantsBoard() {
  const router = useRouter();
  const [tenants, setTenants] = React.useState<Tenant[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [modal, setModal] = React.useState<"create" | "edit" | "delete" | null>(null);
  const [active, setActive] = React.useState<Tenant | null>(null);
  const [busy, setBusy] = React.useState(false);

  const [name, setName] = React.useState("");
  const [slug, setSlug] = React.useState("");
  const [confirmSlug, setConfirmSlug] = React.useState("");

  async function reload() {
    setLoading(true);
    setError(null);
    try {
      setTenants(await listTenants(currentToken(), true));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "加载租户失败");
      setTenants([]);
    } finally {
      setLoading(false);
    }
  }

  React.useEffect(() => { reload(); }, []);

  function openCreate() {
    setActive(null);
    setName("");
    setSlug("");
    setModal("create");
  }

  function openEdit(t: Tenant) {
    setActive(t);
    setName(t.name);
    setSlug(t.slug);
    setModal("edit");
  }

  function openDelete(t: Tenant) {
    setActive(t);
    setConfirmSlug("");
    setModal("delete");
  }

  async function submitCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await createTenant({ name: name.trim(), slug: slug.trim() || undefined }, currentToken());
      setModal(null);
      await reload();
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "创建失败");
    } finally { setBusy(false); }
  }

  async function submitEdit(e: React.FormEvent) {
    e.preventDefault();
    if (!active) return;
    setBusy(true);
    setError(null);
    try {
      await updateTenant(active.id, { name: name.trim(), slug: slug.trim() }, currentToken());
      setModal(null);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "更新失败");
    } finally { setBusy(false); }
  }

  async function submitDelete(e: React.FormEvent) {
    e.preventDefault();
    if (!active) return;
    setBusy(true);
    setError(null);
    try {
      await deleteTenant(active.id, confirmSlug.trim(), currentToken());
      setModal(null);
      await reload();
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "删除失败");
    } finally { setBusy(false); }
  }

  async function toggleStatus(t: Tenant) {
    setBusy(true);
    setError(null);
    try {
      await updateTenant(t.id, { status: t.status === "active" ? "suspended" : "active" }, currentToken());
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "状态更新失败");
    } finally { setBusy(false); }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">租户管理</h1>
          <p className="mt-1 text-sm text-muted">运营端 · 创建 / 编辑 / 暂停 / 删除租户，查看设备与告警统计</p>
        </div>
        <button onClick={openCreate}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90">
          <Plus className="h-4 w-4" /> 新建租户
        </button>
      </div>

      {error && !modal && <p className="text-sm text-danger">{error}</p>}

      {loading ? (
        <div className="flex items-center justify-center py-24 text-muted">
          <Loader2 className="h-6 w-6 animate-spin" />
        </div>
      ) : tenants.length === 0 ? (
        <Card className="py-16 text-center text-sm text-muted">
          暂无租户，点击「新建租户」创建第一个客户组织
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {tenants.map((t) => (
            <Card key={t.id} className="flex flex-col">
              <CardHeader
                title={t.name}
                subtitle={`/${t.slug} · ${t.id}`}
                action={
                  <Badge tone={t.status === "active" ? "success" : "warning"}>
                    <StatusDot tone={t.status === "active" ? "success" : "warning"} />
                    {t.status === "active" ? "运行中" : "已暂停"}
                  </Badge>
                }
              />
              <div className="grid grid-cols-3 gap-2 text-center">
                <Metric label="站点" value={String(t.stats?.site_count ?? 0)} />
                <Metric label="设备" value={String(t.stats?.device_count ?? 0)} />
                <Metric label="告警" value={String(t.stats?.firing_alerts ?? 0)} warn={(t.stats?.firing_alerts ?? 0) > 0} />
              </div>
              <div className="mt-3 flex flex-wrap gap-2 border-t border-border/60 pt-3">
                <ActionBtn icon={Pencil} label="编辑" onClick={() => openEdit(t)} />
                <ActionBtn
                  icon={t.status === "active" ? PauseCircle : PlayCircle}
                  label={t.status === "active" ? "暂停" : "恢复"}
                  onClick={() => toggleStatus(t)}
                  disabled={busy}
                />
                <ActionBtn icon={Trash2} label="删除" tone="danger" onClick={() => openDelete(t)} />
              </div>
              <p className="mt-2 text-[10px] text-muted">
                创建于 {new Date(t.created_at).toLocaleDateString()}
              </p>
            </Card>
          ))}
        </div>
      )}

      {modal === "create" && (
        <Modal title="新建租户" onClose={() => setModal(null)}>
          <form onSubmit={submitCreate} className="space-y-4">
            <Field label="名称" value={name} onChange={setName} placeholder="例如：Acme Corp" required />
            <Field label="Slug（可选）" value={slug} onChange={setSlug} placeholder="自动生成，如 acme-corp" mono />
            {error && <p className="text-sm text-danger">{error}</p>}
            <ModalActions onCancel={() => setModal(null)} busy={busy} submitLabel="创建" />
          </form>
        </Modal>
      )}

      {modal === "edit" && active && (
        <Modal title={`编辑 · ${active.name}`} onClose={() => setModal(null)}>
          <form onSubmit={submitEdit} className="space-y-4">
            <Field label="名称" value={name} onChange={setName} required />
            <Field label="Slug" value={slug} onChange={setSlug} mono required />
            {error && <p className="text-sm text-danger">{error}</p>}
            <ModalActions onCancel={() => setModal(null)} busy={busy} submitLabel="保存" />
          </form>
        </Modal>
      )}

      {modal === "delete" && active && (
        <Modal title={`删除租户 · ${active.name}`} onClose={() => setModal(null)}>
          <form onSubmit={submitDelete} className="space-y-4">
            <p className="text-sm text-muted">
              将永久删除租户及其下所有设备、配置、告警等数据（不可恢复）。
              请输入 slug <strong className="font-mono text-foreground">{active.slug}</strong> 确认。
            </p>
            <Field label="确认 Slug" value={confirmSlug} onChange={setConfirmSlug} mono placeholder={active.slug} required />
            {error && <p className="text-sm text-danger">{error}</p>}
            <ModalActions onCancel={() => setModal(null)} busy={busy} submitLabel="确认删除" danger />
          </form>
        </Modal>
      )}
    </div>
  );
}

function Metric({ label, value, warn }: { label: string; value: string; warn?: boolean }) {
  return (
    <div className={`rounded-lg border py-2 ${warn ? "border-warning/40 bg-warning/10" : "border-border bg-elevated/50"}`}>
      <div className={`text-lg font-semibold ${warn ? "text-warning" : ""}`}>{value}</div>
      <div className="text-[10px] uppercase tracking-wide text-muted">{label}</div>
    </div>
  );
}

function ActionBtn({ icon: Icon, label, onClick, disabled, tone }: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  tone?: "danger";
}) {
  return (
    <button type="button" onClick={onClick} disabled={disabled}
      className={`flex items-center gap-1 rounded-md border px-2 py-1 text-xs hover:border-primary disabled:opacity-50 ${
        tone === "danger" ? "border-danger/30 text-danger hover:border-danger" : "border-border text-muted hover:text-foreground"
      }`}>
      <Icon className="h-3.5 w-3.5" /> {label}
    </button>
  );
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={onClose}>
      <div className="card w-full max-w-md space-y-3" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-sm font-semibold">{title}</h3>
        {children}
      </div>
    </div>
  );
}

function Field({ label, value, onChange, placeholder, mono, required }: {
  label: string; value: string; onChange: (v: string) => void;
  placeholder?: string; mono?: boolean; required?: boolean;
}) {
  return (
    <div>
      <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">{label}</label>
      <input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} required={required}
        className={`w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary ${mono ? "font-mono" : ""}`} />
    </div>
  );
}

function ModalActions({ onCancel, busy, submitLabel, danger }: {
  onCancel: () => void; busy: boolean; submitLabel: string; danger?: boolean;
}) {
  return (
    <div className="flex justify-end gap-2">
      <button type="button" onClick={onCancel} className="rounded-lg border border-border px-3 py-2 text-sm text-muted hover:text-foreground">
        取消
      </button>
      <button type="submit" disabled={busy}
        className={`flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium disabled:opacity-60 ${
          danger ? "bg-danger text-white hover:opacity-90" : "bg-primary text-background hover:opacity-90"
        }`}>
        {busy && <Loader2 className="h-4 w-4 animate-spin" />} {submitLabel}
      </button>
    </div>
  );
}
