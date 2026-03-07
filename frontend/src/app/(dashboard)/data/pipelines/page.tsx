'use client';

import { useState } from 'react';
import { GitBranch, Plus } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { Button } from '@/components/ui/button';
import { useDataTable } from '@/hooks/use-data-table';
import { CreatePipelineWizard } from '@/app/(dashboard)/data/pipelines/_components/create-pipeline-wizard';
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
  const [mutatingId, setMutatingId] = useState<string | null>(null);
  const [wizardOpen, setWizardOpen] = useState(false);
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

  const pausePipeline = async (pipeline: Pipeline) => {
    try {
      setMutatingId(pipeline.id);
      await dataSuiteApi.pausePipeline(pipeline.id);
      showSuccess('Pipeline paused.', `${pipeline.name} will not run until resumed.`);
      void refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setMutatingId(null);
    }
  };

  const resumePipeline = async (pipeline: Pipeline) => {
    try {
      setMutatingId(pipeline.id);
      await dataSuiteApi.resumePipeline(pipeline.id);
      showSuccess('Pipeline resumed.', `${pipeline.name} is active again.`);
      void refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setMutatingId(null);
    }
  };

  const deletePipeline = async (pipeline: Pipeline) => {
    if (!window.confirm(`Delete pipeline "${pipeline.name}"?`)) {
      return;
    }
    try {
      setMutatingId(pipeline.id);
      await dataSuiteApi.deletePipeline(pipeline.id);
      showSuccess('Pipeline deleted.');
      void refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setMutatingId(null);
    }
  };

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Pipelines"
          description="Operational pipeline registry with live execution controls, schedule context, and processed volume."
          actions={
            <Button type="button" onClick={() => setWizardOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create pipeline
            </Button>
          }
        />

        <DataTable
          {...tableProps}
          columns={buildPipelineColumns({
            runningId,
            mutatingId,
            onRun: (pipeline) => void runPipeline(pipeline),
            onPause: (pipeline) => void pausePipeline(pipeline),
            onResume: (pipeline) => void resumePipeline(pipeline),
            onDelete: (pipeline) => void deletePipeline(pipeline),
          })}
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

        <CreatePipelineWizard
          open={wizardOpen}
          onOpenChange={setWizardOpen}
          onCreated={() => {
            void refetch();
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
