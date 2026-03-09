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
      <Card className={cn('group overflow-hidden', href && 'cursor-pointer')}>
        <CardContent className="p-6">
          <div className="flex items-start justify-between gap-4">
            <div className="space-y-3">
              <span className="inline-flex items-center rounded-full border border-border/70 bg-secondary/70 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                {title}
              </span>
              {isLoading ? (
                <div className="h-10 w-20 animate-pulse rounded-xl bg-muted" />
              ) : isError ? (
                <p className="text-2xl font-bold text-muted-foreground" title="Failed to load">
                  —
                </p>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-3xl font-semibold tracking-[-0.04em] text-slate-950">
                    {value ?? '—'}
                    {unit && <span className="ml-1 text-base font-medium text-muted-foreground">{unit}</span>}
                  </p>
                  {showDelta && liveDelta ? (
                    <span className="rounded-full bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-700">
                      {liveDelta > 0 ? `↑${liveDelta}` : `${liveDelta}`}
                    </span>
                  ) : null}
                </div>
              )}
            </div>
            <div className={cn('flex h-11 w-11 items-center justify-center rounded-2xl border border-white/70 bg-gradient-to-br from-white via-secondary/60 to-secondary shadow-sm', iconColor)}>
              <Icon className="h-5 w-5" />
            </div>
          </div>
          {trend && !isLoading && !isError && (
            <div className="mt-4 flex items-center gap-2 text-xs">
              {trend.direction === 'up' ? (
                <TrendingUp className={cn('h-3.5 w-3.5', trend.sentiment === 'bad' ? 'text-destructive' : 'text-green-600')} />
              ) : trend.direction === 'down' ? (
                <TrendingDown className={cn('h-3.5 w-3.5', trend.sentiment === 'good' ? 'text-green-600' : 'text-destructive')} />
              ) : (
                <Minus className="h-3.5 w-3.5 text-muted-foreground" />
              )}
              <span
                className={cn(
                  'inline-flex items-center rounded-full px-3 py-1 font-medium',
                  trend.direction === 'neutral' && 'bg-muted text-muted-foreground',
                  trend.direction === 'up' && trend.sentiment === 'bad' && 'bg-red-50 text-destructive',
                  trend.direction === 'up' && trend.sentiment === 'good' && 'bg-emerald-50 text-green-600',
                  trend.direction === 'down' && trend.sentiment === 'good' && 'bg-emerald-50 text-green-600',
                  trend.direction === 'down' && trend.sentiment === 'bad' && 'bg-red-50 text-destructive',
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
