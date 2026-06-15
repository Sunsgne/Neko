import * as React from "react";
import { cn } from "@/lib/cn";

export function Card({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("card", className)} {...props} />;
}

export function CardHeader({
  title,
  subtitle,
  action,
  className,
  inset,
  border,
}: {
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
  className?: string;
  /** Adds horizontal padding for cards with p-0 */
  inset?: boolean;
  /** Bottom border separator */
  border?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-start justify-between gap-4",
        inset ? "px-4 pt-4" : "mb-4",
        border && "border-b border-border pb-3",
        className,
      )}
    >
      <div className="min-w-0">
        <h3 className="text-sm font-semibold tracking-tight">{title}</h3>
        {subtitle && <p className="mt-0.5 text-xs text-muted">{subtitle}</p>}
      </div>
      {action}
    </div>
  );
}

export function EmptyState({
  icon,
  title,
  description,
  className,
}: {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-1 flex-col items-center justify-center gap-2 px-6 py-16 text-center", className)}>
      {icon && <div className="mb-1 text-primary [&>svg]:h-7 [&>svg]:w-7">{icon}</div>}
      <p className="text-sm font-medium text-foreground/90">{title}</p>
      {description && <p className="max-w-sm text-xs leading-relaxed text-muted">{description}</p>}
    </div>
  );
}

/** Preview panel header: context subtitle only when data is available. */
export function PreviewPanelHeader({
  title,
  context,
  action,
}: {
  title: string;
  context?: string;
  action?: React.ReactNode;
}) {
  return (
    <CardHeader
      title={title}
      subtitle={context}
      action={action}
      inset
      border
    />
  );
}

type Tone = "neutral" | "primary" | "success" | "warning" | "danger";

const toneClasses: Record<Tone, string> = {
  neutral: "bg-border/60 text-muted",
  primary: "bg-primary/15 text-primary",
  success: "bg-success/15 text-success",
  warning: "bg-warning/15 text-warning",
  danger: "bg-danger/15 text-danger",
};

export function Badge({ tone = "neutral", children }: { tone?: Tone; children: React.ReactNode }) {
  return <span className={cn("pill", toneClasses[tone])}>{children}</span>;
}

export function StatusDot({ tone = "neutral" }: { tone?: Tone }) {
  const dot: Record<Tone, string> = {
    neutral: "bg-muted",
    primary: "bg-primary",
    success: "bg-success",
    warning: "bg-warning",
    danger: "bg-danger",
  };
  return <span className={cn("inline-block h-2 w-2 rounded-full", dot[tone])} />;
}

export function Kpi({
  label,
  value,
  delta,
  tone = "primary",
  icon,
}: {
  label: string;
  value: string;
  delta?: string;
  tone?: Tone;
  icon?: React.ReactNode;
}) {
  return (
    <Card className="relative overflow-hidden">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs uppercase tracking-wide text-muted">{label}</p>
          <p className="mt-2 text-3xl font-semibold tracking-tight">{value}</p>
          {delta && <p className="mt-1 text-xs text-muted">{delta}</p>}
        </div>
        <div className={cn("rounded-lg p-2", toneClasses[tone])}>{icon}</div>
      </div>
    </Card>
  );
}
