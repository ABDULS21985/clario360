'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Activity } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { PieChart } from '@/components/shared/charts/pie-chart';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, FilterConfig } from '@/types/table';
import type { SecurityEvent, EventStats } from '@/types/cyber';

import { getEventColumns } from './_components/event-columns';
import { EventDetailPanel } from './_components/event-detail-panel';

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#DC2626',
  high: '#F97316',
  medium: '#EAB308',
  low: '#3B82F6',
  info: '#6B7280',
};

const EVENT_FILTERS: FilterConfig[] = [
  {
    key: 'severity',
    label: 'Severity',
    type: 'multi-select',
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
      { label: 'Info', value: 'info' },
    ],
  },
  {
    key: 'protocol',
    label: 'Protocol',
    type: 'multi-select',
    options: [
      { label: 'TCP', value: 'TCP' },
      { label: 'UDP', value: 'UDP' },
      { label: 'ICMP', value: 'ICMP' },
      { label: 'HTTP', value: 'HTTP' },
      { label: 'DNS', value: 'DNS' },
    ],
  },
];

function fetchEvents(params: FetchParams): Promise<PaginatedResponse<SecurityEvent>> {
  return apiGet<PaginatedResponse<SecurityEvent>>(API_ENDPOINTS.CYBER_EVENTS, flattenParams(params));
}

export default function CyberEventsPage() {
  const [selectedEvent, setSelectedEvent] = useState<SecurityEvent | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const { tableProps } = useDataTable<SecurityEvent>({
    fetchFn: fetchEvents,
    queryKey: 'cyber-events',
    defaultPageSize: 50,
    defaultSort: { column: 'timestamp', direction: 'desc' },
  });

  const statsQuery = useQuery({
    queryKey: ['cyber-event-stats'],
    queryFn: () => apiGet<{ data: EventStats }>(API_ENDPOINTS.CYBER_EVENT_STATS),
    refetchInterval: 60000,
  });

  const stats = statsQuery.data?.data;
  const columns = useMemo(() => getEventColumns(), []);

  const bySeverityChart = (stats?.by_severity ?? []).map((entry) => ({
    name: entry.name.charAt(0).toUpperCase() + entry.name.slice(1),
    value: entry.count,
    color: SEVERITY_COLORS[entry.name] ?? '#6B7280',
  }));

  const bySourceChart = (stats?.by_source ?? []).slice(0, 10).map((entry) => ({
    name: entry.name,
    count: entry.count,
  }));

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Event Explorer"
          description="Search and analyze security events across all log sources — the SIEM log viewer for incident investigations."
        />

        {/* KPI Row */}
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <KpiCard
            title="Total Events"
            value={stats?.total ?? 0}
            loading={statsQuery.isLoading}
          />
          <KpiCard
            title="Sources"
            value={stats?.by_source?.length ?? 0}
            loading={statsQuery.isLoading}
          />
          <KpiCard
            title="Event Types"
            value={stats?.by_type?.length ?? 0}
            loading={statsQuery.isLoading}
          />
          <KpiCard
            title="Critical / High"
            value={
              (stats?.by_severity ?? [])
                .filter((s) => s.name === 'critical' || s.name === 'high')
                .reduce((sum, s) => sum + s.count, 0)
            }
            iconColor="text-red-600"
            loading={statsQuery.isLoading}
          />
        </div>

        {/* Charts Row */}
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <BarChart
            title="Events by Source"
            data={bySourceChart}
            xKey="name"
            yKeys={[{ key: 'count', label: 'Events', color: '#0F766E' }]}
            loading={statsQuery.isLoading}
            height={260}
            showLegend={false}
          />
          <PieChart
            title="Events by Severity"
            data={bySeverityChart}
            loading={statsQuery.isLoading}
            centerLabel="events"
            centerValue={String(stats?.total ?? 0)}
            height={260}
          />
        </div>

        {/* Events Table */}
        <DataTable
          columns={columns}
          filters={EVENT_FILTERS}
          searchPlaceholder="Search events (IP, process, command, text)…"
          emptyState={{
            icon: Activity,
            title: 'No events found',
            description: 'No security events match the current filters.',
          }}
          getRowId={(row) => row.id}
          onRowClick={(row) => {
            setSelectedEvent(row);
            setDetailOpen(true);
          }}
          {...tableProps}
        />
      </div>

      <EventDetailPanel
        event={selectedEvent}
        open={detailOpen}
        onOpenChange={setDetailOpen}
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
