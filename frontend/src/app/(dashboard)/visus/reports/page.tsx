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
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { VisusReport, VisusReportGeneration } from '@/types/suites';

export default function VisusReportsPage() {
  const [runningId, setRunningId] = useState<string | null>(null);

  const { tableProps, refetch } = useDataTable<VisusReport>({
    queryKey: 'visus-reports',
    fetchFn: (params) => enterpriseApi.visus.listReports(params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const generateReport = async (report: VisusReport) => {
    try {
      setRunningId(report.id);
      const response: VisusReportGeneration = await enterpriseApi.visus.generateReport(report.id);
      showSuccess('Report generation started.', `Snapshot ${response.id.slice(0, 8)} queued for ${report.name}.`);
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
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs capitalize text-muted-foreground">{(row.original.report_type ?? 'custom').replace(/_/g, ' ')}</p>
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
      cell: ({ row }) =>
        row.original.last_generated_at ? (
          <RelativeTime date={row.original.last_generated_at} />
        ) : (
          <span className="text-sm text-muted-foreground">Never</span>
        ),
    },
    {
      id: 'total_generated',
      accessorKey: 'total_generated',
      header: 'Generations',
      cell: ({ row }) => (
        <span className="text-sm">{row.original.total_generated}</span>
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
