'use client';

import { useState } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import { PlayCircle, GitBranch } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { Button } from '@/components/ui/button';
import { useDataTable } from '@/hooks/use-data-table';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated, type SuiteEnvelope } from '@/lib/suite-api';
import { pipelineStatusConfig } from '@/lib/status-configs';
import { showApiError, showSuccess } from '@/lib/toast';
import { formatDateTime } from '@/lib/utils';
import type { DataPipeline, DataPipelineRun } from '@/types/suites';

const PIPELINE_FILTERS = [
  {
    key: 'type',
    label: 'Type',
    type: 'select' as const,
    options: [
      { label: 'ETL', value: 'etl' },
      { label: 'ELT', value: 'elt' },
      { label: 'Streaming', value: 'streaming' },
      { label: 'Batch', value: 'batch' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Paused', value: 'paused' },
      { label: 'Failed', value: 'failed' },
      { label: 'Completed', value: 'completed' },
    ],
  },
];

export default function DataPipelinesPage() {
  const [runningId, setRunningId] = useState<string | null>(null);

  const { tableProps, searchValue, setSearch, refetch } = useDataTable<DataPipeline>({
    queryKey: 'data-pipelines',
    fetchFn: (params) => fetchSuitePaginated<DataPipeline>(API_ENDPOINTS.DATA_PIPELINES, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const runPipeline = async (pipeline: DataPipeline) => {
    try {
      setRunningId(pipeline.id);
      const response = await apiPost<SuiteEnvelope<DataPipelineRun>>(
        `${API_ENDPOINTS.DATA_PIPELINES}/${pipeline.id}/run`,
      );
      showSuccess('Pipeline run started.', `${pipeline.name} accepted a new run at ${formatDateTime(response.data.started_at)}.`);
      refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setRunningId(null);
    }
  };

  const columns: ColumnDef<DataPipeline>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Pipeline',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs text-muted-foreground">
            {row.original.source_name ?? 'Unknown source'} → {row.original.target_name ?? 'Unknown target'}
          </p>
        </div>
      ),
    },
    {
      id: 'type',
      accessorKey: 'type',
      header: 'Type',
      enableSorting: true,
      cell: ({ row }) => <span className="capitalize">{row.original.type}</span>,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={pipelineStatusConfig} size="sm" />,
    },
    {
      id: 'schedule',
      accessorKey: 'schedule',
      header: 'Schedule',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{row.original.schedule ?? 'On demand'}</span>
      ),
    },
    {
      id: 'last_run_status',
      accessorKey: 'last_run_status',
      header: 'Last Run',
      cell: ({ row }) => (
        <div className="text-sm text-muted-foreground">
          <p className="capitalize">{row.original.last_run_status ?? 'Never run'}</p>
          {row.original.last_run_at ? <RelativeTime date={row.original.last_run_at} /> : null}
        </div>
      ),
    },
    {
      id: 'throughput',
      header: 'Throughput',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {(row.original.last_run_records_processed ?? 0).toLocaleString()} / {(row.original.last_run_records_failed ?? 0).toLocaleString()}
        </span>
      ),
    },
    {
      id: 'run',
      header: '',
      cell: ({ row }) => (
        <Button
          variant="outline"
          size="sm"
          onClick={() => void runPipeline(row.original)}
          disabled={runningId === row.original.id}
        >
          <PlayCircle className="mr-1.5 h-3.5 w-3.5" />
          {runningId === row.original.id ? 'Starting…' : 'Run'}
        </Button>
      ),
    },
  ];

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Pipelines" description="Live operational view of ingestion, ETL, ELT, and streaming pipelines." />

        <DataTable
          {...tableProps}
          columns={columns}
          filters={PIPELINE_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search pipelines..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: GitBranch,
            title: 'No pipelines found',
            description: 'No pipelines matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
