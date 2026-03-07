'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { Users } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { committeeStatusConfig } from '@/lib/status-configs';
import { summarizeNamedRecords } from '@/lib/suite-utils';
import type { ActaCommittee } from '@/types/suites';

export default function ActaCommitteesPage() {
  const { tableProps, searchValue, setSearch } = useDataTable<ActaCommittee>({
    queryKey: 'acta-committees',
    fetchFn: (params) => fetchSuitePaginated<ActaCommittee>(API_ENDPOINTS.ACTA_COMMITTEES, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const columns: ColumnDef<ActaCommittee>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Committee',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs text-muted-foreground capitalize">{row.original.type.replace(/_/g, ' ')}</p>
        </div>
      ),
    },
    {
      id: 'members',
      header: 'Members',
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{summarizeNamedRecords(row.original.members)}</span>,
    },
    {
      id: 'meeting_frequency',
      accessorKey: 'meeting_frequency',
      header: 'Frequency',
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{row.original.meeting_frequency ?? 'Unscheduled'}</span>,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={committeeStatusConfig} size="sm" />,
    },
    {
      id: 'updated_at',
      accessorKey: 'updated_at',
      header: 'Updated',
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.updated_at} />,
    },
  ];

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Committees" description="Board and governance committees with membership and cadence visibility." />
        <DataTable
          {...tableProps}
          columns={columns}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search committees..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: Users,
            title: 'No committees found',
            description: 'No committees matched the current search.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
