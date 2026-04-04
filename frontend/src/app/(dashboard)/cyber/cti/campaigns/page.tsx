'use client';

import { useMemo } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { formatDistanceToNow } from 'date-fns';
import { Target } from 'lucide-react';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PageHeader } from '@/components/common/page-header';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { fetchCampaigns, flattenCampaignFetchParams } from '@/lib/cti-api';
import type { CTICampaign } from '@/types/cti';
import type { FilterConfig, FetchParams } from '@/types/table';
import type { PaginatedResponse } from '@/types/api';

function fetchCampaignRows(params: FetchParams): Promise<PaginatedResponse<CTICampaign>> {
  return fetchCampaigns(flattenCampaignFetchParams(params));
}

export default function CTICampaignsPage() {
  const table = useDataTable<CTICampaign>({
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

  const columns = useMemo<ColumnDef<CTICampaign>[]>(
    () => [
      {
        accessorKey: 'severity_code',
        header: 'Severity',
        enableSorting: true,
        cell: ({ row }) => <CTISeverityBadge severity={row.original.severity_code} size="sm" />,
      },
      {
        accessorKey: 'name',
        header: 'Campaign',
        enableSorting: true,
        size: 280,
        cell: ({ row }) => (
          <div className="space-y-1">
            <p className="font-medium">{row.original.name}</p>
            <p className="text-xs text-muted-foreground">{row.original.campaign_code}</p>
          </div>
        ),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        enableSorting: true,
        cell: ({ row }) => <CTIStatusBadge status={row.original.status} type="campaign" />,
      },
      {
        accessorKey: 'actor_name',
        header: 'Primary Actor',
        enableSorting: false,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">{row.original.actor_name || 'Unknown actor'}</span>
        ),
      },
      {
        accessorKey: 'ioc_count',
        header: 'IOCs',
        enableSorting: true,
        cell: ({ row }) => <span className="font-medium tabular-nums">{row.original.ioc_count.toLocaleString()}</span>,
      },
      {
        accessorKey: 'event_count',
        header: 'Events',
        enableSorting: true,
        cell: ({ row }) => <span className="font-medium tabular-nums">{row.original.event_count.toLocaleString()}</span>,
      },
      {
        accessorKey: 'first_seen_at',
        header: 'First Seen',
        enableSorting: true,
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatDistanceToNow(new Date(row.original.first_seen_at), { addSuffix: true })}
          </span>
        ),
      },
    ],
    [],
  );

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
    ],
    [],
  );

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Active Campaigns"
          description="Track CTI campaigns, their activity levels, and associated actors."
        />

        <DataTable
          {...table.tableProps}
          columns={columns}
          filters={filters}
          searchPlaceholder="Search campaigns by name, code, or actor…"
          emptyState={{
            icon: Target,
            title: 'No campaigns found',
            description: 'Campaigns will appear here as CTI data is linked and aggregated.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
