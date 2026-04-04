'use client';

import { useMemo } from 'react';
import {
  Activity,
  BarChart3,
  Clock,
  Fingerprint,
  Shield,
  Zap,
} from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import { cn } from '@/lib/utils';
import { getIndicatorSourceLabel } from '@/lib/cyber-indicators';
import type { IndicatorStats as IndicatorStatsType } from '@/types/cyber';

interface IndicatorStatsProps {
  stats?: IndicatorStatsType;
  loading?: boolean;
}

const SOURCE_COLORS: Record<string, { bar: string; dot: string }> = {
  osint:    { bar: 'from-violet-500 to-purple-600',   dot: 'bg-violet-500' },
  stix:     { bar: 'from-blue-500 to-indigo-600',     dot: 'bg-blue-500' },
  internal: { bar: 'from-emerald-500 to-teal-600',    dot: 'bg-emerald-500' },
  vendor:   { bar: 'from-amber-400 to-orange-500',    dot: 'bg-amber-500' },
  manual:   { bar: 'from-rose-400 to-pink-500',       dot: 'bg-rose-500' },
};

function getSourceColor(name: string) {
  return SOURCE_COLORS[name.toLowerCase()] ?? { bar: 'from-slate-400 to-slate-500', dot: 'bg-slate-400' };
}

export function IndicatorStats({ stats, loading = false }: IndicatorStatsProps) {
  const sourceBreakdown = useMemo(() => {
    const total = Math.max(
      1,
      (stats?.by_source ?? []).reduce((sum, item) => sum + item.count, 0),
    );

    return (stats?.by_source ?? []).map((item) => ({
      ...item,
      label: getIndicatorSourceLabel(item.name),
      percent: (item.count / total) * 100,
    }));
  }, [stats?.by_source]);

  return (
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
      <KpiCard
        title="Total IOCs"
        value={stats?.total ?? 0}
        icon={Fingerprint}
        iconColor="text-indigo-600"
        colorTheme="indigo"
        description="Across all sources & types"
        loading={loading}
      />
      <KpiCard
        title="Active IOCs"
        value={stats?.active ?? 0}
        icon={Shield}
        iconColor="text-emerald-600"
        colorTheme="emerald"
        description={stats?.total ? `${Math.round(((stats.active ?? 0) / stats.total) * 100)}% detection rate` : 'Monitoring'}
        loading={loading}
      />
      <KpiCard
        title="Expiring Soon"
        value={stats?.expiring_soon ?? 0}
        icon={Clock}
        iconColor="text-amber-600"
        colorTheme="amber"
        description="Within next 7 days"
        loading={loading}
      />

      {/* Source Mix — custom card */}
      <div className="kpi-card-themed kpi-theme-primary">
        <div className="mb-3 flex items-center gap-2.5">
          <div className="kpi-icon-badge">
            <BarChart3 className="h-[18px] w-[18px]" />
          </div>
          <span className="text-[11px] font-semibold uppercase tracking-[0.15em]" style={{ color: 'var(--kpi-accent)' }}>
            Source Mix
          </span>
        </div>

        <div className="space-y-2.5">
          {loading ? (
            Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="space-y-1.5">
                <div className="h-3 w-24 animate-pulse rounded bg-muted" />
                <div className="h-1.5 animate-pulse rounded-full bg-muted" />
              </div>
            ))
          ) : sourceBreakdown.length > 0 ? (
            sourceBreakdown.map((item) => {
              const colors = getSourceColor(item.name);
              return (
                <div key={item.name} className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <div className="flex items-center gap-1.5">
                      <span className={cn('h-2 w-2 rounded-full', colors.dot)} />
                      <span className="font-medium text-slate-600">{item.label}</span>
                    </div>
                    <span className="font-bold tabular-nums text-slate-900">{item.count.toLocaleString()}</span>
                  </div>
                  <div className="h-1.5 rounded-full bg-slate-100">
                    <div
                      className={cn('h-full rounded-full bg-gradient-to-r transition-all duration-500', colors.bar)}
                      style={{ width: `${Math.max(item.percent, 4)}%` }}
                    />
                  </div>
                </div>
              );
            })
          ) : (
            <p className="text-sm text-muted-foreground">No source telemetry yet.</p>
          )}
        </div>
      </div>
    </div>
  );
}
