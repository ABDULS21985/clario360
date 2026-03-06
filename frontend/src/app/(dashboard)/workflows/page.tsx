'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost } from '@/lib/api';
import { PageHeader } from '@/components/common/page-header';
import { DataTable } from '@/components/shared/data-table/data-table';
import { WorkflowCancelDialog } from '@/components/workflows/workflow-cancel-dialog';
import { getWorkflowInstanceColumns } from '@/components/workflows/workflow-instance-columns';
import { ErrorState } from '@/components/common/error-state';
import { showSuccess, showError } from '@/lib/toast';
import type { WorkflowInstance } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

export default function WorkflowsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [cancelTarget, setCancelTarget] = useState<WorkflowInstance | null>(null);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['workflows', 'instances', page],
    queryFn: () =>
      apiGet<PaginatedResponse<WorkflowInstance>>('/api/v1/workflows/instances', {
        page,
        per_page: 25,
        sort: 'started_at',
        order: 'desc',
      }),
  });

  const retryMutation = useMutation({
    mutationFn: (instanceId: string) =>
      apiPost(`/api/v1/workflows/instances/${instanceId}/retry`),
    onSuccess: () => {
      showSuccess('Workflow retry initiated.');
      queryClient.invalidateQueries({ queryKey: ['workflows', 'instances'] });
    },
    onError: () => showError('Failed to retry workflow.'),
  });

  const columns = getWorkflowInstanceColumns({
    onView: (instance) => router.push(`/workflows/${instance.id}`),
    onCancel: (instance) => setCancelTarget(instance),
    onRetry: (instance) => retryMutation.mutate(instance.id),
  });

  if (isError) {
    return (
      <div className="space-y-6">
        <PageHeader title="Workflows" description="Monitor workflow instances" />
        <ErrorState message="Failed to load workflows" onRetry={() => refetch()} />
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
        data={data?.data ?? []}
        totalRows={data?.meta.total ?? 0}
        page={page}
        pageSize={25}
        onPageChange={setPage}
        onPageSizeChange={() => undefined}
        onSortChange={() => undefined}
        isLoading={isLoading}
        onRowClick={(row) => router.push(`/workflows/${row.id}`)}
      />

      {cancelTarget && (
        <WorkflowCancelDialog
          instanceId={cancelTarget.id}
          definitionName={cancelTarget.definition_name}
          open={!!cancelTarget}
          onOpenChange={(open) => {
            if (!open) setCancelTarget(null);
          }}
          onSuccess={() => {
            setCancelTarget(null);
            queryClient.invalidateQueries({ queryKey: ['workflows', 'instances'] });
          }}
        />
      )}
    </div>
  );
}
