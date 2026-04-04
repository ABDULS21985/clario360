'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { ColumnDef } from '@tanstack/react-table';
import { Edit3, Eye, Plus, Power, Trash2, Users } from 'lucide-react';
import { toast } from 'sonner';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PageHeader } from '@/components/common/page-header';
import { PermissionGate } from '@/components/auth/permission-gate';
import { ActorFormDialog } from '@/components/cyber/cti/actor-form-dialog';
import { ExportMenu } from '@/components/cyber/export-menu';
import { DataTable } from '@/components/shared/data-table/data-table';
import { selectColumn } from '@/components/shared/data-table/columns/common-columns';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { useAuth } from '@/hooks/use-auth';
import { useDataTable } from '@/hooks/use-data-table';
import {
  deleteThreatActor,
  fetchThreatActors,
  flattenThreatActorFetchParams,
  updateThreatActor,
} from '@/lib/cti-api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { countryCodeToFlag, formatRelativeTime } from '@/lib/cti-utils';
import {
  CTI_ACTOR_TYPE_LABELS,
  CTI_MOTIVATION_LABELS,
  CTI_SOPHISTICATION_LABELS,
  type CTIThreatActor,
} from '@/types/cti';
import type { PaginatedResponse } from '@/types/api';
import type { BulkAction, FetchParams, FilterConfig, RowAction } from '@/types/table';

function fetchActorRows(params: FetchParams): Promise<PaginatedResponse<CTIThreatActor>> {
  return fetchThreatActors(flattenThreatActorFetchParams(params));
}

export default function CTIActorsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('cyber:write');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [formOpen, setFormOpen] = useState(false);
  const [editingActor, setEditingActor] = useState<CTIThreatActor | null>(null);
  const [deleteCandidate, setDeleteCandidate] = useState<CTIThreatActor | null>(null);

  const { tableProps, refetch } = useDataTable<CTIThreatActor>({
    fetchFn: fetchActorRows,
    queryKey: 'cti-actors',
    defaultPageSize: 25,
    defaultSort: { column: 'risk_score', direction: 'desc' },
  });

  const setActorsInCache = (
    updater: (actor: CTIThreatActor) => CTIThreatActor | null,
  ): Array<[readonly unknown[], unknown]> => {
    const snapshots = queryClient.getQueriesData({ queryKey: ['cti-actors'] });
    queryClient.setQueriesData<PaginatedResponse<CTIThreatActor>>(
      { queryKey: ['cti-actors'] },
      (current) => {
        if (!current) {
          return current;
        }

        return {
          ...current,
          data: current.data
            .map((actor) => updater(actor))
            .filter((actor): actor is CTIThreatActor => actor !== null),
        };
      },
    );
    return snapshots;
  };

  const restoreSnapshots = (snapshots?: Array<[readonly unknown[], unknown]>) => {
    snapshots?.forEach(([key, value]) => queryClient.setQueryData(key, value));
  };

  const toggleActiveMutation = useMutation({
    mutationFn: async ({ ids, isActive }: { ids: string[]; isActive: boolean }) => {
      await Promise.all(ids.map((id) => updateThreatActor(id, { is_active: isActive })));
    },
    onMutate: async ({ ids, isActive }) => {
      const snapshots = setActorsInCache((actor) => (
        ids.includes(actor.id)
          ? {
              ...actor,
              is_active: isActive,
            }
          : actor
      ));
      return { snapshots };
    },
    onError: (_error, _variables, context) => {
      restoreSnapshots(context?.snapshots);
      toast.error('Failed to update actor status');
    },
    onSuccess: async (_result, variables) => {
      await queryClient.invalidateQueries({ queryKey: ['cti-actors'] });
      toast.success(`${variables.ids.length} actor${variables.ids.length === 1 ? '' : 's'} updated`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async ({ ids }: { ids: string[] }) => {
      await Promise.all(ids.map((id) => deleteThreatActor(id)));
    },
    onMutate: async ({ ids }) => {
      const snapshots = setActorsInCache((actor) => (ids.includes(actor.id) ? null : actor));
      return { snapshots };
    },
    onError: (_error, _variables, context) => {
      restoreSnapshots(context?.snapshots);
      toast.error('Failed to delete threat actor');
    },
    onSuccess: async (_result, variables) => {
      await queryClient.invalidateQueries({ queryKey: ['cti-actors'] });
      toast.success(`${variables.ids.length} actor${variables.ids.length === 1 ? '' : 's'} deleted`);
      setSelectedIds([]);
    },
  });

  const filters = useMemo<FilterConfig[]>(
    () => [
      {
        key: 'actor_type',
        label: 'Actor Type',
        type: 'multi-select',
        options: [
          { label: 'State Sponsored', value: 'state_sponsored' },
          { label: 'Cybercriminal', value: 'cybercriminal' },
          { label: 'Hacktivist', value: 'hacktivist' },
          { label: 'Insider', value: 'insider' },
          { label: 'Unknown', value: 'unknown' },
        ],
      },
      {
        key: 'sophistication',
        label: 'Sophistication',
        type: 'multi-select',
        options: [
          { label: 'Advanced', value: 'advanced' },
          { label: 'Intermediate', value: 'intermediate' },
          { label: 'Basic', value: 'basic' },
        ],
      },
      {
        key: 'is_active',
        label: 'Active',
        type: 'select',
        options: [
          { label: 'Active', value: 'true' },
          { label: 'Inactive', value: 'false' },
        ],
      },
    ],
    [],
  );

  const rowActions = useMemo<RowAction<CTIThreatActor>[]>(() => {
    const baseActions: RowAction<CTIThreatActor>[] = [
      {
        label: 'View Actor',
        icon: Eye,
        onClick: (actor) => router.push(`${ROUTES.CYBER_CTI_ACTORS}/${actor.id}`),
      },
    ];

    if (!canWrite) {
      return baseActions;
    }

    return [
      ...baseActions,
      {
        label: 'Edit Actor',
        icon: Edit3,
        onClick: (actor) => setEditingActor(actor),
      },
      {
        label: 'Toggle Active',
        icon: Power,
        onClick: (actor) => toggleActiveMutation.mutate({ ids: [actor.id], isActive: !actor.is_active }),
      },
      {
        label: 'Delete Actor',
        icon: Trash2,
        variant: 'destructive',
        onClick: (actor) => setDeleteCandidate(actor),
      },
    ];
  }, [canWrite, router, toggleActiveMutation]);

  const bulkActions = useMemo<BulkAction[]>(() => {
    if (!canWrite) {
      return [];
    }

    return [
      {
        label: 'Activate Selected',
        onClick: async (ids) => toggleActiveMutation.mutateAsync({ ids, isActive: true }),
      },
      {
        label: 'Deactivate Selected',
        onClick: async (ids) => toggleActiveMutation.mutateAsync({ ids, isActive: false }),
      },
      {
        label: 'Delete Selected',
        variant: 'destructive',
        onClick: async (ids) => deleteMutation.mutateAsync({ ids }),
      },
    ];
  }, [canWrite, deleteMutation, toggleActiveMutation]);

  const columns = useMemo<ColumnDef<CTIThreatActor>[]>(() => {
    const base: ColumnDef<CTIThreatActor>[] = [
      {
        accessorKey: 'name',
        header: 'Actor',
        enableSorting: true,
        size: 260,
        cell: ({ row }) => (
          <div className="space-y-1">
            <Link
              href={`${ROUTES.CYBER_CTI_ACTORS}/${row.original.id}`}
              className="font-medium text-foreground hover:underline"
              onClick={(event) => event.stopPropagation()}
            >
              {row.original.name}
            </Link>
            <p className="line-clamp-1 text-xs text-muted-foreground">
              {row.original.aliases.length > 0 ? row.original.aliases.join(', ') : 'No aliases'}
            </p>
          </div>
        ),
      },
      {
        accessorKey: 'actor_type',
        header: 'Type',
        enableSorting: true,
        cell: ({ row }) => CTI_ACTOR_TYPE_LABELS[row.original.actor_type] ?? row.original.actor_type,
      },
      {
        accessorKey: 'origin_country_code',
        header: 'Origin',
        enableSorting: true,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {countryCodeToFlag(row.original.origin_country_code)} {row.original.origin_country_code?.toUpperCase() ?? 'Unknown'}
          </span>
        ),
      },
      {
        accessorKey: 'sophistication_level',
        header: 'Sophistication',
        enableSorting: true,
        cell: ({ row }) => CTI_SOPHISTICATION_LABELS[row.original.sophistication_level],
      },
      {
        accessorKey: 'primary_motivation',
        header: 'Motivation',
        enableSorting: true,
        cell: ({ row }) => CTI_MOTIVATION_LABELS[row.original.primary_motivation] ?? row.original.primary_motivation,
      },
      {
        accessorKey: 'risk_score',
        header: 'Risk',
        enableSorting: true,
        cell: ({ row }) => <span className="font-medium tabular-nums">{row.original.risk_score.toFixed(1)}</span>,
      },
      {
        accessorKey: 'is_active',
        header: 'Status',
        enableSorting: true,
        cell: ({ row }) => (
          <span className={row.original.is_active ? 'text-emerald-600' : 'text-muted-foreground'}>
            {row.original.is_active ? 'Active' : 'Inactive'}
          </span>
        ),
      },
      {
        accessorKey: 'last_activity_at',
        header: 'Last Activity',
        enableSorting: true,
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatRelativeTime(row.original.last_activity_at)}
          </span>
        ),
      },
    ];

    return canWrite ? [selectColumn<CTIThreatActor>(), ...base] : base;
  }, [canWrite]);

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Threat Actor Profiles"
          description="Monitor threat actor profiles, motivations, and campaign attribution."
          actions={(
            <div className="flex flex-wrap items-center gap-2">
              <ExportMenu
                entityType="cti-actors"
                baseUrl={API_ENDPOINTS.CTI_ACTORS}
                currentFilters={{ ...tableProps.activeFilters, search: tableProps.searchValue ?? '' }}
                totalCount={tableProps.totalRows}
                enabledFormats={['csv', 'json']}
                selectedCount={selectedIds.length}
              />
              <PermissionGate permission="cyber:write">
                <Button size="sm" onClick={() => setFormOpen(true)}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  New Actor
                </Button>
              </PermissionGate>
            </div>
          )}
        />

        <DataTable
          {...tableProps}
          columns={columns}
          filters={filters}
          getRowId={(row) => row.id}
          enableSelection={canWrite}
          onSelectionChange={setSelectedIds}
          bulkActions={bulkActions}
          rowActions={rowActions}
          searchPlaceholder="Search actors by name, alias, or MITRE group…"
          emptyState={{
            icon: Users,
            title: 'No threat actors found',
            description: 'Threat actor profiles will appear here as intelligence records are curated.',
            action: canWrite
              ? { label: 'Create Actor', onClick: () => setFormOpen(true), icon: Plus }
              : undefined,
          }}
          onRowClick={(row) => router.push(`${ROUTES.CYBER_CTI_ACTORS}/${row.id}`)}
        />
      </div>

      <ActorFormDialog
        open={formOpen || Boolean(editingActor)}
        onOpenChange={(open) => {
          if (!open) {
            setFormOpen(false);
            setEditingActor(null);
          }
        }}
        actor={editingActor}
        onSuccess={() => {
          setFormOpen(false);
          setEditingActor(null);
          void refetch();
        }}
      />

      <ConfirmDialog
        open={Boolean(deleteCandidate)}
        onOpenChange={(open) => !open && setDeleteCandidate(null)}
        title="Delete threat actor"
        description="This removes the threat actor profile from CTI views."
        confirmLabel="Delete Actor"
        variant="destructive"
        typeToConfirm={deleteCandidate?.name}
        loading={deleteMutation.isPending}
        onConfirm={async () => {
          if (!deleteCandidate) {
            return;
          }
          await deleteMutation.mutateAsync({ ids: [deleteCandidate.id] });
          setDeleteCandidate(null);
        }}
      />
    </PermissionRedirect>
  );
}
