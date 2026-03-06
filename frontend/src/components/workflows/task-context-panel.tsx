'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { formatDateTime } from '@/lib/utils';
import { WorkflowStepTimeline } from './workflow-step-timeline';
import { Skeleton } from '@/components/ui/skeleton';
import { ErrorState } from '@/components/common/error-state';
import type { HumanTask, StepExecution, WorkflowInstance } from '@/types/models';

interface TaskContextPanelProps {
  task: HumanTask;
  instanceId: string;
}

function MetadataRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-2 py-1 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium">{value ?? '—'}</span>
    </div>
  );
}

export function TaskContextPanel({ task, instanceId }: TaskContextPanelProps) {
  const {
    data: instance,
    isLoading: instanceLoading,
    isError: instanceError,
    refetch: refetchInstance,
  } = useQuery({
    queryKey: ['workflow-instance', instanceId],
    queryFn: () => apiGet<WorkflowInstance>(`/api/v1/workflows/instances/${instanceId}`),
    enabled: !!instanceId,
  });

  const {
    data: history,
    isLoading: historyLoading,
    isError: historyError,
    refetch: refetchHistory,
  } = useQuery({
    queryKey: ['workflow-instance-history', instanceId],
    queryFn: () => apiGet<{ steps: StepExecution[] }>(`/api/v1/workflows/instances/${instanceId}/history`),
    enabled: !!instanceId,
  });

  return (
    <div className="space-y-6">
      {/* Workflow Progress */}
      <div>
        <h3 className="mb-3 text-sm font-semibold">Workflow Progress</h3>
        {instanceLoading || historyLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-start gap-3">
                <Skeleton className="h-5 w-5 rounded-full" />
                <div className="flex-1 space-y-1">
                  <Skeleton className="h-4 w-3/4" />
                  <Skeleton className="h-3 w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : instanceError || historyError ? (
          <ErrorState
            message="Failed to load workflow progress"
            onRetry={() => {
              refetchInstance();
              refetchHistory();
            }}
          />
        ) : (
          <WorkflowStepTimeline
            steps={history?.steps ?? []}
            currentStepId={instance?.current_step_id ?? null}
            definitionSteps={instance?.definition_steps ?? []}
          />
        )}
      </div>

      {/* Task Metadata */}
      <div>
        <h3 className="mb-2 text-sm font-semibold">Task Details</h3>
        <div className="divide-y rounded-lg border">
          <div className="px-3">
            <MetadataRow
              label="Created"
              value={formatDateTime(task.created_at)}
            />
            <MetadataRow
              label="SLA Deadline"
              value={
                task.sla_deadline ? (
                  <span className={task.sla_breached ? 'text-destructive' : undefined}>
                    {formatDateTime(task.sla_deadline)}
                    {task.sla_breached && ' (overdue)'}
                  </span>
                ) : (
                  'No deadline'
                )
              }
            />
            <MetadataRow
              label="Required Role"
              value={task.assignee_role ?? 'Any'}
            />
            {instance && (
              <MetadataRow
                label="Workflow"
                value={instance.definition_name}
              />
            )}
          </div>
        </div>
      </div>

      {/* Instance Variables (if available) */}
      {instance && Object.keys(instance.variables).length > 0 && (
        <div>
          <h3 className="mb-2 text-sm font-semibold">Context Variables</h3>
          <div className="rounded-lg border">
            <div className="divide-y px-3">
              {Object.entries(instance.variables).slice(0, 8).map(([key, val]) => (
                <MetadataRow
                  key={key}
                  label={key}
                  value={
                    <span className="font-mono text-xs">
                      {typeof val === 'object' ? JSON.stringify(val) : String(val ?? '—')}
                    </span>
                  }
                />
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
