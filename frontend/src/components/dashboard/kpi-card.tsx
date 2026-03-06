'use client';

import Link from 'next/link';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Card, CardContent } from '@/components/ui/card';

interface KpiCardProps {
  title: string;
  value: number | string | undefined;
  unit?: string;
  icon: LucideIcon;
  iconColor?: string;
  href?: string;
  trend?: {
    value: number;
    label: string;
    direction: 'up' | 'down' | 'neutral';
    sentiment: 'good' | 'bad' | 'neutral';
  };
  isLoading?: boolean;
  isError?: boolean;
}

export function KpiCard({
  title,
  value,
  unit,
  icon: Icon,
  iconColor = 'text-primary',
  href,
  trend,
  isLoading,
  isError,
}: KpiCardProps) {
  const content = (
    <Card className={cn(href && 'cursor-pointer transition-shadow hover:shadow-md')}>
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">{title}</p>
            {isLoading ? (
              <div className="h-8 w-16 animate-pulse rounded bg-muted" />
            ) : isError ? (
              <p className="text-2xl font-bold text-muted-foreground" title="Failed to load">
                —
              </p>
            ) : (
              <p className="text-2xl font-bold">
                {value ?? '—'}
                {unit && <span className="text-sm font-normal text-muted-foreground ml-1">{unit}</span>}
              </p>
            )}
          </div>
          <div className={cn('rounded-lg bg-muted p-2', iconColor)}>
            <Icon className="h-5 w-5" />
          </div>
        </div>
        {trend && !isLoading && !isError && (
          <div className="mt-3 flex items-center gap-1 text-xs">
            {trend.direction === 'up' ? (
              <TrendingUp className={cn('h-3.5 w-3.5', trend.sentiment === 'bad' ? 'text-destructive' : 'text-green-600')} />
            ) : trend.direction === 'down' ? (
              <TrendingDown className={cn('h-3.5 w-3.5', trend.sentiment === 'good' ? 'text-green-600' : 'text-destructive')} />
            ) : (
              <Minus className="h-3.5 w-3.5 text-muted-foreground" />
            )}
            <span
              className={cn(
                trend.direction === 'neutral' && 'text-muted-foreground',
                trend.direction === 'up' && trend.sentiment === 'bad' && 'text-destructive',
                trend.direction === 'up' && trend.sentiment === 'good' && 'text-green-600',
                trend.direction === 'down' && trend.sentiment === 'good' && 'text-green-600',
                trend.direction === 'down' && trend.sentiment === 'bad' && 'text-destructive',
              )}
            >
              {trend.value > 0 ? '+' : ''}{trend.value} {trend.label}
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  );

  if (href) {
    return <Link href={href}>{content}</Link>;
  }
  return content;
}
