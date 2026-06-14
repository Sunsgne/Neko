"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  LayoutDashboard,
  Router,
  Building2,
  Network,
  Globe,
  Activity,
  Bell,
  Search,
  Settings,
  ChevronsLeft,
  LogOut,
  Server,
  Rocket,
  Workflow,
  Radar,
  ScrollText,
} from "lucide-react";
import { cn } from "@/lib/cn";
import { logout, API_BASE_URL } from "@/lib/api";
import { clearSession, currentToken, getCookie, EMAIL_COOKIE, ROLE_COOKIE } from "@/lib/session";

const nav = [
  { href: "/", label: "仪表盘", icon: LayoutDashboard },
  { href: "/devices", label: "设备", icon: Router },
  { href: "/discover", label: "发现纳管", icon: Radar },
  { href: "/backbone", label: "骨干节点", icon: Server },
  { href: "/orchestrate", label: "编排下发", icon: Workflow },
  { href: "/accel", label: "加速", icon: Rocket },
  { href: "/tenants", label: "租户", icon: Building2 },
  { href: "/topology", label: "拓扑", icon: Network },
  { href: "/dns", label: "DNS", icon: Globe },
  { href: "/links", label: "链路质量", icon: Activity },
  { href: "/alerts", label: "告警", icon: Bell },
  { href: "/audit", label: "审计", icon: ScrollText },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [collapsed, setCollapsed] = React.useState(false);
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState("");

  const [firing, setFiring] = React.useState(0);

  React.useEffect(() => {
    setEmail(getCookie(EMAIL_COOKIE) ?? "");
    setRole(getCookie(ROLE_COOKIE) ?? "");
  }, [pathname]);

  // Live updates via SSE: refresh the current page when the firing-alert count
  // changes (new/cleared alerts, device on/offline reflected in summary).
  React.useEffect(() => {
    if (pathname === "/login") return;
    const token = getCookie("neko_token");
    if (!token) return;
    const es = new EventSource(`${API_BASE_URL}/api/v1/events?token=${encodeURIComponent(token)}`);
    let prev = -1;
    es.addEventListener("summary", (e) => {
      try {
        const s = JSON.parse((e as MessageEvent).data);
        setFiring(s.firing_alerts ?? 0);
        if (prev >= 0 && s.firing_alerts !== prev) router.refresh();
        prev = s.firing_alerts ?? 0;
      } catch {
        /* ignore */
      }
    });
    es.onerror = () => { /* EventSource auto-reconnects */ };
    return () => es.close();
  }, [pathname, router]);

  // The login route renders without the app chrome.
  if (pathname === "/login") {
    return <>{children}</>;
  }

  async function onLogout() {
    const t = currentToken();
    if (t) await logout(t);
    clearSession();
    router.replace("/login");
    router.refresh();
  }

  const initials = email ? email.slice(0, 2).toUpperCase() : "··";

  return (
    <div className="flex min-h-screen">
      <aside
        className={cn(
          "sticky top-0 flex h-screen flex-col border-r border-border bg-surface/60 backdrop-blur transition-all",
          collapsed ? "w-16" : "w-60"
        )}
      >
        <div className="flex h-16 items-center gap-2 px-4">
          <div className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-primary/20 text-primary shadow-glow">
            <Network className="h-4 w-4" />
          </div>
          {!collapsed && (
            <div className="leading-tight">
              <div className="text-sm font-semibold tracking-tight">Neko SD-WAN</div>
              <div className="text-[10px] uppercase tracking-widest text-muted">Control Plane</div>
            </div>
          )}
        </div>

        <nav className="flex-1 space-y-1 px-2 py-2">
          {nav.map((item) => {
            const active = pathname === item.href;
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors",
                  active
                    ? "bg-primary/15 text-primary"
                    : "text-muted hover:bg-elevated hover:text-foreground"
                )}
              >
                <Icon className="h-4 w-4 shrink-0" />
                {!collapsed && <span>{item.label}</span>}
              </Link>
            );
          })}
        </nav>

        <button
          onClick={() => setCollapsed((c) => !c)}
          className="m-2 flex items-center justify-center rounded-lg px-3 py-2 text-muted hover:bg-elevated hover:text-foreground"
        >
          <ChevronsLeft className={cn("h-4 w-4 transition-transform", collapsed && "rotate-180")} />
        </button>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-10 flex h-16 items-center gap-4 border-b border-border bg-background/80 px-6 backdrop-blur">
          <div className="flex flex-1 items-center gap-2 rounded-lg border border-border bg-surface px-3 py-1.5 text-sm text-muted">
            <Search className="h-4 w-4" />
            <span>搜索设备、租户、链路…</span>
            <kbd className="ml-auto rounded border border-border px-1.5 py-0.5 text-[10px] text-muted">⌘K</kbd>
          </div>
          <Link href="/alerts" className="relative rounded-lg p-2 text-muted hover:bg-surface hover:text-foreground">
            <Bell className="h-4 w-4" />
            {firing > 0 && (
              <span className="absolute -right-0.5 -top-0.5 grid h-4 min-w-4 place-items-center rounded-full bg-danger px-1 text-[10px] font-semibold text-white">
                {firing}
              </span>
            )}
          </Link>
          <button className="rounded-lg p-2 text-muted hover:bg-surface hover:text-foreground">
            <Settings className="h-4 w-4" />
          </button>
          <div className="flex items-center gap-2 border-l border-border pl-3">
            <div className="hidden text-right sm:block">
              <div className="text-xs font-medium leading-tight">{email || "未登录"}</div>
              <div className="text-[10px] uppercase tracking-wide text-muted">{role || ""}</div>
            </div>
            <div className="grid h-8 w-8 place-items-center rounded-full bg-primary/20 text-xs font-semibold text-primary">
              {initials}
            </div>
            <button
              onClick={onLogout}
              title="退出登录"
              className="rounded-lg p-2 text-muted hover:bg-surface hover:text-danger"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </header>

        <main className="flex-1 px-6 py-6">{children}</main>
      </div>
    </div>
  );
}
