'use client';

import Link from 'next/link';
import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { TrendingUp, TrendingDown, Minus, ExternalLink } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { HighlightAnimation } from '@/components/realtime/highlight-animation';

type KpiColorTheme =
  | 'red' | 'orange' | 'amber' | 'yellow'
  | 'green' | 'emerald' | 'teal' | 'cyan'
  | 'sky' | 'blue' | 'indigo' | 'violet'
  | 'purple' | 'pink' | 'primary';

function deriveThemeFromIconColor(iconColor: string): KpiColorTheme {
  const c = iconColor.toLowerCase();
  const families: KpiColorTheme[] = [
    'emerald', 'orange', 'amber', 'yellow',
    'green', 'teal', 'cyan', 'sky',
    'blue', 'indigo', 'violet', 'purple',
    'pink', 'red',
  ];
  for (const f of families) {
    if (c.includes(f)) return f;
  }
  if (c.includes('destructive')) return 'red';
  return 'primary';
}

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
  index?: number;
  children?: React.ReactNode;
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
  index = 0,
  children,
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
      <motion.div
        initial={{ opacity: 0, y: 20, scale: 0.97 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{
          duration: 0.45,
          delay: index * 0.08,
          type: 'spring',
          damping: 25,
        }}
        className={cn(
          `kpi-card-themed kpi-theme-${deriveThemeFromIconColor(iconColor)}`,
          'group/kpi flex h-full flex-col justify-between p-6',
          'hover:-translate-y-0.5',
          href && 'cursor-pointer',
        )}
      >
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-3">
            <span className="inline-flex items-center rounded-full border border-border/50 bg-secondary/60 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              {title}
            </span>
            {isLoading ? (
              <div className="h-10 w-20 animate-pulse rounded-xl bg-muted/60" />
            ) : isError ? (
              <p className="text-2xl font-bold text-muted-foreground" title="Failed to load">
                —
              </p>
            ) : (
              <div className="flex items-center gap-2">
                <p className="text-3xl font-bold tabular-nums tracking-[-0.04em] text-foreground">
                  {value ?? '—'}
                  {unit && <span className="ml-1 text-base font-medium text-muted-foreground">{unit}</span>}
                </p>
                {showDelta && liveDelta ? (
                  <motion.span
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.8 }}
                    className={cn(
                      'rounded-full px-2.5 py-1 text-xs font-semibold',
                      liveDelta > 0 ? 'bg-amber-50 text-amber-700' : 'bg-emerald-50 text-emerald-700',
                    )}
                  >
                    {liveDelta > 0 ? `+${liveDelta}` : `${liveDelta}`}
                  </motion.span>
                ) : null}
              </div>
            )}
          </div>
          <div className="flex flex-col items-end gap-2">
            <div className="kpi-icon-badge h-11 w-11 rounded-2xl transition-transform duration-200 group-hover/kpi:scale-110">
              <Icon className="h-5 w-5" />
            </div>
            {href && (
              <ExternalLink
                className="h-3 w-3 text-muted-foreground opacity-0 transition-opacity group-hover/kpi:opacity-100"
              />
            )}
          </div>
        </div>

        {/* Trend + optional sparkline children */}
        <div className="mt-4 flex items-center justify-between gap-3">
          {trend && !isLoading && !isError ? (
            <div className="flex items-center gap-2 text-xs">
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
          ) : (
            <div />
          )}
          {children && !isLoading && <div className="flex-shrink-0">{children}</div>}
        </div>
      </motion.div>
    </HighlightAnimation>
  );

  if (href) {
    return <Link href={href}>{content}</Link>;
  }
  return content;
}
