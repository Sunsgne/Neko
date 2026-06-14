import { Card, CardHeader, Badge, StatusDot } from "@/components/ui";
import { listAlerts, type Alert } from "@/lib/api";

export const dynamic = "force-dynamic";

const demo: Alert[] = [
  { id: "1", tenant_id: "", device_id: "pop-gz", severity: "critical", title: "广州-电信 链路中断", detail: "丢包 4.5%，已本地切换备线", state: "firing", fired_at: "2026-06-14T03:51:00Z" },
  { id: "2", tenant_id: "", device_id: "edge-bj", severity: "warning", title: "edge-bj-02 sfp1 利用率 > 85%", detail: "持续 6 分钟", state: "firing", fired_at: "2026-06-14T03:58:00Z" },
  { id: "3", tenant_id: "", device_id: "edge-sh", severity: "info", title: "BGP 策略 v7 已确认", detail: "commit-confirm 成功", state: "resolved", fired_at: "2026-06-14T03:33:00Z" },
];

const sevTone: Record<string, "danger" | "warning" | "primary"> = {
  critical: "danger",
  warning: "warning",
  info: "primary",
};

function fmt(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toISOString().slice(11, 16);
}

export default async function AlertsPage() {
  let alerts: Alert[] = [];
  try {
    alerts = await listAlerts();
  } catch {
    alerts = [];
  }
  if (alerts.length === 0) alerts = demo;

  const firing = alerts.filter((a) => a.state === "firing");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">告警中心</h1>
        <p className="mt-1 text-sm text-muted">
          阈值规则引擎 · 去重 / 抑制 / 升级 — {firing.length} 条活跃
        </p>
      </div>

      <Card>
        <CardHeader title="告警" subtitle="活跃优先" />
        <ul className="divide-y divide-border/60">
          {alerts.map((a) => (
            <li key={a.id} className="flex items-start gap-3 py-3">
              <StatusDot tone={a.state === "firing" ? sevTone[a.severity] : "neutral"} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{a.title}</span>
                  <Badge tone={sevTone[a.severity]}>{a.severity}</Badge>
                  {a.state === "resolved" && <Badge tone="neutral">resolved</Badge>}
                </div>
                <p className="mt-0.5 text-xs text-muted">{a.detail}</p>
              </div>
              <span className="font-mono text-xs text-muted">{fmt(a.fired_at)}</span>
            </li>
          ))}
        </ul>
      </Card>
    </div>
  );
}
