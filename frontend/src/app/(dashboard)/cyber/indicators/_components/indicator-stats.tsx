'use client';

import { useMemo } from 'react';
import { BarChart3 } from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import { cn } from '@/lib/utils';
import { getIndicatorSourceLabel } from '@/lib/cyber-indicators';
import type { IndicatorStats as IndicatorStatsType } from '@/types/cyber';

interface IndicatorStatsProps {
  stats?: IndicatorStatsType;
  loading?: boolean;
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
        loading={loading}
      />
      <KpiCard
        title="Active IOCs"
        value={stats?.active ?? 0}
        iconColor="text-emerald-600"
        loading={loading}
      />
      <KpiCard
        title="Expiring Soon"
        value={stats?.expiring_soon ?? 0}
        iconColor="text-amber-600"
        loading={loading}
      />
      <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
        <div className="mb-4 flex items-center gap-2">
          <div className="rounded-full bg-slate-100 p-2 text-slate-700">
            <BarChart3 className="h-4 w-4" />
          </div>
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
              Source Mix
            </p>
            <p className="text-sm text-slate-700">Manual, STIX, OSINT, internal, vendor</p>
          </div>
        </div>

        <div className="space-y-3">
          {loading ? (
            Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="space-y-1.5">
                <div className="h-3 w-24 animate-pulse rounded bg-slate-200" />
                <div className="h-2.5 animate-pulse rounded-full bg-slate-200" />
              </div>
            ))
          ) : sourceBreakdown.length > 0 ? (
            sourceBreakdown.map((item) => (
              <div key={item.name} className="space-y-1.5">
                <div className="flex items-center justify-between text-xs text-slate-600">
                  <span>{item.label}</span>
                  <span className="font-medium text-slate-900">{item.count}</span>
                </div>
                <div className="h-2 rounded-full bg-slate-100">
                  <div
                    className={cn('h-full rounded-full bg-gradient-to-r from-emerald-500 to-cyan-500')}
                    style={{ width: `${Math.max(item.percent, 6)}%` }}
                  />
                </div>
              </div>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">No source telemetry yet.</p>
          )}
        </div>
      </div>
    </div>
  );
}
