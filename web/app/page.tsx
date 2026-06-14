import { Router, Activity, AlertTriangle, Gauge, ArrowUpRight } from "lucide-react";
import { Card, CardHeader, Kpi, Badge, StatusDot } from "@/components/ui";
import { listDevices, listAlerts, listLinks, type Device, type Alert, type Link } from "@/lib/api";
import { serverToken, serverIdentity } from "@/lib/server-session";

export const dynamic = "force-dynamic";

async function safe<T>(p: Promise<T[]>): Promise<T[]> {
  try {
    return await p;
  } catch {
    return [];
  }
}

const recentEvents = [
  { time: "21:14", tone: "success" as const, text: "POP-上海 链路评分恢复至 98" },
  { time: "21:02", tone: "warning" as const, text: "edge-bj-02 接口 sfp1 利用率 > 85%" },
  { time: "20:51", tone: "primary" as const, text: "租户 acme-corp 下发 BGP 策略 v7 已确认" },
  { time: "20:33", tone: "danger" as const, text: "POP-广州 → 北京 链路丢包 4.2%，已本地切换备线" },
];

export default async function DashboardPage() {
  const token = serverToken();
  const { role } = serverIdentity();
  const [devices, alerts, links] = await Promise.all([
    safe<Device>(listDevices(token)),
    safe<Alert>(listAlerts(token)),
    safe<Link>(listLinks(token)),
  ]);

  const total = devices.length;
  const managed = devices.filter((d) => d.trust_state === "managed").length;
  const onlineRate = total ? Math.round((managed / total) * 100) : 99;
  const firing = alerts.filter((a) => a.state === "firing");
  const critical = firing.filter((a) => a.severity === "critical").length;
  const warnings = firing.filter((a) => a.severity === "warning").length;
  const avgScore = links.length
    ? Math.round(links.reduce((s, l) => s + l.score, 0) / links.length)
    : 0;
  const dist = {
    excellent: links.filter((l) => l.score >= 90).length,
    good: links.filter((l) => l.score >= 75 && l.score < 90).length,
    degraded: links.filter((l) => l.score >= 60 && l.score < 75).length,
    down: links.filter((l) => l.score < 60).length,
  };
  const pct = (n: number) => (links.length ? Math.round((n / links.length) * 100) : 0);
  const scope = role === "tenant" ? "租户总览" : "运营总览";

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{scope}</h1>
          <p className="mt-1 text-sm text-muted">网络健康、链路质量与告警实时视图</p>
        </div>
        <Badge tone={critical > 0 ? "danger" : firing.length > 0 ? "warning" : "success"}>
          <StatusDot tone={critical > 0 ? "danger" : firing.length > 0 ? "warning" : "success"} />
          {critical > 0 ? "存在严重告警" : firing.length > 0 ? "存在告警" : "系统正常"}
        </Badge>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Kpi label="纳管设备" value={String(total)} delta={`${managed} 已托管`} tone="primary" icon={<Router className="h-5 w-5" />} />
        <Kpi label="在线率" value={`${onlineRate}%`} delta="trust=managed 占比" tone="success" icon={<Gauge className="h-5 w-5" />} />
        <Kpi label="活跃告警" value={String(firing.length)} delta={`${critical} 严重 · ${warnings} 警告`} tone={firing.length > 0 ? "warning" : "success"} icon={<AlertTriangle className="h-5 w-5" />} />
        <Kpi label="链路均分" value={String(avgScore)} delta={`${links.length} 条链路`} tone="primary" icon={<Activity className="h-5 w-5" />} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader title="流量趋势" subtitle="过去 24 小时聚合 (Gbps)" action={<Badge tone="primary">实时</Badge>} />
          <Sparkline />
        </Card>

        <Card>
          <CardHeader title="链路质量分布" subtitle="按评分" />
          <div className="space-y-3">
            <ScoreBar label="优 (90-100)" value={pct(dist.excellent)} tone="success" />
            <ScoreBar label="良 (75-89)" value={pct(dist.good)} tone="primary" />
            <ScoreBar label="降级 (60-74)" value={pct(dist.degraded)} tone="warning" />
            <ScoreBar label="故障 (<60)" value={pct(dist.down)} tone="danger" />
          </div>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader title="最近事件" subtitle="设备、链路与配置变更" action={<a className="flex items-center gap-1 text-xs text-primary" href="#">全部 <ArrowUpRight className="h-3 w-3" /></a>} />
          <ul className="space-y-3">
            {recentEvents.map((e, i) => (
              <li key={i} className="flex items-center gap-3 text-sm">
                <StatusDot tone={e.tone} />
                <span className="font-mono text-xs text-muted">{e.time}</span>
                <span className="text-foreground/90">{e.text}</span>
              </li>
            ))}
          </ul>
        </Card>

        <Card>
          <CardHeader title="POP 健康" subtitle="双 POP / Route Reflector" />
          <div className="grid grid-cols-2 gap-3">
            <PopCard name="POP-上海" tone="success" latency="8 ms" sessions="iBGP ×24" />
            <PopCard name="POP-北京" tone="success" latency="11 ms" sessions="iBGP ×19" />
            <PopCard name="POP-广州" tone="warning" latency="23 ms" sessions="iBGP ×12" />
            <PopCard name="POP-深圳" tone="success" latency="9 ms" sessions="iBGP ×15" />
          </div>
        </Card>
      </div>
    </div>
  );
}

function Sparkline() {
  const pts = [22, 28, 25, 31, 35, 30, 38, 42, 39, 44, 41, 47, 43, 40, 45, 49, 46, 51, 48, 43, 41, 44, 42, 43];
  const max = Math.max(...pts);
  const w = 100;
  const h = 40;
  const d = pts
    .map((p, i) => `${(i / (pts.length - 1)) * w},${h - (p / max) * h}`)
    .join(" ");
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="h-40 w-full" preserveAspectRatio="none">
      <defs>
        <linearGradient id="g" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="hsl(199 89% 52%)" stopOpacity="0.35" />
          <stop offset="100%" stopColor="hsl(199 89% 52%)" stopOpacity="0" />
        </linearGradient>
      </defs>
      <polyline points={d} fill="none" stroke="hsl(199 89% 52%)" strokeWidth="0.8" vectorEffect="non-scaling-stroke" />
      <polygon points={`0,${h} ${d} ${w},${h}`} fill="url(#g)" />
    </svg>
  );
}

function ScoreBar({ label, value, tone }: { label: string; value: number; tone: "success" | "primary" | "warning" | "danger" }) {
  const bar = { success: "bg-success", primary: "bg-primary", warning: "bg-warning", danger: "bg-danger" }[tone];
  return (
    <div>
      <div className="mb-1 flex justify-between text-xs">
        <span className="text-muted">{label}</span>
        <span className="font-mono">{value}%</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-border/60">
        <div className={`h-full rounded-full ${bar}`} style={{ width: `${value}%` }} />
      </div>
    </div>
  );
}

function PopCard({ name, tone, latency, sessions }: { name: string; tone: "success" | "warning" | "danger"; latency: string; sessions: string }) {
  return (
    <div className="rounded-lg border border-border bg-elevated/60 p-3">
      <div className="flex items-center gap-2 text-sm font-medium">
        <StatusDot tone={tone} /> {name}
      </div>
      <div className="mt-2 flex justify-between text-xs text-muted">
        <span>{latency}</span>
        <span className="font-mono">{sessions}</span>
      </div>
    </div>
  );
}
