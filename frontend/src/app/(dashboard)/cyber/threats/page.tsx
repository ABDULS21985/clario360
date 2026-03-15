'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Plus, Search, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PermissionGate } from '@/components/auth/permission-gate';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { PieChart } from '@/components/shared/charts/pie-chart';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { SEVERITY_COLORS } from '@/lib/cyber-threats';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, FilterConfig } from '@/types/table';
import type { NamedCount, Threat, ThreatStats, ThreatTrendPoint } from '@/types/cyber';

import { getThreatColumns } from './_components/threat-columns';
import { IndicatorCheckDialog } from './_components/indicator-check-dialog';
import { CreateThreatDialog } from './_components/create-threat-dialog';

const THREAT_FILTERS: FilterConfig[] = [
  {
    key: 'severity',
    label: 'Severity',
    type: 'multi-select',
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select',
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Contained', value: 'contained' },
      { label: 'Eradicated', value: 'eradicated' },
      { label: 'Monitoring', value: 'monitoring' },
      { label: 'Closed', value: 'closed' },
    ],
  },
];

function fetchThreats(params: FetchParams): Promise<PaginatedResponse<Threat>> {
  return apiGet<PaginatedResponse<Threat>>(API_ENDPOINTS.CYBER_THREATS, flattenParams(params));
}

export default function CyberThreatsPage() {
  const router = useRouter();
  const [indicatorCheckOpen, setIndicatorCheckOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);

  const { tableProps, refetch } = useDataTable<Threat>({
    fetchFn: fetchThreats,
    queryKey: 'cyber-threats',
    defaultPageSize: 25,
    defaultSort: { column: 'last_seen_at', direction: 'desc' },
    wsTopics: ['cyber.threat.detected', 'cyber.threat.updated'],
  });

  const statsQuery = useQuery({
    queryKey: ['cyber-threat-stats'],
    queryFn: () => apiGet<{ data: ThreatStats }>(API_ENDPOINTS.CYBER_THREAT_STATS),
  });
  const trendQuery = useQuery({
    queryKey: ['cyber-threat-trend'],
    queryFn: () => apiGet<{ data: ThreatTrendPoint[] }>(API_ENDPOINTS.CYBER_THREAT_STATS_TREND),
  });

  const stats = statsQuery.data?.data;
  const trendPoints = trendQuery.data?.data ?? [];
  const activeDelta = useMemo(
    () => computePercentDelta(trendPoints.map((point) => point.active)),
    [trendPoints],
  );

  const columns = useMemo(
    () => getThreatColumns({
      onViewDetail: (threat) => router.push(`${ROUTES.CYBER_THREATS}/${threat.id}`),
    }),
    [router],
  );

  const criticalHighCount = sumCounts(
    stats?.by_severity ?? [],
    ['critical', 'high'],
  );

  const byTypeChart = (stats?.by_type ?? []).map((entry) => ({
    name: titleizeCount(entry.name),
    count: entry.count,
  }));
  const bySeverityChart = (stats?.by_severity ?? []).map((entry) => ({
    name: titleizeCount(entry.name),
    value: entry.count,
    color: SEVERITY_COLORS[entry.name] ?? '#6B7280',
  }));

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Threat Intelligence"
          description="Track active threats, manage their lifecycle, and pivot from threat campaigns into indicators and related alerts."
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => setIndicatorCheckOpen(true)}>
                <Search className="mr-1.5 h-3.5 w-3.5" />
                Check Indicators
              </Button>
              <PermissionGate permission="cyber:write">
                <Button size="sm" onClick={() => setCreateOpen(true)}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  New Threat
                </Button>
              </PermissionGate>
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <KpiCard
            title="Active Threats"
            value={stats?.active ?? 0}
            change={activeDelta}
            changeLabel="vs 7d"
            loading={statsQuery.isLoading || trendQuery.isLoading}
          />
          <KpiCard
            title="Critical / High"
            value={criticalHighCount}
            iconColor="text-red-600"
            loading={statsQuery.isLoading}
          />
          <KpiCard
            title="IOCs Tracked"
            value={stats?.indicators_total ?? 0}
            iconColor="text-blue-600"
            loading={statsQuery.isLoading}
          />
          <KpiCard
            title="Contained This Month"
            value={stats?.contained_this_month ?? 0}
            iconColor="text-green-600"
            loading={statsQuery.isLoading}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <BarChart
            title="Threats by Type"
            data={byTypeChart}
            xKey="name"
            yKeys={[{ key: 'count', label: 'Threats', color: '#0F766E' }]}
            loading={statsQuery.isLoading}
            error={statsQuery.error instanceof Error ? statsQuery.error.message : undefined}
            onRetry={() => void statsQuery.refetch()}
            height={280}
            showLegend={false}
          />
          <PieChart
            title="Threats by Severity"
            data={bySeverityChart}
            loading={statsQuery.isLoading}
            error={statsQuery.error instanceof Error ? statsQuery.error.message : undefined}
            onRetry={() => void statsQuery.refetch()}
            centerLabel="threats"
            centerValue={String(stats?.total ?? 0)}
            height={280}
          />
        </div>

        <DataTable
          columns={columns}
          filters={THREAT_FILTERS}
          searchPlaceholder="Search threats…"
          emptyState={{
            icon: Shield,
            title: 'No threats found',
            description: 'No threats match the current filters.',
          }}
          getRowId={(row) => row.id}
          onRowClick={(row) => router.push(`${ROUTES.CYBER_THREATS}/${row.id}`)}
          {...tableProps}
        />
      </div>

      <CreateThreatDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={(threat) => {
          void statsQuery.refetch();
          void trendQuery.refetch();
          refetch();
          router.push(`${ROUTES.CYBER_THREATS}/${threat.id}`);
        }}
      />
      <IndicatorCheckDialog
        open={indicatorCheckOpen}
        onOpenChange={setIndicatorCheckOpen}
      />
    </PermissionRedirect>
  );
}

function flattenParams(params: FetchParams): Record<string, unknown> {
  const flat: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
  };

  for (const [key, value] of Object.entries(params.filters ?? {})) {
    flat[key] = value;
  }

  return flat;
}

function sumCounts(items: NamedCount[], names: string[]): number {
  const set = new Set(names);
  return items.reduce((total, item) => (
    set.has(item.name) ? total + item.count : total
  ), 0);
}

function computePercentDelta(values: number[]): number {
  if (values.length < 8) {
    return 0;
  }
  const current = values[values.length - 1] ?? 0;
  const prior = values[values.length - 8] ?? 0;
  if (prior === 0) {
    return current === 0 ? 0 : 100;
  }
  return ((current - prior) / prior) * 100;
}

function titleizeCount(value: string): string {
  return value
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}
