import { Card } from "@/components/ui";
import { listLinks, listDevices, type Link, type Device } from "@/lib/api";
import { serverToken } from "@/lib/server-session";

export const dynamic = "force-dynamic";

const demoLinks: Link[] = [
  { id: "l1", tenant_id: "", name: "上海-电信", kind: "wan", isp: "telecom", role: "primary", status: "up", latency_ms: 8, jitter_ms: 2, loss: 0, score: 98 },
  { id: "l3", tenant_id: "", name: "北京-移动", kind: "wan", isp: "mobile", role: "primary", status: "degraded", latency_ms: 46, jitter_ms: 18, loss: 0.012, score: 71 },
  { id: "l4", tenant_id: "", name: "广州-电信", kind: "wan", isp: "telecom", role: "primary", status: "down", latency_ms: 220, jitter_ms: 80, loss: 0.045, score: 38 },
];

function statusColor(s: string): string {
  if (s === "up") return "hsl(152 60% 45%)";
  if (s === "degraded") return "hsl(38 92% 55%)";
  if (s === "down") return "hsl(0 72% 55%)";
  return "hsl(215 20% 65%)";
}

export default async function TopologyPage() {
  let links: Link[] = [];
  let devices: Device[] = [];
  try {
    [links, devices] = await Promise.all([listLinks(serverToken()), listDevices(serverToken())]);
  } catch {
    /* fall through to demo */
  }
  if (links.length === 0) links = demoLinks;

  const cx = 480;
  const cy = 230;
  const radius = 170;
  const nodes = links.map((l, i) => {
    const angle = (Math.PI * 2 * i) / Math.max(links.length, 1) - Math.PI / 2;
    return {
      link: l,
      x: cx + radius * Math.cos(angle),
      y: cy + radius * Math.sin(angle),
    };
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">网络拓扑</h1>
        <p className="mt-1 text-sm text-muted">
          中心控制平面 · 站点 / POP / 链路（按链路状态着色）— {devices.length} 设备 / {links.length} 链路
        </p>
      </div>

      <Card className="overflow-x-auto">
        <svg viewBox="0 0 960 460" className="h-[460px] w-full">
          {nodes.map((n) => (
            <line
              key={`edge-${n.link.id}`}
              x1={cx}
              y1={cy}
              x2={n.x}
              y2={n.y}
              stroke={statusColor(n.link.status)}
              strokeWidth={2 + (n.link.score / 100) * 3}
              strokeOpacity={0.7}
              strokeDasharray={n.link.kind === "overlay" ? "6 4" : undefined}
            />
          ))}

          {/* Center: control plane / hub */}
          <g>
            <circle cx={cx} cy={cy} r={44} fill="hsl(199 89% 52% / 0.18)" stroke="hsl(199 89% 52%)" strokeWidth={2} />
            <text x={cx} y={cy - 2} textAnchor="middle" fill="hsl(210 40% 96%)" fontSize="13" fontWeight="600">Neko</text>
            <text x={cx} y={cy + 14} textAnchor="middle" fill="hsl(215 20% 65%)" fontSize="10">Control Plane</text>
          </g>

          {/* Spokes: links/POPs */}
          {nodes.map((n) => (
            <g key={`node-${n.link.id}`}>
              <circle cx={n.x} cy={n.y} r={30} fill="hsl(222 38% 14%)" stroke={statusColor(n.link.status)} strokeWidth={2} />
              <circle cx={n.x + 22} cy={n.y - 22} r={5} fill={statusColor(n.link.status)} />
              <text x={n.x} y={n.y - 2} textAnchor="middle" fill="hsl(210 40% 96%)" fontSize="11" fontWeight="500">
                {n.link.name.length > 7 ? n.link.name.slice(0, 7) : n.link.name}
              </text>
              <text x={n.x} y={n.y + 12} textAnchor="middle" fill="hsl(215 20% 65%)" fontSize="10" fontFamily="monospace">
                {n.link.score.toFixed(0)} · {n.link.latency_ms}ms
              </text>
            </g>
          ))}
        </svg>

        <div className="mt-4 flex flex-wrap gap-4 text-xs text-muted">
          <Legend color="hsl(152 60% 45%)" label="健康 up" />
          <Legend color="hsl(38 92% 55%)" label="降级 degraded" />
          <Legend color="hsl(0 72% 55%)" label="故障 down" />
          <span className="text-muted">虚线 = Overlay 隧道 · 线宽 = 评分</span>
        </div>
      </Card>
    </div>
  );
}

function Legend({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />
      {label}
    </span>
  );
}
