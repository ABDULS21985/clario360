'use client';

import { useState } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import { FileBarChart, PlayCircle } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { DataTable } from '@/components/shared/data-table/data-table';
import { Button } from '@/components/ui/button';
import { useDataTable } from '@/hooks/use-data-table';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated, type SuiteEnvelope } from '@/lib/suite-api';
import { showApiError, showSuccess } from '@/lib/toast';
import type { VisusReport, VisusReportGeneration } from '@/types/suites';

export default function VisusReportsPage() {
  const [runningId, setRunningId] = useState<string | null>(null);

  const { tableProps, refetch } = useDataTable<VisusReport>({
    queryKey: 'visus-reports',
    fetchFn: (params) => fetchSuitePaginated<VisusReport>(API_ENDPOINTS.VISUS_REPORTS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const generateReport = async (report: VisusReport) => {
    try {
      setRunningId(report.id);
      const response = await apiPost<SuiteEnvelope<VisusReportGeneration>>(
        `${API_ENDPOINTS.VISUS_REPORTS}/${report.id}/generate`,
      );
      showSuccess('Report generation started.', `Snapshot ${response.data.snapshot_id.slice(0, 8)} queued for ${report.name}.`);
      refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setRunningId(null);
    }
  };

  const columns: ColumnDef<VisusReport>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Report',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs capitalize text-muted-foreground">{row.original.type.replace(/_/g, ' ')}</p>
        </div>
      ),
    },
    {
      id: 'schedule',
      accessorKey: 'schedule',
      header: 'Schedule',
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{row.original.schedule ?? 'On demand'}</span>,
    },
    {
      id: 'last_generated_at',
      accessorKey: 'last_generated_at',
      header: 'Last Generated',
      enableSorting: true,
      cell: ({ row }) =>
        row.original.last_generated_at ? (
          <RelativeTime date={row.original.last_generated_at} />
        ) : (
          <span className="text-sm text-muted-foreground">Never</span>
        ),
    },
    {
      id: 'file_url',
      accessorKey: 'file_url',
      header: 'Output',
      cell: ({ row }) =>
        row.original.file_url ? (
          <a href={row.original.file_url} className="text-sm text-primary hover:underline" target="_blank" rel="noreferrer">
            Latest output
          </a>
        ) : (
          <span className="text-sm text-muted-foreground">None</span>
        ),
    },
    {
      id: 'generate',
      header: '',
      cell: ({ row }) => (
        <Button
          variant="outline"
          size="sm"
          onClick={() => void generateReport(row.original)}
          disabled={runningId === row.original.id}
        >
          <PlayCircle className="mr-1.5 h-3.5 w-3.5" />
          {runningId === row.original.id ? 'Generating…' : 'Generate'}
        </Button>
      ),
    },
  ];

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader title="Reports" description="Executive reports, schedules, and on-demand generation triggers." />
        <DataTable
          {...tableProps}
          columns={columns}
          emptyState={{
            icon: FileBarChart,
            title: 'No reports found',
            description: 'No executive reports are configured for this tenant.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
