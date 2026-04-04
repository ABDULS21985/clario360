'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ColumnDef } from '@tanstack/react-table';
import { Edit3, Eye, Plus, Target, Trash2, Waves } from 'lucide-react';
import { toast } from 'sonner';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PageHeader } from '@/components/common/page-header';
import { PermissionGate } from '@/components/auth/permission-gate';
import { CampaignFormDialog } from '@/components/cyber/cti/campaign-form-dialog';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { ExportMenu } from '@/components/cyber/export-menu';
import { DataTable } from '@/components/shared/data-table/data-table';
import { selectColumn } from '@/components/shared/data-table/columns/common-columns';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { useAuth } from '@/hooks/use-auth';
import { useDataTable } from '@/hooks/use-data-table';
import {
  deleteCampaign,
  fetchCampaigns,
  fetchThreatActors,
  flattenCampaignFetchParams,
  updateCampaignStatus,
} from '@/lib/cti-api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { formatRelativeTime } from '@/lib/cti-utils';
import type { CTICampaign } from '@/types/cti';
import type { PaginatedResponse } from '@/types/api';
import type { BulkAction, FetchParams, FilterConfig, RowAction } from '@/types/table';

function fetchCampaignRows(params: FetchParams): Promise<PaginatedResponse<CTICampaign>> {
  return fetchCampaigns(flattenCampaignFetchParams(params));
}

const STATUS_ORDER = ['active', 'monitoring', 'dormant', 'resolved'] as const;

export default function CTICampaignsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('cyber:write');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [formOpen, setFormOpen] = useState(false);
  const [editingCampaign, setEditingCampaign] = useState<CTICampaign | null>(null);
  const [deleteCandidate, setDeleteCandidate] = useState<CTICampaign | null>(null);

  const { tableProps, refetch } = useDataTable<CTICampaign>({
    fetchFn: fetchCampaignRows,
    queryKey: 'cti-campaigns',
    defaultPageSize: 25,
    defaultSort: { column: 'first_seen_at', direction: 'desc' },
    wsTopics: [
      'com.clario360.cyber.cti.campaign.created',
      'com.clario360.cyber.cti.campaign.updated',
      'com.clario360.cyber.cti.campaign.status-changed',
    ],
  });

  const actorsQuery = useQuery({
    queryKey: ['cti-campaign-filter-actors'],
    queryFn: () => fetchThreatActors({ page: 1, per_page: 200, sort: 'name', order: 'asc' }),
  });

  const countsQuery = useQuery({
    queryKey: ['cti-campaign-status-counts'],
    queryFn: async () => {
      const results = await Promise.all(
        STATUS_ORDER.map(async (status) => {
          const response = await fetchCampaigns({ page: 1, per_page: 1, status });
          return [status, response.meta.total] as const;
        }),
      );
      return Object.fromEntries(results);
    },
  });

  const mutateCampaignRows = (
    updater: (campaign: CTICampaign) => CTICampaign | null,
  ): Array<[readonly unknown[], unknown]> => {
    const snapshots = queryClient.getQueriesData({ queryKey: ['cti-campaigns'] });
    queryClient.setQueriesData<PaginatedResponse<CTICampaign>>(
      { queryKey: ['cti-campaigns'] },
      (current) => {
        if (!current) {
          return current;
        }

        const nextData = current.data
          .map((campaign) => updater(campaign))
          .filter((campaign): campaign is CTICampaign => campaign !== null);

        return {
          ...current,
          data: nextData,
          meta: {
            ...current.meta,
            total: Math.max(current.meta.total + (nextData.length - current.data.length), 0),
          },
        };
      },
    );
    return snapshots;
  };

  const restoreCampaignRows = (snapshots: Array<[readonly unknown[], unknown]>) => {
    snapshots.forEach(([key, value]) => {
      queryClient.setQueryData(key, value);
    });
  };

  const statusMutation = useMutation({
    mutationFn: async ({ ids, status }: { ids: string[]; status: CTICampaign['status'] }) => {
      await Promise.all(ids.map((id) => updateCampaignStatus(id, status)));
    },
    onMutate: async ({ ids, status }) => {
      const snapshots = mutateCampaignRows((campaign) => (
        ids.includes(campaign.id)
          ? {
              ...campaign,
              status,
            }
          : campaign
      ));
      return { snapshots };
    },
    onError: (_error, _variables, context) => {
      if (context?.snapshots) {
        restoreCampaignRows(context.snapshots);
      }
      toast.error('Failed to update campaign status');
    },
    onSuccess: async (_result, variables) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-campaigns'] }),
        queryClient.invalidateQueries({ queryKey: ['cti-campaign-status-counts'] }),
      ]);
      toast.success(`${variables.ids.length} campaign${variables.ids.length === 1 ? '' : 's'} moved to ${variables.status}`);
      await refetch();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async ({ ids }: { ids: string[] }) => {
      await Promise.all(ids.map((id) => deleteCampaign(id)));
    },
    onMutate: async ({ ids }) => {
      const snapshots = mutateCampaignRows((campaign) => (ids.includes(campaign.id) ? null : campaign));
      return { snapshots };
    },
    onError: (_error, _variables, context) => {
      if (context?.snapshots) {
        restoreCampaignRows(context.snapshots);
      }
      toast.error('Failed to delete campaign');
    },
    onSuccess: async (_result, variables) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-campaigns'] }),
        queryClient.invalidateQueries({ queryKey: ['cti-campaign-status-counts'] }),
      ]);
      toast.success(`${variables.ids.length} campaign${variables.ids.length === 1 ? '' : 's'} deleted`);
      setSelectedIds([]);
    },
  });

  const filters = useMemo<FilterConfig[]>(
    () => [
      {
        key: 'status',
        label: 'Status',
        type: 'multi-select',
        options: [
          { label: 'Active', value: 'active' },
          { label: 'Monitoring', value: 'monitoring' },
          { label: 'Dormant', value: 'dormant' },
          { label: 'Resolved', value: 'resolved' },
          { label: 'Archived', value: 'archived' },
        ],
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
          { label: 'Informational', value: 'informational' },
        ],
      },
      {
        key: 'actor_id',
        label: 'Actor',
        type: 'select',
        options: (actorsQuery.data?.data ?? []).map((actor) => ({
          label: actor.name,
          value: actor.id,
        })),
      },
    ],
    [actorsQuery.data],
  );

  const rowActions = useMemo<RowAction<CTICampaign>[]>(() => {
    const baseActions: RowAction<CTICampaign>[] = [
      {
        label: 'View Campaign',
        icon: Eye,
        onClick: (campaign) => router.push(`${ROUTES.CYBER_CTI_CAMPAIGNS}/${campaign.id}`),
      },
    ];

    if (!canWrite) {
      return baseActions;
    }

    return [
      ...baseActions,
      {
        label: 'Edit Campaign',
        icon: Edit3,
        onClick: (campaign) => setEditingCampaign(campaign),
      },
      {
        label: 'Move to Monitoring',
        icon: Waves,
        hidden: (campaign) => campaign.status === 'monitoring',
        onClick: (campaign) => statusMutation.mutate({ ids: [campaign.id], status: 'monitoring' }),
      },
      {
        label: 'Resolve Campaign',
        icon: Target,
        hidden: (campaign) => campaign.status === 'resolved',
        onClick: (campaign) => statusMutation.mutate({ ids: [campaign.id], status: 'resolved' }),
      },
      {
        label: 'Delete Campaign',
        icon: Trash2,
        variant: 'destructive',
        onClick: (campaign) => setDeleteCandidate(campaign),
      },
    ];
  }, [canWrite, router, statusMutation]);

  const bulkActions = useMemo<BulkAction[]>(() => {
    if (!canWrite) {
      return [];
    }

    return [
      {
        label: 'Set Monitoring',
        onClick: async (ids) => statusMutation.mutateAsync({ ids, status: 'monitoring' }),
      },
      {
        label: 'Resolve Selected',
        onClick: async (ids) => statusMutation.mutateAsync({ ids, status: 'resolved' }),
      },
      {
        label: 'Delete Selected',
        variant: 'destructive',
        onClick: async (ids) => deleteMutation.mutateAsync({ ids }),
      },
    ];
  }, [canWrite, deleteMutation, statusMutation]);

  const columns = useMemo<ColumnDef<CTICampaign>[]>(
    () => {
      const base: ColumnDef<CTICampaign>[] = [
        {
          accessorKey: 'campaign_code',
          header: 'Code',
          enableSorting: true,
          size: 130,
          cell: ({ row }) => (
            <Link
              href={`${ROUTES.CYBER_CTI_CAMPAIGNS}/${row.original.id}`}
              className="font-mono text-xs text-primary hover:underline"
              onClick={(event) => event.stopPropagation()}
            >
              {row.original.campaign_code}
            </Link>
          ),
        },
        {
          accessorKey: 'name',
          header: 'Campaign',
          enableSorting: true,
          size: 280,
          cell: ({ row }) => (
            <div className="space-y-1">
              <Link
                href={`${ROUTES.CYBER_CTI_CAMPAIGNS}/${row.original.id}`}
                className="font-mono text-sm font-semibold uppercase tracking-[0.08em] text-foreground hover:underline"
                onClick={(event) => event.stopPropagation()}
              >
                {row.original.name}
              </Link>
              <p className="line-clamp-1 text-xs text-muted-foreground">
                {row.original.description || row.original.target_description || 'No analyst narrative'}
              </p>
            </div>
          ),
        },
        {
          accessorKey: 'actor_name',
          header: 'Actor',
          size: 190,
          cell: ({ row }) => (
            row.original.primary_actor_id && row.original.actor_name ? (
              <Link
                href={`${ROUTES.CYBER_CTI_ACTORS}/${row.original.primary_actor_id}`}
                className="text-sm text-primary hover:underline"
                onClick={(event) => event.stopPropagation()}
              >
                {row.original.actor_name}
              </Link>
            ) : (
              <span className="text-sm text-muted-foreground">Unassigned</span>
            )
          ),
        },
        {
          accessorKey: 'status',
          header: 'Status',
          enableSorting: true,
          cell: ({ row }) => <CTIStatusBadge status={row.original.status} type="campaign" />,
        },
        {
          accessorKey: 'severity_code',
          header: 'Severity',
          enableSorting: true,
          cell: ({ row }) => <CTISeverityBadge severity={row.original.severity_code} size="sm" />,
        },
        {
          id: 'targets',
          header: 'Targets',
          size: 220,
          cell: ({ row }) => (
            <span className="line-clamp-2 text-xs text-muted-foreground">
              {row.original.target_description || 'Sector and region targets not captured'}
            </span>
          ),
        },
        {
          accessorKey: 'ioc_count',
          header: 'IOCs',
          enableSorting: true,
          cell: ({ row }) => <span className="font-medium tabular-nums text-orange-500">{row.original.ioc_count.toLocaleString()}</span>,
        },
        {
          accessorKey: 'event_count',
          header: 'Events',
          enableSorting: true,
          cell: ({ row }) => <span className="font-medium tabular-nums">{row.original.event_count.toLocaleString()}</span>,
        },
        {
          accessorKey: 'last_seen_at',
          header: 'Last Seen',
          enableSorting: true,
          cell: ({ row }) => (
            <span className="text-xs text-muted-foreground">
              {formatRelativeTime(row.original.last_seen_at ?? row.original.first_seen_at)}
            </span>
          ),
        },
      ];

      return canWrite ? [selectColumn<CTICampaign>(), ...base] : base;
    },
    [canWrite],
  );

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Active Campaigns"
          description="Track and profile ongoing cyber attack campaigns."
          actions={(
            <div className="flex flex-wrap items-center gap-2">
              <ExportMenu
                entityType="cti-campaigns"
                baseUrl={API_ENDPOINTS.CTI_CAMPAIGNS}
                currentFilters={{ ...tableProps.activeFilters, search: tableProps.searchValue ?? '' }}
                totalCount={tableProps.totalRows}
                enabledFormats={['csv', 'json']}
                selectedCount={selectedIds.length}
              />
              <PermissionGate permission="cyber:write">
                <Button size="sm" onClick={() => setFormOpen(true)}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  New Campaign
                </Button>
              </PermissionGate>
            </div>
          )}
        />

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {STATUS_ORDER.map((status) => (
            <CTIKPIStatCard
              key={status}
              label={status.charAt(0).toUpperCase() + status.slice(1)}
              value={countsQuery.data?.[status] ?? 0}
              subtitle="Campaigns"
              onClick={() => tableProps.onFilterChange?.('status', status)}
            />
          ))}
        </div>

        <DataTable
          {...tableProps}
          columns={columns}
          filters={filters}
          getRowId={(row) => row.id}
          enableSelection={canWrite}
          onSelectionChange={setSelectedIds}
          bulkActions={bulkActions}
          rowActions={rowActions}
          searchPlaceholder="Search campaigns by name, code, or actor…"
          emptyState={{
            icon: Target,
            title: 'No campaigns found',
            description: 'Campaigns will appear here as CTI data is linked and aggregated.',
            action: canWrite
              ? { label: 'Create Campaign', onClick: () => setFormOpen(true), icon: Plus }
              : undefined,
          }}
          onRowClick={(row) => router.push(`${ROUTES.CYBER_CTI_CAMPAIGNS}/${row.id}`)}
        />
      </div>

      <CampaignFormDialog
        open={formOpen || Boolean(editingCampaign)}
        onOpenChange={(open) => {
          if (!open) {
            setFormOpen(false);
            setEditingCampaign(null);
          }
        }}
        campaign={editingCampaign}
        onSuccess={() => {
          setFormOpen(false);
          setEditingCampaign(null);
          void refetch();
          void countsQuery.refetch();
        }}
      />

      <ConfirmDialog
        open={Boolean(deleteCandidate)}
        onOpenChange={(open) => !open && setDeleteCandidate(null)}
        title="Delete campaign"
        description="This permanently removes the campaign record from CTI views."
        confirmLabel="Delete Campaign"
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
