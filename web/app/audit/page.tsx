import { Card, CardHeader, Badge } from "@/components/ui";
import { listAudit, type AuditEntry } from "@/lib/api";
import { serverToken } from "@/lib/server-session";

export const dynamic = "force-dynamic";

const actionTone: Record<string, "success" | "primary" | "warning" | "danger" | "neutral"> = {
  create: "success",
  enroll: "primary",
  config_push: "warning",
  orchestrate: "warning",
  config_snapshot: "neutral",
  config_restore: "warning",
  config_apply: "warning",
  trust_change: "primary",
  batch_onboard: "success",
};

export default async function AuditPage() {
  let entries: AuditEntry[] = [];
  try {
    entries = await listAudit(serverToken());
  } catch {
    entries = [];
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">审计日志</h1>
        <p className="mt-1 text-sm text-muted">谁、何时、对哪个对象、做了什么（追加写，不可篡改）</p>
      </div>

      <Card className="p-0">
        <CardHeader title="操作记录" subtitle={`${entries.length} 条`} className="px-5 pt-5" />
        {entries.length === 0 ? (
          <p className="px-5 py-10 text-center text-sm text-muted">暂无审计记录。执行创建/纳管/下发等操作后会记录在此。</p>
        ) : (
          <div className="data-table-scroll border-t border-border">
            <table className="data-table">
              <thead>
                <tr>
                  <th>时间</th>
                  <th>操作者</th>
                  <th>动作</th>
                  <th>对象</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((e) => (
                  <tr key={e.id}>
                    <td className="whitespace-nowrap font-mono text-xs text-muted">{new Date(e.at).toLocaleString()}</td>
                    <td>{e.actor_id || "—"}</td>
                    <td><Badge tone={actionTone[e.action] ?? "neutral"}>{e.action}</Badge></td>
                    <td className="text-muted">{e.object_type}{e.object_id ? ` · ${e.object_id}` : ""}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
