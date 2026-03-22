'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, XCircle, RotateCcw, PauseCircle, PlayCircle, Loader2 } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/shared/status-badge';
import { WorkflowInstanceDetail } from '@/components/workflows/workflow-instance-detail';
import { WorkflowCancelDialog } from '@/components/workflows/workflow-cancel-dialog';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { workflowStatusConfig } from '@/lib/status-configs';
import { formatDateTime } from '@/lib/utils';
import { showSuccess, showError } from '@/lib/toast';
import type { WorkflowInstance, StepExecution } from '@/types/models';

export function WorkflowInstancePageClient() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const instanceId = (params?.id as string | undefined) ?? '';
  const [cancelOpen, setCancelOpen] = useState(false);

  const { data: instance, isLoading, isError, refetch } = useQuery({
    queryKey: ['workflow-instances', instanceId],
    queryFn: () => apiGet<WorkflowInstance>(`/api/v1/workflows/instances/${instanceId}`),
  });

  const { data: history, isLoading: historyLoading } = useQuery({
    queryKey: ['workflow-instances', instanceId, 'history'],
    queryFn: async () => {
      const resp = await apiGet<{ instance_id: string; step_executions: StepExecution[] }>(
        `/api/v1/workflows/instances/${instanceId}/history`,
      );
      return { steps: resp.step_executions ?? [] };
    },
    enabled: !!instanceId,
  });

  const retryMutation = useMutation({
    mutationFn: () => apiPost(`/api/v1/workflows/instances/${instanceId}/retry`),
    onSuccess: () => {
      showSuccess('Workflow retry initiated.');
      queryClient.invalidateQueries({ queryKey: ['workflow-instances', instanceId] });
    },
    onError: () => showError('Failed to retry workflow.'),
  });

  const suspendMutation = useMutation({
    mutationFn: () => apiPost(`/api/v1/workflows/instances/${instanceId}/suspend`),
    onSuccess: () => {
      showSuccess('Workflow suspended.');
      queryClient.invalidateQueries({ queryKey: ['workflow-instances', instanceId] });
    },
    onError: () => showError('Failed to suspend workflow.'),
  });

  const resumeMutation = useMutation({
    mutationFn: () => apiPost(`/api/v1/workflows/instances/${instanceId}/resume`),
    onSuccess: () => {
      showSuccess('Workflow resumed.');
      queryClient.invalidateQueries({ queryKey: ['workflow-instances', instanceId] });
    },
    onError: () => showError('Failed to resume workflow.'),
  });

  if (isLoading || historyLoading) {
    return (
      <div className="space-y-6">
        <LoadingSkeleton variant="card" count={3} />
      </div>
    );
  }

  if (isError || !instance) {
    return (
      <ErrorState
        message="Failed to load workflow instance"
        onRetry={() => refetch()}
      />
    );
  }

  const isRunning = instance.status === 'running';
  const isFailed = instance.status === 'failed';
  const isSuspended = instance.status === 'suspended';

  return (
    <div className="space-y-6">
      <div>
        <button
          onClick={() => router.push('/workflows')}
          className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
          type="button"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Workflows
        </button>
      </div>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{instance.definition_name ?? 'Workflow Instance'}</h1>
          <div className="mt-1.5 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            <StatusBadge status={instance.status} config={workflowStatusConfig} />
            <span>Started {formatDateTime(instance.started_at)}</span>
            {instance.started_by_name && <span>by {instance.started_by_name}</span>}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isRunning && (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={() => suspendMutation.mutate()}
                disabled={suspendMutation.isPending}
              >
                {suspendMutation.isPending ? (
                  <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <PauseCircle className="mr-1 h-3.5 w-3.5" />
                )}
                Suspend
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setCancelOpen(true)}
              >
                <XCircle className="mr-1 h-3.5 w-3.5" />
                Cancel Workflow
              </Button>
            </>
          )}
          {isFailed && (
            <Button
              size="sm"
              onClick={() => retryMutation.mutate()}
              disabled={retryMutation.isPending}
            >
              {retryMutation.isPending ? (
                <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
              ) : (
                <RotateCcw className="mr-1 h-3.5 w-3.5" />
              )}
              Retry
            </Button>
          )}
          {isSuspended && (
            <Button
              size="sm"
              onClick={() => resumeMutation.mutate()}
              disabled={resumeMutation.isPending}
            >
              {resumeMutation.isPending ? (
                <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
              ) : (
                <PlayCircle className="mr-1 h-3.5 w-3.5" />
              )}
              Resume
            </Button>
          )}
        </div>
      </div>

      <WorkflowInstanceDetail
        instance={instance}
        history={history?.steps ?? []}
      />

      <WorkflowCancelDialog
        instanceId={instanceId}
        definitionName={instance.definition_name ?? 'Workflow'}
        open={cancelOpen}
        onOpenChange={setCancelOpen}
        onSuccess={() => {
          setCancelOpen(false);
          queryClient.invalidateQueries({ queryKey: ['workflow-instances', instanceId] });
        }}
      />
    </div>
  );
}
