'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost } from '@/lib/api';
import { PageHeader } from '@/components/common/page-header';
import { DataTable } from '@/components/shared/data-table/data-table';
import { WorkflowCancelDialog } from '@/components/workflows/workflow-cancel-dialog';
import { getWorkflowInstanceColumns } from '@/components/workflows/workflow-instance-columns';
import { ErrorState } from '@/components/common/error-state';
import { showSuccess, showApiError } from '@/lib/toast';
import { useDataTable } from '@/hooks/use-data-table';
import { workflowInstanceFilters } from '@/components/workflows/workflow-instance-filters';
import { SearchInput } from '@/components/shared/forms/search-input';
import type { WorkflowInstance } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

export function WorkflowsPageClient() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [cancelTarget, setCancelTarget] = useState<WorkflowInstance | null>(null);

  const workflowsTable = useDataTable<WorkflowInstance>({
    queryKey: 'workflow-instances',
    defaultPageSize: 25,
    defaultSort: { column: 'started_at', direction: 'desc' },
    fetchFn: (params) => {
      const startedRange =
        typeof params.filters?.started_at === 'string'
          ? params.filters.started_at.split(',')
          : [];

      return apiGet<PaginatedResponse<WorkflowInstance>>('/api/v1/workflows/instances', {
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
      });
    },
  });

  const retryMutation = useMutation({
    mutationFn: (instanceId: string) => apiPost(`/api/v1/workflows/instances/${instanceId}/retry`),
    onSuccess: () => {
      showSuccess('Workflow retry initiated.');
      queryClient.invalidateQueries({ queryKey: ['workflow-instances'] });
    },
    onError: (error) => showApiError(error),
  });

  const columns = getWorkflowInstanceColumns({
    onView: (instance) => router.push(`/workflows/${instance.id}`),
    onCancel: (instance) => setCancelTarget(instance),
    onRetry: (instance) => retryMutation.mutate(instance.id),
  });

  if (workflowsTable.error) {
    return (
      <div className="space-y-6">
        <PageHeader title="Workflows" description="Monitor workflow instances" />
        <ErrorState message="Failed to load workflows" onRetry={() => workflowsTable.refetch()} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Workflows"
        description="Monitor workflow instances across your organization."
      />

      <DataTable
        columns={columns}
        filters={workflowInstanceFilters}
        searchSlot={
          <SearchInput
            value={workflowsTable.searchValue}
            onChange={workflowsTable.setSearch}
            placeholder="Search workflows..."
          />
        }
        {...workflowsTable.tableProps}
        onRowClick={(row) => router.push(`/workflows/${row.id}`)}
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
    </div>
  );
}
