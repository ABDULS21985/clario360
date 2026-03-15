'use client';

import { ShieldCheck, Activity, Radar, Target } from 'lucide-react';

import { KpiCard } from '@/components/shared/kpi-card';
import type { DetectionRuleStats } from '@/types/cyber';

interface RuleStatsProps {
  stats?: DetectionRuleStats;
  loading?: boolean;
}

function countFor(items: DetectionRuleStats['by_type'], name: string): number {
  return items.find((item) => item.name === name)?.count ?? 0;
}

export function RuleStats({ stats, loading = false }: RuleStatsProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
      <KpiCard
        title="Total Rules"
        value={stats?.total ?? 0}
        icon={ShieldCheck}
        iconColor="text-sky-700"
        description="All tenant-scoped detection rules."
        loading={loading}
      />
      <KpiCard
        title="Active Rules"
        value={stats?.active ?? 0}
        icon={Activity}
        iconColor="text-emerald-700"
        change={stats ? (stats.total > 0 ? (stats.active / stats.total) * 100 : 0) : 0}
        changeLabel="enabled"
        loading={loading}
      />
      <KpiCard
        title="Type Mix"
        value={
          stats
            ? `${countFor(stats.by_type, 'sigma')}/${countFor(stats.by_type, 'threshold')}/${countFor(stats.by_type, 'correlation')}/${countFor(stats.by_type, 'anomaly')}`
            : '0/0/0/0'
        }
        icon={Radar}
        iconColor="text-amber-700"
        description="Sigma / Threshold / Correlation / Anomaly"
        loading={loading}
      />
      <KpiCard
        title="True Positive Rate"
        value={`${((stats?.true_positive_rate ?? 0) * 100).toFixed(1)}%`}
        icon={Target}
        iconColor="text-orange-700"
        description={`${stats?.alerts_last_30_days ?? 0} alerts in the last 30 days`}
        loading={loading}
      />
    </div>
  );
}
