import { TrendingUp, TrendingDown, Minus, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

type KpiColorTheme =
  | "red" | "orange" | "amber" | "yellow"
  | "green" | "emerald" | "teal" | "cyan"
  | "sky" | "blue" | "indigo" | "violet"
  | "purple" | "pink" | "primary";

interface KpiCardProps {
  title: string;
  value: string | number;
  change?: number;
  changeLabel?: string;
  icon?: LucideIcon;
  iconColor?: string;
  /** Explicit color theme. When omitted, auto-derived from iconColor. */
  colorTheme?: KpiColorTheme;
  description?: string;
  loading?: boolean;
  className?: string;
}

/** Map common Tailwind icon-color class fragments to theme names. */
function deriveTheme(iconColor: string | undefined): KpiColorTheme {
  if (!iconColor) return "primary";
  const c = iconColor.toLowerCase();
  const families: KpiColorTheme[] = [
    "emerald", "orange", "amber", "yellow",
    "green", "teal", "cyan", "sky",
    "blue", "indigo", "violet", "purple",
    "pink", "red",
  ];
  for (const f of families) {
    if (c.includes(f)) return f;
  }
  if (c.includes("destructive")) return "red";
  return "primary";
}

export function KpiCard({
  title,
  value,
  change,
  changeLabel,
  icon: Icon,
  iconColor = "text-primary",
  colorTheme,
  description,
  loading = false,
  className,
}: KpiCardProps) {
  const theme = colorTheme ?? deriveTheme(iconColor);
  const isPositive = change !== undefined && change > 0;
  const isNegative = change !== undefined && change < 0;
  const TrendIcon = isPositive ? TrendingUp : isNegative ? TrendingDown : Minus;
  const trendColor = isPositive ? "text-green-600" : isNegative ? "text-red-600" : "text-muted-foreground";

  if (loading) {
    return (
      <div className={cn("kpi-card-themed kpi-theme-primary", className)}>
        <div className="mb-3 flex items-center gap-2.5">
          <div className="h-9 w-9 animate-pulse rounded-xl bg-muted" />
          <div className="h-4 w-24 animate-pulse rounded-full bg-muted" />
        </div>
        <div className="mb-2 h-9 w-28 animate-pulse rounded-xl bg-muted" />
        <div className="h-3 w-20 animate-pulse rounded bg-muted" />
      </div>
    );
  }

  return (
    <div className={cn(`kpi-card-themed kpi-theme-${theme}`, className)}>
      <div className="mb-3 flex items-center gap-2.5">
        {Icon && (
          <div className="kpi-icon-badge">
            <Icon className="h-[18px] w-[18px]" aria-hidden />
          </div>
        )}
        <span className="text-[11px] font-semibold uppercase tracking-[0.15em]" style={{ color: "var(--kpi-accent)" }}>
          {title}
        </span>
      </div>

      <div className="text-[1.75rem] font-bold leading-none tracking-tight text-slate-900">
        {typeof value === "number" ? value.toLocaleString() : value}
      </div>

      {change !== undefined && (
        <div
          className={cn(
            "mt-2.5 inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium",
            trendColor,
            isPositive && "bg-emerald-50",
            isNegative && "bg-red-50",
            !isPositive && !isNegative && "bg-muted"
          )}
        >
          <TrendIcon className="h-3 w-3 shrink-0" aria-hidden />
          <span>
            {isPositive && "+"}
            {change.toFixed(1)}%
            {changeLabel && <span className="ml-1 text-muted-foreground">{changeLabel}</span>}
          </span>
        </div>
      )}

      {description && (
        <p className="mt-2 text-xs leading-5" style={{ color: "var(--kpi-accent)", opacity: 0.6 }}>
          {description}
        </p>
      )}
    </div>
  );
}
