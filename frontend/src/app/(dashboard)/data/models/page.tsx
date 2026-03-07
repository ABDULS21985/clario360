'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { Boxes } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { datasetStatusConfig } from '@/lib/status-configs';
import { objectKeyCount } from '@/lib/suite-utils';
import type { Dataset } from '@/types/suites';

const DATASET_FILTERS = [
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Published', value: 'published' },
      { label: 'Draft', value: 'draft' },
      { label: 'Archived', value: 'archived' },
    ],
  },
];

export default function DataModelsPage() {
  const { tableProps, searchValue, setSearch } = useDataTable<Dataset>({
    queryKey: 'data-models',
    fetchFn: (params) => fetchSuitePaginated<Dataset>(API_ENDPOINTS.DATA_DATASETS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const columns: ColumnDef<Dataset>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Model / Dataset',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs text-muted-foreground">
            {row.original.source_name ?? 'Unmapped source'} • v{row.original.version}
          </p>
        </div>
      ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={datasetStatusConfig} size="sm" />,
    },
    {
      id: 'schema_definition',
      header: 'Schema',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {objectKeyCount(row.original.schema_definition)} field{objectKeyCount(row.original.schema_definition) === 1 ? '' : 's'}
        </span>
      ),
    },
    {
      id: 'lineage',
      header: 'Lineage',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {objectKeyCount(row.original.lineage)} link{objectKeyCount(row.original.lineage) === 1 ? '' : 's'}
        </span>
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
        <PageHeader title="Data Models" description="Registered datasets, semantic models, and their metadata completeness." />

        <DataTable
          {...tableProps}
          columns={columns}
          filters={DATASET_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search datasets and models..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: Boxes,
            title: 'No datasets found',
            description: 'No datasets or models matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
