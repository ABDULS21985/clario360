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
      <Card className={cn("", className)}>
        <CardHeader className="pb-2">
          <div className="h-4 w-24 rounded animate-pulse bg-muted" />
        </CardHeader>
        <CardContent>
          <div className="h-8 w-32 rounded animate-pulse bg-muted mb-2" />
          <div className="h-3 w-20 rounded animate-pulse bg-muted" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={cn("", className)}>
      <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        {Icon && <Icon className={cn("h-5 w-5 shrink-0", iconColor)} aria-hidden />}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {change !== undefined && (
          <div className={cn("flex items-center gap-1 mt-1 text-xs", trendColor)}>
            <TrendIcon className="h-3 w-3 shrink-0" aria-hidden />
            <span>
              {isPositive && "+"}
              {change.toFixed(1)}%
              {changeLabel && <span className="text-muted-foreground ml-1">{changeLabel}</span>}
            </span>
          </div>
        )}
        {description && <p className="text-xs text-muted-foreground mt-1">{description}</p>}
      </CardContent>
    </Card>
  );
}
