'use client';

import { Boxes } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { buildModelColumns } from '@/app/(dashboard)/data/models/_components/model-columns';
import { dataSuiteApi, type DataModel } from '@/lib/data-suite';

const MODEL_FILTERS = [
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select' as const,
    options: [
      { label: 'Draft', value: 'draft' },
      { label: 'Active', value: 'active' },
      { label: 'Deprecated', value: 'deprecated' },
      { label: 'Archived', value: 'archived' },
    ],
  },
  {
    key: 'data_classification',
    label: 'Classification',
    type: 'multi-select' as const,
    options: [
      { label: 'Public', value: 'public' },
      { label: 'Internal', value: 'internal' },
      { label: 'Confidential', value: 'confidential' },
      { label: 'Restricted', value: 'restricted' },
    ],
  },
];

export default function DataModelsPage() {
  const { tableProps, searchValue, setSearch } = useDataTable<DataModel>({
    queryKey: 'data-models',
    fetchFn: (params) => dataSuiteApi.listModels(params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Models"
          description="Governed semantic models derived from discovered sources and used by analytics, quality, and lineage."
        />

        <DataTable
          {...tableProps}
          columns={buildModelColumns()}
          filters={MODEL_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search models..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: Boxes,
            title: 'No models found',
            description: 'No data models matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
