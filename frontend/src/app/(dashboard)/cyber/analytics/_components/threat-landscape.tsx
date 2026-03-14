'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { KpiCard } from '@/components/shared/kpi-card';
import { PieChart } from '@/components/shared/charts/pie-chart';
import type { AnalyticsLandscape } from '@/types/cyber';

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#DC2626',
  high: '#F97316',
  medium: '#EAB308',
  low: '#3B82F6',
  info: '#6B7280',
};

const TYPE_COLORS = ['#3B82F6', '#8B5CF6', '#EC4899', '#F97316', '#10B981', '#6366F1', '#EAB308', '#14B8A6'];

export function ThreatLandscape() {
  const { data, isLoading } = useQuery({
    queryKey: ['cyber-analytics-landscape'],
    queryFn: () => apiGet<{ data: AnalyticsLandscape }>(API_ENDPOINTS.CYBER_ANALYTICS_LANDSCAPE),
    refetchInterval: 120000,
  });

  const landscape = data?.data;

  const byTypeChart = (landscape?.by_type ?? []).map((entry, i) => ({
    name: entry.name.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
    value: entry.count,
    color: TYPE_COLORS[i % TYPE_COLORS.length],
  }));
  const bySevChart = (landscape?.by_severity ?? []).map((entry) => ({
    name: entry.name.charAt(0).toUpperCase() + entry.name.slice(1),
    value: entry.count,
    color: SEVERITY_COLORS[entry.name] ?? '#6B7280',
  }));

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold">Threat Landscape</h3>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <KpiCard
          title="Active Threats"
          value={landscape?.active_threat_count ?? 0}
          iconColor="text-red-600"
          loading={isLoading}
        />
        <KpiCard
          title="Total IOCs"
          value={landscape?.indicators_total ?? 0}
          iconColor="text-blue-600"
          loading={isLoading}
        />
        <KpiCard
          title="Top Threat Type"
          value={landscape?.top_threat_type?.replace(/_/g, ' ') ?? '—'}
          loading={isLoading}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <PieChart
          title="Threats by Type"
          data={byTypeChart}
          loading={isLoading}
          height={280}
          centerLabel="types"
          centerValue={String(landscape?.by_type?.length ?? 0)}
        />
        <PieChart
          title="Threats by Severity"
          data={bySevChart}
          loading={isLoading}
          height={280}
          centerLabel="threats"
          centerValue={String(landscape?.total_threats ?? 0)}
        />
      </div>
    </div>
  );
}
