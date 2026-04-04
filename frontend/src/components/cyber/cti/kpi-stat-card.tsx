'use client';

import type { ReactNode } from 'react';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface KPIStatCardProps {
  label: string;
  value: number | string;
  subtitle?: string;
  trend?: { direction: 'increasing' | 'decreasing' | 'stable' | 'up' | 'down'; percentage: number };
  color?: string;
  icon?: ReactNode;
  onClick?: () => void;
  className?: string;
}

export function CTIKPIStatCard({
  label,
  value,
  subtitle,
  trend,
  color,
  icon,
  onClick,
  className,
}: KPIStatCardProps) {
  const formatted = typeof value === 'number' ? value.toLocaleString() : value;

  const normalizedDirection =
    trend?.direction === 'up'
      ? 'increasing'
      : trend?.direction === 'down'
        ? 'decreasing'
        : trend?.direction;

  const TrendIcon = normalizedDirection === 'increasing' ? TrendingUp
    : normalizedDirection === 'decreasing' ? TrendingDown
    : Minus;

  const trendColor = normalizedDirection === 'increasing' ? 'text-red-400'
    : normalizedDirection === 'decreasing' ? 'text-green-400'
    : 'text-slate-400';

  return (
    <Card
      className={cn(onClick && 'cursor-pointer transition-transform hover:-translate-y-0.5', className)}
      onClick={onClick}
      style={color ? { borderColor: color } : undefined}
    >
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-3">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{label}</p>
          {icon && <div className="text-muted-foreground">{icon}</div>}
        </div>
        <div className="mt-1 flex items-baseline gap-2">
          <span className="text-2xl font-bold tabular-nums" style={color ? { color } : undefined}>{formatted}</span>
          {trend && (
            <span className={`flex items-center gap-0.5 text-xs font-medium ${trendColor}`}>
              <TrendIcon className="h-3 w-3" />
              {Math.abs(trend.percentage).toFixed(1)}%
            </span>
          )}
        </div>
        {subtitle && <p className="mt-0.5 text-xs text-muted-foreground">{subtitle}</p>}
      </CardContent>
    </Card>
  );
}
