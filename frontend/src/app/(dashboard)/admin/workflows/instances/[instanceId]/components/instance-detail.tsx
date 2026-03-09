'use client';

import { useParams, useRouter } from 'next/navigation';
import {
  ArrowLeft,
  XCircle,
  RotateCcw,
  PauseCircle,
  PlayCircle,
  Loader2,
  AlertTriangle,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { StatusBadge } from '@/components/shared/status-badge';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { workflowStatusConfig } from '@/lib/status-configs';
import { formatDateTime, formatDuration } from '@/lib/format';
import {
  useWorkflowInstance,
  useWorkflowInstanceHistory,
  usePauseWorkflowInstance,
  useResumeWorkflowInstance,
  useRetryWorkflowInstance,
} from '@/hooks/use-workflow-instances-ext';
import { useWorkflowDefinition } from '@/hooks/use-workflow-definitions';
import { WorkflowCanvas } from '../../../definitions/[defId]/designer/components/workflow-canvas';
import { StepHistory } from './step-history';
import { InstanceProgress } from './instance-progress';

export function InstanceDetailClient() {
  const params = useParams();
  const router = useRouter();
  const instanceId = (params?.instanceId as string | undefined) ?? '';

  const {
    data: instance,
    isLoading,
    isError,
    refetch,
  } = useWorkflowInstance(instanceId);

  const { data: historyData, isLoading: historyLoading } =
    useWorkflowInstanceHistory(instanceId);

  const { data: definition } = useWorkflowDefinition(
    instance?.definition_id ?? '',
  );

  const pauseMutation = usePauseWorkflowInstance();
  const resumeMutation = useResumeWorkflowInstance();
  const retryMutation = useRetryWorkflowInstance();

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

  const history = historyData?.steps ?? [];
  const isRunning = instance.status === 'running';
  const isFailed = instance.status === 'failed';
  const isSuspended = instance.status === 'suspended';

  // Build step status map for canvas overlay
  const stepStatuses: Record<string, 'completed' | 'running' | 'failed' | 'pending'> = {};
  for (const step of history) {
    if (step.status === 'completed') stepStatuses[step.step_id] = 'completed';
    else if (step.status === 'running') stepStatuses[step.step_id] = 'running';
    else if (step.status === 'failed') stepStatuses[step.step_id] = 'failed';
    else stepStatuses[step.step_id] = 'pending';
  }

  return (
    <div className="space-y-6">
      {/* Back */}
      <button
        onClick={() => router.push('/workflows')}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        type="button"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Workflows
      </button>

      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{instance.definition_name}</h1>
          <div className="mt-1.5 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            <StatusBadge
              status={instance.status}
              config={workflowStatusConfig}
            />
            <span>Started {formatDateTime(instance.started_at)}</span>
            {instance.started_by_name && (
              <span>by {instance.started_by_name}</span>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isRunning && (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={() => pauseMutation.mutate(instanceId)}
                disabled={pauseMutation.isPending}
              >
                {pauseMutation.isPending ? (
                  <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <PauseCircle className="mr-1 h-3.5 w-3.5" />
                )}
                Pause
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => router.push('/workflows')}
              >
                <XCircle className="mr-1 h-3.5 w-3.5" />
                Cancel
              </Button>
            </>
          )}
          {isFailed && (
            <Button
              size="sm"
              onClick={() => retryMutation.mutate(instanceId)}
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
              onClick={() => resumeMutation.mutate(instanceId)}
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

      {/* Error alert */}
      {isFailed && instance.error_message && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{instance.error_message}</AlertDescription>
        </Alert>
      )}

      {/* Progress bar */}
      <InstanceProgress instance={instance} />

      {/* Visual canvas (if definition loaded) */}
      {definition && (
        <Card className="overflow-hidden">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Workflow Progress
            </CardTitle>
          </CardHeader>
          <div className="h-[400px]">
            <WorkflowCanvas
              definition={definition}
              readOnly
              isSaving={false}
              isPublishing={false}
              stepStatuses={stepStatuses}
              onSave={() => undefined}
              onPublish={() => undefined}
            />
          </div>
        </Card>
      )}

      {/* Variables */}
      {Object.keys(instance.variables).length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Variables</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Object.entries(instance.variables).map(([key, value]) => (
                  <TableRow key={key}>
                    <TableCell className="font-mono text-xs">{key}</TableCell>
                    <TableCell className="text-xs font-mono">
                      {typeof value === 'object'
                        ? JSON.stringify(value)
                        : String(value)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Step execution history */}
      <StepHistory steps={history} />
    </div>
  );
}
