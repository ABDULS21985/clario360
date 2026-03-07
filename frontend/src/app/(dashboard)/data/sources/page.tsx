'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { FolderOpen } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { objectKeyCount } from '@/lib/suite-utils';
import { sourceStatusConfig } from '@/lib/status-configs';
import type { DataSource } from '@/types/suites';

const SOURCE_FILTERS = [
  {
    key: 'type',
    label: 'Type',
    type: 'select' as const,
    options: [
      { label: 'Database', value: 'database' },
      { label: 'API', value: 'api' },
      { label: 'File', value: 'file' },
      { label: 'Stream', value: 'stream' },
      { label: 'Cloud Storage', value: 'cloud_storage' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Syncing', value: 'syncing' },
      { label: 'Inactive', value: 'inactive' },
      { label: 'Error', value: 'error' },
    ],
  },
];

export default function DataSourcesPage() {
  const { tableProps, searchValue, setSearch } = useDataTable<DataSource>({
    queryKey: 'data-sources',
    fetchFn: (params) => fetchSuitePaginated<DataSource>(API_ENDPOINTS.DATA_SOURCES, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const columns: ColumnDef<DataSource>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Source',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs capitalize text-muted-foreground">{row.original.type.replace(/_/g, ' ')}</p>
        </div>
      ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={sourceStatusConfig} size="sm" />,
    },
    {
      id: 'sync_frequency',
      accessorKey: 'sync_frequency',
      header: 'Sync Frequency',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{row.original.sync_frequency ?? 'On demand'}</span>
      ),
    },
    {
      id: 'schema_metadata',
      header: 'Schema Coverage',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {objectKeyCount(row.original.schema_metadata)} mapped field{objectKeyCount(row.original.schema_metadata) === 1 ? '' : 's'}
        </span>
      ),
    },
    {
      id: 'last_synced_at',
      accessorKey: 'last_synced_at',
      header: 'Last Synced',
      cell: ({ row }) =>
        row.original.last_synced_at ? (
          <RelativeTime date={row.original.last_synced_at} />
        ) : (
          <span className="text-sm text-muted-foreground">Never synced</span>
        ),
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
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Data Sources" description="Connected operational and analytical sources available to the data platform." />

        <DataTable
          {...tableProps}
          columns={columns}
          filters={SOURCE_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search data sources..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: FolderOpen,
            title: 'No data sources found',
            description: 'No connected data sources matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
