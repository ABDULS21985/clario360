'use client';

import { useCallback, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Activity, Copy, FileJson } from 'lucide-react';
import { useState } from 'react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { PieChart } from '@/components/shared/charts/pie-chart';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { useDataTable } from '@/hooks/use-data-table';
import api, { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { downloadBlob } from '@/lib/format';
import { showSuccess, showError } from '@/lib/toast';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, FilterConfig, RowAction } from '@/types/table';
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
    key: 'time_range',
    label: 'Time Range',
    type: 'date-range',
  },
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
  {
    key: 'source',
    label: 'Source',
    type: 'text',
    placeholder: 'e.g. firewall, endpoint…',
  },
  {
    key: 'type',
    label: 'Event Type',
    type: 'text',
    placeholder: 'e.g. connection_attempt…',
  },
  {
    key: 'source_ip',
    label: 'Source IP',
    type: 'text',
    placeholder: 'e.g. 192.168.1.1',
  },
  {
    key: 'dest_ip',
    label: 'Dest IP',
    type: 'text',
    placeholder: 'e.g. 10.0.0.1',
  },
  {
    key: 'username',
    label: 'Username',
    type: 'text',
    placeholder: 'e.g. jsmith',
  },
  {
    key: 'process',
    label: 'Process',
    type: 'text',
    placeholder: 'e.g. powershell.exe',
  },
  {
    key: 'cmd_contains',
    label: 'Command',
    type: 'text',
    placeholder: 'Command line substring…',
  },
  {
    key: 'file_hash',
    label: 'File Hash',
    type: 'text',
    placeholder: 'SHA256 or MD5…',
  },
  {
    key: 'matched_rule',
    label: 'Rule ID',
    type: 'text',
    placeholder: 'Rule UUID…',
  },
];

function flattenParams(params: FetchParams): Record<string, unknown> {
  const flat: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
  };
  for (const [key, value] of Object.entries(params.filters ?? {})) {
    if (key === 'time_range' && typeof value === 'string') {
      // date-range filter stores as "ISO_FROM,ISO_TO"
      const [from, to] = value.split(',');
      if (from) flat.from = from;
      if (to) flat.to = to;
    } else {
      flat[key] = value;
    }
  }
  return flat;
}

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

  // Sync stats time range with the active time_range filter
  const activeTimeRange = tableProps.activeFilters?.['time_range'];
  const [statsFrom, statsTo] =
    typeof activeTimeRange === 'string' ? activeTimeRange.split(',') : [];

  const statsQuery = useQuery({
    queryKey: ['cyber-event-stats', statsFrom, statsTo],
    queryFn: () => {
      const params: Record<string, string> = {};
      if (statsFrom) params.from = statsFrom;
      if (statsTo) params.to = statsTo;
      return apiGet<{ data: EventStats }>(API_ENDPOINTS.CYBER_EVENT_STATS, params);
    },
    refetchInterval: 60000,
  });

  const stats = statsQuery.data?.data;
  const columns = useMemo(() => getEventColumns(), []);

  const rowActions = useMemo<RowAction<SecurityEvent>[]>(
    () => [
      {
        label: 'Copy ID',
        icon: Copy,
        onClick: (row) => {
          navigator.clipboard.writeText(row.id);
          showSuccess('Event ID copied');
        },
      },
      {
        label: 'Copy Raw JSON',
        icon: FileJson,
        onClick: (row) => {
          navigator.clipboard.writeText(JSON.stringify(row.raw_event, null, 2));
          showSuccess('Raw JSON copied');
        },
      },
    ],
    [],
  );

  const handleExport = useCallback(
    async (format: 'csv' | 'json') => {
      const serverFormat = format === 'json' ? 'ndjson' : 'csv';
      const ext = format === 'json' ? 'ndjson' : 'csv';

      // Build query params from active filters
      const exportParams: Record<string, unknown> = {};
      const filters = tableProps.activeFilters ?? {};
      for (const [key, value] of Object.entries(filters)) {
        if (key === 'time_range' && typeof value === 'string') {
          const [from, to] = value.split(',');
          if (from) exportParams.from = from;
          if (to) exportParams.to = to;
        } else {
          exportParams[key] = value;
        }
      }

      // Server-side export requires 'from'; default to 30 days ago if not set
      if (!exportParams.from) {
        const thirtyDaysAgo = new Date();
        thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
        exportParams.from = thirtyDaysAgo.toISOString();
      }

      if (tableProps.searchValue) exportParams.search = tableProps.searchValue;
      if (tableProps.sortColumn) exportParams.sort = tableProps.sortColumn;
      if (tableProps.sortDirection) exportParams.order = tableProps.sortDirection;
      exportParams.format = serverFormat;

      try {
        const response = await api.get(API_ENDPOINTS.CYBER_EVENTS_EXPORT, {
          params: exportParams,
          responseType: 'blob',
        });
        const filename = `cyber-events-${new Date().toISOString().slice(0, 10)}.${ext}`;
        downloadBlob(response.data as Blob, filename);
        showSuccess('Export downloaded');
      } catch {
        showError('Export failed', 'Unable to download the export file.');
      }
    },
    [tableProps.activeFilters, tableProps.searchValue, tableProps.sortColumn, tableProps.sortDirection],
  );

  const bySeverityChart = (stats?.by_severity ?? []).map((entry) => ({
    name: entry.name.charAt(0).toUpperCase() + entry.name.slice(1),
    value: entry.count,
    color: SEVERITY_COLORS[entry.name] ?? '#6B7280',
  }));

  const bySourceChart = (stats?.by_source ?? []).slice(0, 10).map((entry) => ({
    name: entry.name,
    count: entry.count,
  }));

  const byTypeChart = (stats?.by_type ?? []).slice(0, 8).map((entry) => ({
    name: entry.name.replace(/_/g, ' '),
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
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <BarChart
            title="Events by Source"
            data={bySourceChart}
            xKey="name"
            yKeys={[{ key: 'count', label: 'Events', color: '#0F766E' }]}
            loading={statsQuery.isLoading}
            height={240}
            showLegend={false}
          />
          <BarChart
            title="Events by Type"
            data={byTypeChart}
            xKey="name"
            yKeys={[{ key: 'count', label: 'Events', color: '#1B5E20' }]}
            loading={statsQuery.isLoading}
            height={240}
            showLegend={false}
          />
          <PieChart
            title="Events by Severity"
            data={bySeverityChart}
            loading={statsQuery.isLoading}
            centerLabel="events"
            centerValue={String(stats?.total ?? 0)}
            height={240}
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
          rowActions={rowActions}
          enableExport
          onExport={handleExport}
          enableColumnToggle
          defaultHiddenColumns={['parent_process', 'command_line', 'file_hash', 'asset_id']}
          compact
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
