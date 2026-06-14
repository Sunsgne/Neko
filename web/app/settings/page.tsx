"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { KeyRound, Loader2, LogOut, Info, User } from "lucide-react";
import { Card, CardHeader, Badge } from "@/components/ui";
import { getSystem, changePassword, logout, type SystemInfo, ApiError } from "@/lib/api";
import { currentToken, getCookie, clearSession, EMAIL_COOKIE, ROLE_COOKIE } from "@/lib/session";

export default function SettingsPage() {
  const router = useRouter();
  const [sys, setSys] = React.useState<SystemInfo | null>(null);
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState("");

  const [oldPw, setOldPw] = React.useState("");
  const [newPw, setNewPw] = React.useState("");
  const [confirmPw, setConfirmPw] = React.useState("");
  const [busy, setBusy] = React.useState(false);
  const [msg, setMsg] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    setEmail(getCookie(EMAIL_COOKIE) ?? "");
    setRole(getCookie(ROLE_COOKIE) ?? "");
    getSystem(currentToken()).then(setSys).catch(() => setSys(null));
  }, []);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setMsg(null);
    if (newPw !== confirmPw) return setError("两次输入的新密码不一致");
    if (newPw.length < 8) return setError("新密码至少 8 位");
    setBusy(true);
    try {
      await changePassword(oldPw, newPw, currentToken());
      setMsg("密码已修改");
      setOldPw(""); setNewPw(""); setConfirmPw("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "修改失败");
    } finally {
      setBusy(false);
    }
  }

  async function doLogout() {
    const t = currentToken();
    if (t) await logout(t);
    clearSession();
    router.replace("/login");
    router.refresh();
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">设置</h1>
        <p className="mt-1 text-sm text-muted">账号、安全与系统信息</p>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader title="账号" subtitle="当前登录身份" />
          <dl className="space-y-2 text-sm">
            <Row k={<span className="inline-flex items-center gap-1.5"><User className="h-3.5 w-3.5" /> 邮箱</span>} v={email || "—"} />
            <Row k="角色" v={<Badge tone={role === "operator" ? "primary" : "neutral"}>{role === "operator" ? "平台运营" : role === "tenant" ? "租户" : role || "—"}</Badge>} />
          </dl>
          <button onClick={doLogout} className="mt-4 flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted hover:border-danger/40 hover:text-danger">
            <LogOut className="h-4 w-4" /> 退出登录
          </button>
        </Card>

        <Card>
          <CardHeader title="系统信息" subtitle="平台运行状态" />
          {sys ? (
            <dl className="space-y-2 text-sm">
              <Row k={<span className="inline-flex items-center gap-1.5"><Info className="h-3.5 w-3.5" /> 版本</span>} v={<span className="font-mono text-xs">{sys.version}</span>} />
              <Row k="存储后端" v={<Badge tone={sys.store === "postgres" ? "success" : "warning"}>{sys.store}</Badge>} />
              <Row k="鉴权" v={sys.auth_enabled ? <Badge tone="success">已启用</Badge> : <Badge tone="warning">未启用</Badge>} />
            </dl>
          ) : (
            <p className="text-sm text-muted">加载中…</p>
          )}
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader title="修改密码" subtitle="建议定期更换强密码" />
          <form onSubmit={submit} className="grid max-w-md grid-cols-1 gap-3">
            <Field label="原密码" value={oldPw} onChange={setOldPw} />
            <Field label="新密码（至少 8 位）" value={newPw} onChange={setNewPw} />
            <Field label="确认新密码" value={confirmPw} onChange={setConfirmPw} />
            {error && <p className="text-sm text-danger">{error}</p>}
            {msg && <p className="text-sm text-success">{msg}</p>}
            <button type="submit" disabled={busy} className="flex w-fit items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background hover:opacity-90 disabled:opacity-60">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />} 修改密码
            </button>
          </form>
        </Card>
      </div>
    </div>
  );
}

function Row({ k, v }: { k: React.ReactNode; v: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="text-muted">{k}</dt>
      <dd className="text-right">{v}</dd>
    </div>
  );
}

function Field({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">{label}</label>
      <input type="password" value={value} onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary" required />
    </div>
  );
}
