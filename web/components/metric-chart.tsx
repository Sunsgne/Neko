import { type MetricSeries } from "@/lib/api";

// MetricChart renders a simple SVG area chart for a 0..100 metric series.
export function MetricChart({
  series,
  label,
  unit = "%",
  color = "hsl(199 89% 52%)",
  max = 100,
}: {
  series?: MetricSeries;
  label: string;
  unit?: string;
  color?: string;
  max?: number;
}) {
  const pts = series?.points ?? [];
  const last = pts.length ? pts[pts.length - 1].v : null;

  if (pts.length < 2) {
    return (
      <div className="rounded-lg border border-border bg-elevated/40 p-4">
        <div className="mb-1 text-xs uppercase tracking-wide text-muted">{label}</div>
        <div className="py-6 text-center text-xs text-muted">采集中…（轮询累积后显示曲线）</div>
      </div>
    );
  }

  const w = 100;
  const h = 32;
  const n = pts.length;
  const coords = pts.map((p, i) => {
    const x = (i / (n - 1)) * w;
    const y = h - Math.min(p.v / max, 1) * h;
    return `${x.toFixed(2)},${y.toFixed(2)}`;
  });
  const lineId = `g-${label}`;

  return (
    <div className="rounded-lg border border-border bg-elevated/40 p-4">
      <div className="mb-2 flex items-baseline justify-between">
        <span className="text-xs uppercase tracking-wide text-muted">{label}</span>
        <span className="font-mono text-sm" style={{ color }}>
          {last != null ? `${last.toFixed(0)}${unit}` : "—"}
        </span>
      </div>
      <svg viewBox={`0 0 ${w} ${h}`} className="h-16 w-full" preserveAspectRatio="none">
        <defs>
          <linearGradient id={lineId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.35" />
            <stop offset="100%" stopColor={color} stopOpacity="0" />
          </linearGradient>
        </defs>
        <polyline points={coords.join(" ")} fill="none" stroke={color} strokeWidth="0.8" vectorEffect="non-scaling-stroke" />
        <polygon points={`0,${h} ${coords.join(" ")} ${w},${h}`} fill={`url(#${lineId})`} />
      </svg>
      <div className="mt-1 text-right text-[10px] text-muted">{pts.length} 点 · 最近 1 小时</div>
    </div>
  );
}
