'use client';

import { Activity, BadgeAlert, Clock3, ShieldAlert, Siren } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { cn } from '@/lib/utils';
import type { AlertStats, NamedCount } from '@/types/cyber';

interface AlertStatsBarProps {
  onFilterByStatus?: (status: string[]) => void;
}

export function AlertStatsBar({ onFilterByStatus }: AlertStatsBarProps) {
  const { data: envelope } = useRealtimeData<{ data: AlertStats }>(
    API_ENDPOINTS.CYBER_ALERTS_STATS,
    {
      pollInterval: 60000,
      wsTopics: ['cyber.alert.created', 'cyber.alert.status_changed', 'cyber.alert.escalated'],
    },
  );

  const stats = envelope?.data;
  const byStatus = Object.fromEntries((stats?.by_status ?? []).map((item) => [item.name, item.count]));

  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
      <StatCard
        title="New Alerts"
        value={byStatus.new ?? 0}
        subtitle="Awaiting triage"
        icon={Siren}
        tone="red"
        pulse={(byStatus.new ?? 0) > 0}
        onClick={() => onFilterByStatus?.(['new'])}
      />
      <StatCard
        title="Investigating"
        value={sumCounts(stats?.by_status ?? [], ['investigating', 'in_progress'])}
        subtitle="Active analyst workload"
        icon={BadgeAlert}
        tone="amber"
        onClick={() => onFilterByStatus?.(['investigating', 'in_progress'])}
      />
      <StatCard
        title="False Positive Rate"
        value={formatPercent(stats?.false_positive_rate)}
        subtitle="Rule feedback quality"
        icon={ShieldAlert}
        tone="purple"
      />
      <StatCard
        title="Mean Time to Acknowledge"
        value={formatHours(stats?.mtta_hours)}
        subtitle="Average first response"
        icon={Clock3}
        tone="blue"
      />
      <StatCard
        title="Mean Time to Resolve"
        value={formatHours(stats?.mttr_hours)}
        subtitle="Average containment cycle"
        icon={Activity}
        tone="green"
      />
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle: string;
  icon: typeof Activity;
  tone: 'red' | 'amber' | 'purple' | 'blue' | 'green';
  pulse?: boolean;
  onClick?: () => void;
}

function StatCard({ title, value, subtitle, icon: Icon, tone, pulse = false, onClick }: StatCardProps) {
  const clickable = typeof onClick === 'function';

  return (
    <Card className={cn(
      'overflow-hidden border-[color:var(--card-border)] bg-[rgba(255,255,255,0.82)] shadow-[var(--card-shadow)] backdrop-blur-md',
      clickable && 'cursor-pointer transition-transform hover:-translate-y-0.5',
    )}>
      <button
        type="button"
        onClick={onClick}
        disabled={!clickable}
        className={cn('w-full text-left', !clickable && 'cursor-default')}
      >
        <CardContent className="flex items-start justify-between gap-4 p-4">
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                {title}
              </span>
              {pulse && <span className="h-2.5 w-2.5 rounded-full bg-red-500 animate-pulse" aria-hidden="true" />}
            </div>
            <div className="text-3xl font-semibold tracking-[-0.05em] text-slate-950">
              {value}
            </div>
            <p className="text-sm text-muted-foreground">{subtitle}</p>
          </div>
          <div className={cn(
            'flex h-11 w-11 items-center justify-center rounded-2xl border',
            toneClasses[tone],
          )}>
            <Icon className="h-5 w-5" />
          </div>
        </CardContent>
      </button>
    </Card>
  );
}

const toneClasses = {
  red: 'border-red-200 bg-red-50 text-red-600',
  amber: 'border-amber-200 bg-amber-50 text-amber-600',
  purple: 'border-purple-200 bg-purple-50 text-purple-600',
  blue: 'border-blue-200 bg-blue-50 text-blue-600',
  green: 'border-emerald-200 bg-emerald-50 text-emerald-600',
} as const;

function sumCounts(items: NamedCount[], names: string[]): number {
  const lookup = new Set(names);
  return items.reduce((total, item) => (
    lookup.has(item.name) ? total + item.count : total
  ), 0);
}

function formatPercent(value?: number): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '0%';
  }
  return `${(value * 100).toFixed(1)}%`;
}

function formatHours(value?: number): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '0h';
  }
  return `${value.toFixed(1)}h`;
}
