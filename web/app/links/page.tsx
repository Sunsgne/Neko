import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import { listLinks, type Link } from "@/lib/api";

export const dynamic = "force-dynamic";

const demo: Link[] = [
  { id: "l1", tenant_id: "", name: "上海-电信", kind: "wan", isp: "telecom", role: "primary", status: "up", latency_ms: 8, jitter_ms: 2, loss: 0, score: 98 },
  { id: "l2", tenant_id: "", name: "上海-联通", kind: "wan", isp: "unicom", role: "backup", status: "up", latency_ms: 14, jitter_ms: 4, loss: 0.002, score: 92 },
  { id: "l3", tenant_id: "", name: "北京-移动", kind: "wan", isp: "mobile", role: "primary", status: "degraded", latency_ms: 46, jitter_ms: 18, loss: 0.012, score: 71 },
  { id: "l4", tenant_id: "", name: "广州-电信", kind: "wan", isp: "telecom", role: "primary", status: "down", latency_ms: 220, jitter_ms: 80, loss: 0.045, score: 38 },
];

type Tone = "success" | "primary" | "warning" | "danger";

function scoreTone(score: number): Tone {
  if (score >= 90) return "success";
  if (score >= 75) return "primary";
  if (score >= 60) return "warning";
  return "danger";
}

const scoreText: Record<Tone, string> = {
  success: "text-success",
  primary: "text-primary",
  warning: "text-warning",
  danger: "text-danger",
};

const scoreBar: Record<Tone, string> = {
  success: "bg-success",
  primary: "bg-primary",
  warning: "bg-warning",
  danger: "bg-danger",
};

function statusTone(status: string): "success" | "warning" | "danger" | "neutral" {
  if (status === "up") return "success";
  if (status === "degraded") return "warning";
  if (status === "down") return "danger";
  return "neutral";
}

export default async function LinksPage() {
  let links: Link[] = [];
  try {
    links = await listLinks();
  } catch {
    links = [];
  }
  if (links.length === 0) links = demo;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">链路质量</h1>
        <p className="mt-1 text-sm text-muted">
          延迟 / 丢包 / 抖动监控 · 评分 · 本地与全局切换 · 防震荡
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {links.map((l) => (
          <Card key={l.id}>
            <CardHeader
              title={l.name}
              subtitle={`${l.kind} · ${l.isp || "overlay"} · ${l.role}`}
              action={<Badge tone={statusTone(l.status)}><StatusDot tone={statusTone(l.status)} />{l.status}</Badge>}
            />
            <div className="mb-3 flex items-end justify-between">
              <span className="text-xs uppercase tracking-wide text-muted">评分</span>
              <span className={`font-mono text-2xl font-semibold ${scoreText[scoreTone(l.score)]}`}>{l.score.toFixed(0)}</span>
            </div>
            <div className="mb-4 h-2 overflow-hidden rounded-full bg-border/60">
              <div className={`h-full rounded-full ${scoreBar[scoreTone(l.score)]}`} style={{ width: `${l.score}%` }} />
            </div>
            <div className="grid grid-cols-3 gap-2 text-center">
              <Metric label="延迟" value={`${l.latency_ms} ms`} />
              <Metric label="抖动" value={`${l.jitter_ms} ms`} />
              <Metric label="丢包" value={`${(l.loss * 100).toFixed(2)}%`} />
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-elevated/50 py-2">
      <div className="font-mono text-sm font-medium">{value}</div>
      <div className="text-[10px] uppercase tracking-wide text-muted">{label}</div>
    </div>
  );
}
