'use client';

import { useMemo, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { FolderOpen, LayoutGrid, Rows3 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { EmptyState } from '@/components/common/empty-state';
import { Button } from '@/components/ui/button';
import { SearchInput } from '@/components/shared/forms/search-input';
import { DataTableToolbar } from '@/components/shared/data-table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { dataSuiteApi, type ConnectionTestResult, type DataSource, type SyncHistory } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';
import type { FilterConfig } from '@/types/table';
import { CreateSourceWizard } from '@/app/(dashboard)/data/sources/_components/create-source-wizard';
import { EditSourceDialog } from '@/app/(dashboard)/data/sources/_components/edit-source-dialog';
import { SourceGridView } from '@/app/(dashboard)/data/sources/_components/source-grid-view';
import { SourceTableView } from '@/app/(dashboard)/data/sources/_components/source-table-view';
import { SyncProgressIndicator } from '@/app/(dashboard)/data/sources/_components/sync-progress-indicator';

type TestState = {
  loading: boolean;
  result?: ConnectionTestResult | null;
  error?: string | null;
};

const SOURCE_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'multi-select',
    options: [
      { label: 'PostgreSQL', value: 'postgresql' },
      { label: 'MySQL', value: 'mysql' },
      { label: 'ClickHouse', value: 'clickhouse' },
      { label: 'Dolt', value: 'dolt' },
      { label: 'Apache Impala', value: 'impala' },
      { label: 'Apache Hive', value: 'hive' },
      { label: 'HDFS', value: 'hdfs' },
      { label: 'Apache Spark', value: 'spark' },
      { label: 'Dagster', value: 'dagster' },
      { label: 'API', value: 'api' },
      { label: 'CSV', value: 'csv' },
      { label: 'S3', value: 's3' },
      { label: 'Stream', value: 'stream' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select',
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Syncing', value: 'syncing' },
      { label: 'Inactive', value: 'inactive' },
      { label: 'Error', value: 'error' },
      { label: 'Pending Test', value: 'pending_test' },
    ],
  },
];

export default function DataSourcesPage() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<DataSource | null>(null);
  const [syncSource, setSyncSource] = useState<DataSource | null>(null);
  const [testStates, setTestStates] = useState<Record<string, TestState>>({});

  const view = searchParams.get('view') === 'table' ? 'table' : 'cards';

  const { tableProps, searchValue, setSearch, refetch } = useDataTable<DataSource>({
    queryKey: 'data-sources',
    fetchFn: (params) => dataSuiteApi.listSources(params),
    defaultPageSize: 24,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const sources = tableProps.data;
  const hasSources = sources.length > 0;

  const viewActions = useMemo(
    () => (
      <div className="flex items-center gap-2">
        <Button
          type="button"
          size="icon"
          variant={view === 'cards' ? 'default' : 'outline'}
          onClick={() => {
            const params = new URLSearchParams(searchParams.toString());
            params.set('view', 'cards');
            router.push(`${pathname}?${params.toString()}`);
          }}
        >
          <LayoutGrid className="h-4 w-4" />
        </Button>
        <Button
          type="button"
          size="icon"
          variant={view === 'table' ? 'default' : 'outline'}
          onClick={() => {
            const params = new URLSearchParams(searchParams.toString());
            params.set('view', 'table');
            router.push(`${pathname}?${params.toString()}`);
          }}
        >
          <Rows3 className="h-4 w-4" />
        </Button>
      </div>
    ),
    [pathname, router, searchParams, view],
  );

  const clearTestStateLater = (sourceId: string) => {
    window.setTimeout(() => {
      setTestStates((current) => {
        const next = { ...current };
        delete next[sourceId];
        return next;
      });
    }, 10_000);
  };

  const handleTest = async (source: DataSource) => {
    setTestStates((current) => ({ ...current, [source.id]: { loading: true } }));
    try {
      const result = await dataSuiteApi.testSource(source.id);
      setTestStates((current) => ({ ...current, [source.id]: { loading: false, result } }));
      clearTestStateLater(source.id);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Connection test failed';
      setTestStates((current) => ({ ...current, [source.id]: { loading: false, error: message } }));
      clearTestStateLater(source.id);
    }
  };

  const handleSync = async (source: DataSource) => {
    try {
      await dataSuiteApi.syncSource(source.id, 'full');
      setSyncSource(source);
      showSuccess('Sync started.', `${source.name} is now syncing.`);
    } catch (error) {
      showApiError(error);
    }
  };

  const handleDelete = async (source: DataSource) => {
    if (!window.confirm(`Delete source "${source.name}"?`)) {
      return;
    }
    try {
      await dataSuiteApi.deleteSource(source.id);
      showSuccess('Source deleted.');
      void refetch();
    } catch (error) {
      showApiError(error);
    }
  };

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Sources"
          description="Connected operational, file, API, and object-store sources available to the data platform."
          actions={
            <div className="flex items-center gap-2">
              {viewActions}
              <Button type="button" onClick={() => setWizardOpen(true)}>
                + Add Source
              </Button>
            </div>
          }
        />

        {!hasSources && !tableProps.isLoading ? (
          <EmptyState
            icon={FolderOpen}
            title="No data sources found"
            description="Connect your first governed source to begin schema discovery and pipeline orchestration."
            action={{ label: 'Add Source', onClick: () => setWizardOpen(true) }}
          />
        ) : view === 'cards' ? (
          <div className="space-y-4">
            <DataTableToolbar
              searchSlot={
                <SearchInput
                  value={searchValue}
                  onChange={setSearch}
                  placeholder="Search sources..."
                  loading={tableProps.isLoading}
                />
              }
              filters={SOURCE_FILTERS}
              activeFilters={tableProps.activeFilters}
              onFilterChange={tableProps.onFilterChange}
              onClearFilters={tableProps.onClearFilters}
            />
            <SourceGridView
              sources={sources}
              testStates={testStates}
              onTest={(source) => void handleTest(source)}
              onSync={(source) => void handleSync(source)}
              onEdit={setEditingSource}
              onDelete={(source) => void handleDelete(source)}
            />
          </div>
        ) : (
          <SourceTableView
            tableProps={tableProps}
            searchValue={searchValue}
            setSearch={setSearch}
            filters={SOURCE_FILTERS}
            onRowClick={(source) => router.push(`/data/sources/${source.id}`)}
            onEdit={setEditingSource}
            onDelete={(source) => void handleDelete(source)}
            onTest={(source) => void handleTest(source)}
            onSync={(source) => void handleSync(source)}
          />
        )}

        {view === 'cards' && hasSources ? (
          <div className="flex items-center justify-between rounded-lg border px-4 py-3 text-sm">
            <span>
              Page {tableProps.page} of {Math.max(1, Math.ceil(tableProps.totalRows / tableProps.pageSize))}
            </span>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={tableProps.page <= 1}
                onClick={() => tableProps.onPageChange(tableProps.page - 1)}
              >
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={tableProps.page >= Math.ceil(tableProps.totalRows / tableProps.pageSize)}
                onClick={() => tableProps.onPageChange(tableProps.page + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        ) : null}

        <CreateSourceWizard
          open={wizardOpen}
          onOpenChange={setWizardOpen}
          onCreated={() => void refetch()}
        />

        <EditSourceDialog
          open={Boolean(editingSource)}
          onOpenChange={(next) => {
            if (!next) {
              setEditingSource(null);
            }
          }}
          source={editingSource}
          onUpdated={() => void refetch()}
        />

        <SyncProgressIndicator
          open={Boolean(syncSource)}
          onOpenChange={(next) => {
            if (!next) {
              setSyncSource(null);
            }
          }}
          source={syncSource}
          onComplete={() => void refetch()}
        />
      </div>
    </PermissionRedirect>
  );
}
