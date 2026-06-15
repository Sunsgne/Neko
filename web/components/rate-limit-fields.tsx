"use client";

import * as React from "react";
import { Gauge } from "lucide-react";

const PRESETS = ["", "1M", "5M", "10M", "20M", "50M", "100M"];

export type RateLimitValue = {
  enabled: boolean;
  rateLimit: string;
  rateTarget: string;
};

export function defaultRateLimit(): RateLimitValue {
  return { enabled: false, rateLimit: "10M", rateTarget: "" };
}

export function rateLimitPayload(v: RateLimitValue): { rate_limit?: string; rate_target?: string } {
  if (!v.enabled || !v.rateLimit.trim()) return {};
  return {
    rate_limit: v.rateLimit.trim(),
    ...(v.rateTarget.trim() ? { rate_target: v.rateTarget.trim() } : {}),
  };
}

export function RateLimitSection({
  value,
  onChange,
  targetHint = "留空则对内网前缀限速",
}: {
  value: RateLimitValue;
  onChange: (v: RateLimitValue) => void;
  targetHint?: string;
}) {
  return (
    <div className="rounded-lg border border-border bg-elevated/30 p-3 space-y-2">
      <label className="flex cursor-pointer items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={value.enabled}
          onChange={(e) => onChange({ ...value, enabled: e.target.checked })}
        />
        <Gauge className="h-4 w-4 text-primary" />
        <span className="font-medium">下发限速 (Simple Queue)</span>
      </label>
      {value.enabled && (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-[11px] uppercase tracking-wide text-muted">max-limit</label>
            <div className="flex gap-1">
              <input
                value={value.rateLimit}
                onChange={(e) => onChange({ ...value, rateLimit: e.target.value })}
                placeholder="10M 或 10M/5M"
                className="w-full rounded-md border border-border bg-elevated px-2 py-1.5 font-mono text-sm outline-none focus:border-primary"
              />
              <select
                value={value.rateLimit}
                onChange={(e) => onChange({ ...value, rateLimit: e.target.value })}
                className="rounded-md border border-border bg-elevated px-2 text-xs outline-none"
              >
                {PRESETS.map((p) => (
                  <option key={p || "off"} value={p}>{p || "自定义"}</option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="mb-1 block text-[11px] uppercase tracking-wide text-muted">target（可选）</label>
            <input
              value={value.rateTarget}
              onChange={(e) => onChange({ ...value, rateTarget: e.target.value })}
              placeholder={targetHint}
              className="w-full rounded-md border border-border bg-elevated px-2 py-1.5 font-mono text-sm outline-none focus:border-primary"
            />
          </div>
        </div>
      )}
    </div>
  );
}
