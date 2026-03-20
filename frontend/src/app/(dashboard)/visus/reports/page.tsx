'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { type ColumnDef } from '@tanstack/react-table';
import { Edit, Eye, FileBarChart, PlayCircle, Plus, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { DataTable } from '@/components/shared/data-table/data-table';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { useDataTable } from '@/hooks/use-data-table';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { FilterConfig, RowAction } from '@/types/table';
import type { UserDirectoryEntry, VisusReport, VisusReportGeneration } from '@/types/suites';
import { ReportFormDialog } from './_components/report-form-dialog';
import { ReportSnapshotsDialog } from './_components/report-snapshots-dialog';

const REPORT_FILTERS: FilterConfig[] = [
  {
    key: 'report_type',
    label: 'Type',
    type: 'select',
    options: [
      { label: 'Executive Summary', value: 'executive_summary' },
      { label: 'Security Posture', value: 'security_posture' },
      { label: 'Data Intelligence', value: 'data_intelligence' },
      { label: 'Governance', value: 'governance' },
      { label: 'Legal', value: 'legal' },
      { label: 'Custom', value: 'custom' },
    ],
  },
  {
    key: 'auto_send',
    label: 'Auto Send',
    type: 'select',
    options: [
      { label: 'Enabled', value: 'true' },
      { label: 'Disabled', value: 'false' },
    ],
  },
];

export default function VisusReportsPage() {
  const queryClient = useQueryClient();
  const [runningId, setRunningId] = useState<string | null>(null);
  const [previewReportId, setPreviewReportId] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<VisusReport | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VisusReport | null>(null);

  const { tableProps, refetch } = useDataTable<VisusReport>({
    queryKey: 'visus-reports',
    fetchFn: (params) => enterpriseApi.visus.listReports(params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const usersQuery = useQuery({
    queryKey: ['visus-report-users'],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, sort: 'first_name', order: 'asc' }),
  });

  const createMutation = useMutation({
    mutationFn: enterpriseApi.visus.createReport,
    onSuccess: async () => {
      showSuccess('Report created.');
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['visus-reports'] });
      refetch();
    },
    onError: showApiError,
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: unknown }) => enterpriseApi.visus.updateReport(id, payload),
    onSuccess: async () => {
      showSuccess('Report updated.');
      setEditTarget(null);
      await queryClient.invalidateQueries({ queryKey: ['visus-reports'] });
      refetch();
    },
    onError: showApiError,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => enterpriseApi.visus.deleteReport(id),
    onSuccess: async () => {
      showSuccess('Report deleted.');
      setDeleteTarget(null);
      await queryClient.invalidateQueries({ queryKey: ['visus-reports'] });
      refetch();
    },
    onError: showApiError,
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

  const rowActions: RowAction<VisusReport>[] = [
    {
      label: 'View snapshots',
      icon: Eye,
      onClick: (row) => setPreviewReportId(row.id),
    },
    {
      label: 'Edit',
      icon: Edit,
      onClick: (row) => setEditTarget(row),
    },
    {
      label: 'Delete',
      icon: Trash2,
      variant: 'destructive',
      onClick: (row) => setDeleteTarget(row),
    },
  ];

  const columns: ColumnDef<VisusReport>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Report',
      enableSorting: true,
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
      enableSorting: true,
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          {row.original.last_generated_at ? (
            <>
              <RelativeTime date={row.original.last_generated_at} />
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={(event) => { event.stopPropagation(); setPreviewReportId(row.original.id); }}>
                <Eye className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : (
            <span className="text-sm text-muted-foreground">Never</span>
          )}
        </div>
      ),
    },
    {
      id: 'total_generated',
      accessorKey: 'total_generated',
      header: 'Generations',
      cell: ({ row }) => <span className="text-sm">{row.original.total_generated}</span>,
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
          {runningId === row.original.id ? 'Generating...' : 'Generate'}
        </Button>
      ),
    },
  ];

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader
          title="Reports"
          description="Executive report definitions, delivery schedules, and historical output snapshots."
          actions={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Report
            </Button>
          }
        />
        <DataTable
          {...tableProps}
          columns={columns}
          filters={REPORT_FILTERS}
          rowActions={rowActions}
          emptyState={{
            icon: FileBarChart,
            title: 'No reports found',
            description: 'No executive reports are configured for this tenant.',
          }}
        />
      </div>

      <ReportFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        users={(usersQuery.data?.data ?? []) as UserDirectoryEntry[]}
        pending={createMutation.isPending}
        onSubmit={async (payload) => {
          await createMutation.mutateAsync(payload);
        }}
      />

      <ReportFormDialog
        open={Boolean(editTarget)}
        onOpenChange={(open) => {
          if (!open) setEditTarget(null);
        }}
        report={editTarget}
        users={(usersQuery.data?.data ?? []) as UserDirectoryEntry[]}
        pending={updateMutation.isPending}
        onSubmit={async (payload) => {
          if (!editTarget) return;
          await updateMutation.mutateAsync({ id: editTarget.id, payload });
        }}
      />

      <ReportSnapshotsDialog
        reportId={previewReportId}
        open={previewReportId !== null}
        onOpenChange={(open) => {
          if (!open) setPreviewReportId(null);
        }}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title="Delete Report"
        description={`Delete "${deleteTarget?.name}"? Existing snapshots remain historical records, but future generation for this definition will stop.`}
        confirmLabel="Delete Report"
        variant="destructive"
        loading={deleteMutation.isPending}
        onConfirm={async () => {
          if (!deleteTarget) return;
          await deleteMutation.mutateAsync(deleteTarget.id);
        }}
      />
    </PermissionRedirect>
  );
}
