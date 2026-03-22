'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { type ColumnDef } from '@tanstack/react-table';
import { LineChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { Edit, PlayCircle, Plus, TrendingUp, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { useDataTable } from '@/hooks/use-data-table';
import { enterpriseApi } from '@/lib/enterprise';
import { SectionCard } from '@/components/suites/section-card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { KpiCard } from '@/components/shared/kpi-card';
import { showApiError, showSuccess } from '@/lib/toast';
import type { FilterConfig, RowAction } from '@/types/table';
import type { VisusKPIDefinition } from '@/types/suites';
import { KpiFormDialog } from './_components/kpi-form-dialog';

const KPI_FILTERS: FilterConfig[] = [
  {
    key: 'suite',
    label: 'Suite',
    type: 'select',
    options: [
      { label: 'Cyber', value: 'cyber' },
      { label: 'Data', value: 'data' },
      { label: 'Acta', value: 'acta' },
      { label: 'Lex', value: 'lex' },
      { label: 'Platform', value: 'platform' },
      { label: 'Custom', value: 'custom' },
    ],
  },
  {
    key: 'enabled',
    label: 'Enabled',
    type: 'select',
    options: [
      { label: 'Enabled', value: 'true' },
      { label: 'Disabled', value: 'false' },
    ],
  },
];

export default function VisusKpisPage() {
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<VisusKPIDefinition | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VisusKPIDefinition | null>(null);

  const { tableProps, data, refetch } = useDataTable<VisusKPIDefinition>({
    queryKey: 'visus-kpis',
    fetchFn: (params) => enterpriseApi.visus.listKpis(params),
    defaultPageSize: 25,
    defaultSort: { column: 'name', direction: 'asc' },
  });

  const selected = selectedId ?? data[0]?.id ?? null;
  const detailQuery = useQuery({
    queryKey: ['visus-kpi-detail', selected],
    queryFn: () => enterpriseApi.visus.getKpi(selected!),
    enabled: Boolean(selected),
  });

  const createMutation = useMutation({
    mutationFn: enterpriseApi.visus.createKpi,
    onSuccess: async () => {
      showSuccess('KPI created.');
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['visus-kpis'] });
      refetch();
    },
    onError: showApiError,
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: unknown }) => enterpriseApi.visus.updateKpi(id, payload),
    onSuccess: async () => {
      showSuccess('KPI updated.');
      setEditTarget(null);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['visus-kpis'] }),
        queryClient.invalidateQueries({ queryKey: ['visus-kpi-detail', selected] }),
      ]);
      refetch();
    },
    onError: showApiError,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => enterpriseApi.visus.deleteKpi(id),
    onSuccess: async () => {
      showSuccess('KPI deleted.');
      if (deleteTarget?.id === selectedId) {
        setSelectedId(null);
      }
      setDeleteTarget(null);
      await queryClient.invalidateQueries({ queryKey: ['visus-kpis'] });
      refetch();
    },
    onError: showApiError,
  });

  const snapshotMutation = useMutation({
    mutationFn: () => enterpriseApi.visus.triggerKpiSnapshot(),
    onSuccess: async () => {
      showSuccess('Snapshot refresh started.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['visus-kpis'] }),
        queryClient.invalidateQueries({ queryKey: ['visus-kpi-detail', selected] }),
      ]);
      refetch();
    },
    onError: showApiError,
  });

  const columns: ColumnDef<VisusKPIDefinition>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'KPI',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs text-muted-foreground">{row.original.description}</p>
        </div>
      ),
    },
    {
      id: 'suite',
      accessorKey: 'suite',
      header: 'Suite',
      enableSorting: true,
      cell: ({ row }) => <Badge variant="outline">{row.original.suite}</Badge>,
    },
    {
      id: 'last_value',
      accessorKey: 'last_value',
      header: 'Latest',
      enableSorting: true,
      cell: ({ row }) => <span className="text-sm">{row.original.last_value ?? '—'}</span>,
    },
    {
      id: 'last_status',
      accessorKey: 'last_status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <Badge variant={statusVariant(row.original.last_status)}>{row.original.last_status ?? 'unknown'}</Badge>,
    },
  ];

  const rowActions: RowAction<VisusKPIDefinition>[] = [
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

  const history = detailQuery.data?.history ?? [];
  const definition = detailQuery.data?.definition;

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader
          title="KPIs"
          description="Executive KPI catalogue, thresholds, and collection configuration."
          actions={
            <>
              <Button variant="outline" size="sm" onClick={() => snapshotMutation.mutate()} disabled={snapshotMutation.isPending}>
                <PlayCircle className="mr-2 h-4 w-4" />
                {snapshotMutation.isPending ? 'Refreshing...' : 'Run Snapshot Refresh'}
              </Button>
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create KPI
              </Button>
            </>
          }
        />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.1fr_0.9fr]">
          <DataTable
            {...tableProps}
            columns={columns}
            filters={KPI_FILTERS}
            rowActions={rowActions}
            onRowClick={(row) => setSelectedId(row.id)}
            emptyState={{
              icon: TrendingUp,
              title: 'No KPIs found',
              description: 'No KPI definitions are configured for this tenant.',
            }}
          />
          <SectionCard title={definition?.name ?? 'KPI detail'} description={definition?.description ?? 'Select a KPI to inspect its latest history.'}>
            {definition ? (
              <div className="space-y-4">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <KpiCard title="Latest Value" value={definition.last_value ?? 0} />
                  <KpiCard title="Target" value={definition.target_value ?? '—'} />
                </div>
                <div className="rounded-xl border p-4">
                  <p className="mb-3 text-sm font-medium">History</p>
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={history.map((point) => ({ at: point.created_at.slice(5, 10), value: point.value }))}>
                        <XAxis dataKey="at" />
                        <YAxis />
                        <Tooltip />
                        <Line type="monotone" dataKey="value" stroke="#0f766e" strokeWidth={2} dot={false} />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Select a KPI to inspect its current status.</p>
            )}
          </SectionCard>
        </div>
      </div>

      <KpiFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        pending={createMutation.isPending}
        onSubmit={async (payload) => {
          await createMutation.mutateAsync(payload);
        }}
      />

      <KpiFormDialog
        open={Boolean(editTarget)}
        onOpenChange={(open) => {
          if (!open) setEditTarget(null);
        }}
        kpi={editTarget}
        pending={updateMutation.isPending}
        onSubmit={async (payload) => {
          if (!editTarget) return;
          await updateMutation.mutateAsync({ id: editTarget.id, payload });
        }}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title="Delete KPI"
        description={`Delete "${deleteTarget?.name}"? Historical snapshots will remain, but the KPI definition will no longer be available for widgets or executive rollups.`}
        confirmLabel="Delete KPI"
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

function statusVariant(status: string | null | undefined): 'default' | 'warning' | 'destructive' | 'outline' {
  if (status === 'warning') return 'warning';
  if (status === 'critical') return 'destructive';
  if (status === 'normal') return 'default';
  return 'outline';
}
