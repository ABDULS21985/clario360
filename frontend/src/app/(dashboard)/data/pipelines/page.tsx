'use client';

import { useState } from 'react';
import { GitBranch } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { buildPipelineColumns } from '@/app/(dashboard)/data/pipelines/_components/pipeline-columns';
import { dataSuiteApi, type Pipeline } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';

const PIPELINE_FILTERS = [
  {
    key: 'type',
    label: 'Type',
    type: 'multi-select' as const,
    options: [
      { label: 'ETL', value: 'etl' },
      { label: 'ELT', value: 'elt' },
      { label: 'Batch', value: 'batch' },
      { label: 'Streaming', value: 'streaming' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select' as const,
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Paused', value: 'paused' },
      { label: 'Disabled', value: 'disabled' },
      { label: 'Error', value: 'error' },
    ],
  },
];

export default function DataPipelinesPage() {
  const [runningId, setRunningId] = useState<string | null>(null);
  const { tableProps, searchValue, setSearch, refetch } = useDataTable<Pipeline>({
    queryKey: 'data-pipelines',
    fetchFn: (params) => dataSuiteApi.listPipelines(params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
    wsTopics: ['pipeline.run.completed', 'pipeline.run.failed'],
  });

  const runPipeline = async (pipeline: Pipeline) => {
    try {
      setRunningId(pipeline.id);
      await dataSuiteApi.runPipeline(pipeline.id);
      showSuccess('Pipeline run started.', `${pipeline.name} is now executing.`);
      void refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setRunningId(null);
    }
  };

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Pipelines"
          description="Operational pipeline registry with live execution controls, schedule context, and processed volume."
        />

        <DataTable
          {...tableProps}
          columns={buildPipelineColumns({ runningId, onRun: (pipeline) => void runPipeline(pipeline) })}
          filters={PIPELINE_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search pipelines..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: GitBranch,
            title: 'No pipelines found',
            description: 'No pipelines matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
