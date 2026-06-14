"use client";

import * as React from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Network, LogIn, Loader2 } from "lucide-react";
import { login, ApiError } from "@/lib/api";
import { saveSession } from "@/lib/session";

export default function LoginPage() {
  return (
    <React.Suspense fallback={null}>
      <LoginForm />
    </React.Suspense>
  );
}

function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const [email, setEmail] = React.useState("admin@neko.io");
  const [password, setPassword] = React.useState("neko12345");
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const { token, user } = await login(email, password);
      saveSession(token, user.email, user.is_operator);
      const next = params.get("next") || "/";
      router.replace(next);
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "登录失败，请稍后重试");
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3 text-center">
          <div className="grid h-12 w-12 place-items-center rounded-xl bg-primary/20 text-primary shadow-glow">
            <Network className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-xl font-semibold tracking-tight">Neko SD-WAN</h1>
            <p className="text-xs uppercase tracking-widest text-muted">Control Plane</p>
          </div>
        </div>

        <form onSubmit={onSubmit} className="card space-y-4">
          <div>
            <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">邮箱</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="username"
              className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary"
              required
            />
          </div>
          <div>
            <label className="mb-1.5 block text-xs uppercase tracking-wide text-muted">密码</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              className="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm outline-none focus:border-primary"
              required
            />
          </div>

          {error && (
            <div className="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-background transition-opacity hover:opacity-90 disabled:opacity-60"
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <LogIn className="h-4 w-4" />}
            登录
          </button>

          <div className="rounded-lg border border-border bg-elevated/50 p-3 text-xs text-muted">
            <p className="mb-1 font-medium text-foreground/80">演示账号</p>
            <p>运营端：<span className="font-mono">admin@neko.io / neko12345</span></p>
            <p>租户端：<span className="font-mono">ops@acme-corp.com / acme12345</span></p>
          </div>
        </form>
      </div>
    </div>
  );
}
