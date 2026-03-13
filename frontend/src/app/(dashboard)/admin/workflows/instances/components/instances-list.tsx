'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Play } from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { DataTable } from '@/components/shared/data-table/data-table';
import { WorkflowCancelDialog } from '@/components/workflows/workflow-cancel-dialog';
import { ErrorState } from '@/components/common/error-state';
import { useDataTable } from '@/hooks/use-data-table';
import { workflowInstanceFilters } from '@/components/workflows/workflow-instance-filters';
import { SearchInput } from '@/components/shared/forms/search-input';
import { StartWorkflowDialog } from './start-workflow-dialog';
import { getAdminInstanceColumns } from './admin-instance-columns';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import {
  usePauseWorkflowInstance,
  useResumeWorkflowInstance,
  useRetryWorkflowInstance,
} from '@/hooks/use-workflow-instances-ext';
import type { WorkflowInstance } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

export function InstancesList() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [cancelTarget, setCancelTarget] = useState<WorkflowInstance | null>(null);
  const [startOpen, setStartOpen] = useState(false);

  const pauseMutation = usePauseWorkflowInstance();
  const resumeMutation = useResumeWorkflowInstance();
  const retryMutation = useRetryWorkflowInstance();

  const table = useDataTable<WorkflowInstance>({
    queryKey: 'workflow-instances',
    defaultPageSize: 25,
    defaultSort: { column: 'started_at', direction: 'desc' },
    fetchFn: (params) => {
      const startedRange =
        typeof params.filters?.started_at === 'string'
          ? params.filters.started_at.split(',')
          : [];

      return apiGet<PaginatedResponse<WorkflowInstance>>(
        API_ENDPOINTS.WORKFLOWS_INSTANCES,
        {
          page: params.page,
          per_page: params.per_page,
          sort: params.sort ?? 'started_at',
          order: params.order ?? 'desc',
          search: params.search,
          ...(params.filters?.status
            ? {
                status: Array.isArray(params.filters.status)
                  ? params.filters.status.join(',')
                  : params.filters.status,
              }
            : {}),
          ...(startedRange[0] ? { date_from: startedRange[0] } : {}),
          ...(startedRange[1] ? { date_to: startedRange[1] } : {}),
        },
      );
    },
  });

  const columns = getAdminInstanceColumns({
    onView: (instance) => router.push(`/admin/workflows/instances/${instance.id}`),
    onCancel: (instance) => setCancelTarget(instance),
    onRetry: (instance) => retryMutation.mutate(instance.id),
    onPause: (instance) => pauseMutation.mutate(instance.id),
    onResume: (instance) => resumeMutation.mutate(instance.id),
  });

  if (table.error) {
    return (
      <div className="space-y-6">
        <PageHeader title="Workflow Instances" description="Manage all workflow instances" />
        <ErrorState message="Failed to load instances" onRetry={() => table.refetch()} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Workflow Instances"
        description="Monitor and manage all workflow instances across the tenant."
        actions={
          <Button size="sm" onClick={() => setStartOpen(true)}>
            <Play className="mr-1.5 h-3.5 w-3.5" />
            Start Workflow
          </Button>
        }
      />

      <DataTable
        columns={columns}
        filters={workflowInstanceFilters}
        searchSlot={
          <SearchInput
            value={table.searchValue}
            onChange={table.setSearch}
            placeholder="Search instances..."
          />
        }
        {...table.tableProps}
        onRowClick={(row) => router.push(`/admin/workflows/instances/${row.id}`)}
      />

      {cancelTarget && (
        <WorkflowCancelDialog
          instanceId={cancelTarget.id}
          definitionName={cancelTarget.definition_name}
          open={Boolean(cancelTarget)}
          onOpenChange={(open) => {
            if (!open) setCancelTarget(null);
          }}
          onSuccess={() => {
            setCancelTarget(null);
            queryClient.invalidateQueries({ queryKey: ['workflow-instances'] });
          }}
        />
      )}

      <StartWorkflowDialog open={startOpen} onOpenChange={setStartOpen} />
    </div>
  );
}
