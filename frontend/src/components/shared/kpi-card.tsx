import { TrendingUp, TrendingDown, Minus, type LucideIcon } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface KpiCardProps {
  title: string;
  value: string | number;
  change?: number; // percentage change, e.g. 12.5 = +12.5%
  changeLabel?: string;
  icon?: LucideIcon;
  iconColor?: string;
  description?: string;
  loading?: boolean;
  className?: string;
}

export function KpiCard({
  title,
  value,
  change,
  changeLabel,
  icon: Icon,
  iconColor = "text-primary",
  description,
  loading = false,
  className,
}: KpiCardProps) {
  const isPositive = change !== undefined && change > 0;
  const isNegative = change !== undefined && change < 0;
  const TrendIcon = isPositive ? TrendingUp : isNegative ? TrendingDown : Minus;
  const trendColor = isPositive ? "text-green-600" : isNegative ? "text-red-600" : "text-muted-foreground";

  if (loading) {
    return (
      <Card className={cn("overflow-hidden", className)}>
        <CardHeader className="p-4 pb-3 sm:p-6 sm:pb-3">
          <div className="h-4 w-28 rounded-full animate-pulse bg-muted" />
        </CardHeader>
        <CardContent className="p-4 pt-0 sm:p-6 sm:pt-0">
          <div className="mb-3 h-9 w-32 rounded-xl animate-pulse bg-muted" />
          <div className="h-3 w-24 rounded animate-pulse bg-muted" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={cn("group overflow-hidden", className)}>
      <CardHeader className="flex flex-row items-start justify-between gap-3 p-4 pb-3 space-y-0 sm:p-6 sm:pb-3">
        <div className="space-y-2">
          <span className="inline-flex items-center rounded-full border border-border/70 bg-secondary/70 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            {title}
          </span>
        </div>
        {Icon && (
          <div className="flex h-11 w-11 items-center justify-center rounded-2xl border border-white/70 bg-gradient-to-br from-white via-secondary/60 to-secondary shadow-sm">
            <Icon className={cn("h-5 w-5 shrink-0", iconColor)} aria-hidden />
          </div>
        )}
      </CardHeader>
      <CardContent className="p-4 pt-0 sm:p-6 sm:pt-0">
        <div className="text-2xl font-semibold tracking-[-0.04em] text-slate-950 sm:text-3xl">
          {value}
        </div>
        {change !== undefined && (
          <div
            className={cn(
              "mt-3 inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium",
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
              {changeLabel && <span className="text-muted-foreground ml-1">{changeLabel}</span>}
            </span>
          </div>
        )}
        {description && <p className="mt-3 text-sm leading-6 text-muted-foreground">{description}</p>}
      </CardContent>
    </Card>
  );
}
