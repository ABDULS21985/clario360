'use client';

import { useMemo } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { formatDistanceToNow } from 'date-fns';
import { Users } from 'lucide-react';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PageHeader } from '@/components/common/page-header';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { fetchThreatActors, flattenThreatActorFetchParams } from '@/lib/cti-api';
import {
  CTI_ACTOR_TYPE_LABELS,
  CTI_MOTIVATION_LABELS,
  CTI_SOPHISTICATION_LABELS,
  type CTIThreatActor,
} from '@/types/cti';
import type { FilterConfig, FetchParams } from '@/types/table';
import type { PaginatedResponse } from '@/types/api';

function fetchActorRows(params: FetchParams): Promise<PaginatedResponse<CTIThreatActor>> {
  return fetchThreatActors(flattenThreatActorFetchParams(params));
}

export default function CTIActorsPage() {
  const table = useDataTable<CTIThreatActor>({
    fetchFn: fetchActorRows,
    queryKey: 'cti-actors',
    defaultPageSize: 25,
    defaultSort: { column: 'risk_score', direction: 'desc' },
  });

  const columns = useMemo<ColumnDef<CTIThreatActor>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Actor',
        enableSorting: true,
        size: 260,
        cell: ({ row }) => (
          <div className="space-y-1">
            <p className="font-medium">{row.original.name}</p>
            <p className="text-xs text-muted-foreground">
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
        cell: ({ row }) => row.original.origin_country_code?.toUpperCase() ?? '—',
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
        header: 'Risk Score',
        enableSorting: true,
        cell: ({ row }) => <span className="font-medium tabular-nums">{row.original.risk_score.toFixed(1)}</span>,
      },
      {
        accessorKey: 'last_activity_at',
        header: 'Last Activity',
        enableSorting: true,
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {row.original.last_activity_at
              ? formatDistanceToNow(new Date(row.original.last_activity_at), { addSuffix: true })
              : 'Unknown'}
          </span>
        ),
      },
    ],
    [],
  );

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

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Threat Actors"
          description="Monitor threat actor profiles, motivations, and activity patterns."
        />

        <DataTable
          {...table.tableProps}
          columns={columns}
          filters={filters}
          searchPlaceholder="Search actors by name, alias, or MITRE group…"
          emptyState={{
            icon: Users,
            title: 'No threat actors found',
            description: 'Threat actor profiles will appear here as intelligence records are curated.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
