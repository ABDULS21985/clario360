'use client';

import Link from 'next/link';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { type ColumnDef } from '@tanstack/react-table';
import { CopyPlus, LayoutDashboard, Pencil, Plus, Share2, Trash2 } from 'lucide-react';
import { useDataTable } from '@/hooks/use-data-table';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { DataTable } from '@/components/shared/data-table/data-table';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { FilterConfig, RowAction } from '@/types/table';
import type { UserDirectoryEntry, VisusDashboard } from '@/types/suites';
import { DashboardFormDialog } from './_components/dashboard-form-dialog';
import { DashboardShareDialog } from './_components/dashboard-share-dialog';

const DASHBOARD_FILTERS: FilterConfig[] = [
  {
    key: 'visibility',
    label: 'Visibility',
    type: 'select',
    options: [
      { label: 'Private', value: 'private' },
      { label: 'Team', value: 'team' },
      { label: 'Organization', value: 'organization' },
      { label: 'Public', value: 'public' },
    ],
  },
];

function visibilityVariant(value: VisusDashboard['visibility']): 'default' | 'secondary' | 'outline' {
  if (value === 'public') return 'default';
  if (value === 'organization') return 'secondary';
  return 'outline';
}

export default function VisusDashboardsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<VisusDashboard | null>(null);
  const [shareTarget, setShareTarget] = useState<VisusDashboard | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VisusDashboard | null>(null);

  const usersQuery = useQuery({
    queryKey: ['visus-dashboard-users'],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, sort: 'first_name', order: 'asc' }),
  });

  const { tableProps, refetch } = useDataTable<VisusDashboard>({
    queryKey: 'visus-dashboards',
    fetchFn: (params) => enterpriseApi.visus.listDashboards(params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const createMutation = useMutation({
    mutationFn: enterpriseApi.visus.createDashboard,
    onSuccess: async () => {
      showSuccess('Dashboard created.');
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboards'] });
      refetch();
    },
    onError: showApiError,
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: unknown }) => enterpriseApi.visus.updateDashboard(id, payload),
    onSuccess: async () => {
      showSuccess('Dashboard updated.');
      setEditTarget(null);
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboards'] });
      refetch();
    },
    onError: showApiError,
  });

  const shareMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: unknown }) => enterpriseApi.visus.shareDashboard(id, payload),
    onSuccess: async () => {
      showSuccess('Dashboard access updated.');
      setShareTarget(null);
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboards'] });
      refetch();
    },
    onError: showApiError,
  });

  const duplicateMutation = useMutation({
    mutationFn: (id: string) => enterpriseApi.visus.duplicateDashboard(id),
    onSuccess: async () => {
      showSuccess('Dashboard duplicated.');
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboards'] });
      refetch();
    },
    onError: showApiError,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => enterpriseApi.visus.deleteDashboard(id),
    onSuccess: async () => {
      showSuccess('Dashboard deleted.');
      setDeleteTarget(null);
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboards'] });
      refetch();
    },
    onError: showApiError,
  });

  const columns: ColumnDef<VisusDashboard>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Dashboard',
      enableSorting: true,
      cell: ({ row }) => (
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Link href={`/visus/dashboards/${row.original.id}`} className="font-medium hover:underline">
              {row.original.name}
            </Link>
            {row.original.is_default ? <Badge variant="secondary">Default</Badge> : null}
            {row.original.is_system ? <Badge variant="outline">System</Badge> : null}
          </div>
          <p className="max-w-xl text-xs text-muted-foreground">{row.original.description}</p>
        </div>
      ),
    },
    {
      id: 'visibility',
      accessorKey: 'visibility',
      header: 'Visibility',
      enableSorting: true,
      cell: ({ row }) => <Badge variant={visibilityVariant(row.original.visibility)}>{row.original.visibility}</Badge>,
    },
    {
      id: 'widget_count',
      accessorKey: 'widget_count',
      header: 'Widgets',
      enableSorting: true,
      cell: ({ row }) => <span className="text-sm">{row.original.widget_count ?? 0}</span>,
    },
    {
      id: 'updated_at',
      accessorKey: 'updated_at',
      header: 'Updated',
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.updated_at} />,
    },
  ];

  const rowActions: RowAction<VisusDashboard>[] = [
    {
      label: 'Open',
      icon: LayoutDashboard,
      onClick: (row) => {
        router.push(`/visus/dashboards/${row.id}`);
      },
    },
    {
      label: 'Edit',
      icon: Pencil,
      onClick: (row) => setEditTarget(row),
      disabled: (row) => row.is_system,
    },
    {
      label: 'Share',
      icon: Share2,
      onClick: (row) => setShareTarget(row),
    },
    {
      label: 'Duplicate',
      icon: CopyPlus,
      onClick: (row) => duplicateMutation.mutate(row.id),
    },
    {
      label: 'Delete',
      icon: Trash2,
      variant: 'destructive',
      onClick: (row) => setDeleteTarget(row),
      hidden: (row) => row.is_system,
    },
  ];

  const users = usersQuery.data?.data ?? ([] as UserDirectoryEntry[]);

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader
          title="Dashboards"
          description="Author, duplicate, share, and maintain executive dashboard definitions."
          actions={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Dashboard
            </Button>
          }
        />

        <DataTable
          {...tableProps}
          columns={columns}
          filters={DASHBOARD_FILTERS}
          rowActions={rowActions}
          searchPlaceholder="Search dashboards..."
          onRowClick={(row) => {
            router.push(`/visus/dashboards/${row.id}`);
          }}
          emptyState={{
            icon: LayoutDashboard,
            title: 'No dashboards found',
            description: 'Create an executive dashboard to start composing a reusable reporting surface.',
            action: {
              label: 'Create Dashboard',
              onClick: () => setCreateOpen(true),
              icon: Plus,
            },
          }}
        />

        <DashboardFormDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          users={users}
          pending={createMutation.isPending}
          onSubmit={async (payload) => {
            await createMutation.mutateAsync(payload);
          }}
        />

        <DashboardFormDialog
          open={Boolean(editTarget)}
          onOpenChange={(open) => {
            if (!open) setEditTarget(null);
          }}
          dashboard={editTarget}
          users={users}
          pending={updateMutation.isPending}
          onSubmit={async (payload) => {
            if (!editTarget) return;
            await updateMutation.mutateAsync({ id: editTarget.id, payload });
          }}
        />

        <DashboardShareDialog
          open={Boolean(shareTarget)}
          onOpenChange={(open) => {
            if (!open) setShareTarget(null);
          }}
          dashboard={shareTarget}
          users={users}
          pending={shareMutation.isPending}
          onSubmit={async (payload) => {
            if (!shareTarget) return;
            await shareMutation.mutateAsync({ id: shareTarget.id, payload });
          }}
        />

        <ConfirmDialog
          open={Boolean(deleteTarget)}
          onOpenChange={(open) => {
            if (!open) setDeleteTarget(null);
          }}
          title="Delete Dashboard"
          description={`Delete "${deleteTarget?.name}"? Widgets attached to this dashboard will no longer be accessible from Visus.`}
          confirmLabel="Delete Dashboard"
          variant="destructive"
          loading={deleteMutation.isPending}
          onConfirm={async () => {
            if (!deleteTarget) return;
            await deleteMutation.mutateAsync(deleteTarget.id);
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
