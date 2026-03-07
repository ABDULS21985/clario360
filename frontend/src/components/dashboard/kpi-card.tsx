'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Card, CardContent } from '@/components/ui/card';
import { HighlightAnimation } from '@/components/realtime/highlight-animation';

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
  highlightKey?: number | null;
  liveDelta?: number | null;
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
  highlightKey = null,
  liveDelta = null,
}: KpiCardProps) {
  const [showDelta, setShowDelta] = useState(false);

  useEffect(() => {
    if (!liveDelta) {
      return;
    }

    setShowDelta(true);
    const timeout = window.setTimeout(() => setShowDelta(false), 3000);
    return () => window.clearTimeout(timeout);
  }, [liveDelta]);

  const content = (
    <HighlightAnimation highlight={highlightKey !== null} highlightKey={highlightKey}>
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
                <div className="flex items-center gap-2">
                  <p className="text-2xl font-bold">
                    {value ?? '—'}
                    {unit && <span className="ml-1 text-sm font-normal text-muted-foreground">{unit}</span>}
                  </p>
                  {showDelta && liveDelta ? (
                    <span className="text-xs font-medium text-yellow-700">
                      {liveDelta > 0 ? `↑${liveDelta}` : `${liveDelta}`}
                    </span>
                  ) : null}
                </div>
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
    </HighlightAnimation>
  );

  if (href) {
    return <Link href={href}>{content}</Link>;
  }
  return content;
}
