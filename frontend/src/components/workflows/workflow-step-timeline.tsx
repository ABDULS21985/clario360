'use client';

import { useState } from 'react';
import { CheckCircle, XCircle, Circle, Minus, ChevronDown, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatDateTime } from '@/lib/utils';
import { getStepIcon, formatStepType } from '@/lib/workflow-utils';
import { Badge } from '@/components/ui/badge';
import type { StepExecution, StepDefinition } from '@/types/models';

interface WorkflowStepTimelineProps {
  steps: StepExecution[];
  currentStepId: string | null;
  definitionSteps: StepDefinition[];
}

function StepIndicator({ status, isCurrent }: { status: string; isCurrent: boolean }) {
  if (isCurrent) {
    return (
      <div className="relative flex h-6 w-6 items-center justify-center">
        <span className="absolute inline-flex h-6 w-6 animate-ping rounded-full bg-blue-400 opacity-50" />
        <span className="relative h-3.5 w-3.5 rounded-full bg-blue-500" />
      </div>
    );
  }
  if (status === 'completed') {
    return <CheckCircle className="h-5 w-5 text-green-500" />;
  }
  if (status === 'failed') {
    return <XCircle className="h-5 w-5 text-red-500" />;
  }
  if (status === 'skipped') {
    return <Minus className="h-5 w-5 text-gray-400" />;
  }
  return <Circle className="h-5 w-5 text-gray-300" />;
}

interface StepRowProps {
  step: StepDefinition;
  execution?: StepExecution;
  isCurrent: boolean;
  isLast: boolean;
}

function StepRow({ step, execution, isCurrent, isLast }: StepRowProps) {
  const [expanded, setExpanded] = useState(false);
  const StepIcon = getStepIcon(step.type);
  const status = isCurrent ? 'running' : execution?.status ?? 'pending';

  const lineColor = cn(
    'absolute left-2.5 top-6 -ml-px w-0.5',
    !isLast && 'h-full',
    status === 'completed' && 'bg-green-300',
    isCurrent && 'bg-blue-300',
    status === 'pending' && 'bg-gray-200 bg-dashed',
  );

  const hasOutput = execution?.output && Object.keys(execution.output).length > 0;

  return (
    <div className="relative flex gap-3 pb-4">
      {!isLast && <div className={lineColor} />}

      <div className="relative z-10 mt-0.5 shrink-0">
        <StepIndicator status={status} isCurrent={isCurrent} />
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <div>
            <div className="flex items-center gap-1.5">
              <StepIcon className="h-3.5 w-3.5 text-muted-foreground" />
              <span
                className={cn(
                  'text-sm font-medium',
                  isCurrent && 'text-blue-700',
                  status === 'failed' && 'text-red-700',
                  status === 'pending' && 'text-muted-foreground',
                )}
              >
                {step.name}
              </span>
              {isCurrent && (
                <Badge variant="secondary" className="text-xs bg-blue-100 text-blue-700">
                  In Progress
                </Badge>
              )}
            </div>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {formatStepType(step.type)}
            </p>
          </div>

          {hasOutput && (
            <button
              onClick={() => setExpanded(!expanded)}
              className="shrink-0 p-0.5 rounded hover:bg-muted"
            >
              {expanded ? (
                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
              )}
            </button>
          )}
        </div>

        {/* Status details */}
        {execution && (
          <div className="mt-1 text-xs text-muted-foreground">
            {status === 'completed' && execution.completed_at && (
              <span>
                Completed
                {execution.completed_by ? ` by ${execution.completed_by}` : ''}
                {execution.duration_seconds
                  ? ` · ${Math.round(execution.duration_seconds / 60)}min`
                  : ''}
                {' · '}
                {formatDateTime(execution.completed_at)}
              </span>
            )}
            {isCurrent && (
              <span>
                In progress
                {execution.assigned_to ? ` · Claimed by ${execution.assigned_to}` : ' · Waiting for assignment'}
              </span>
            )}
            {status === 'failed' && execution.error && (
              <span className="text-red-600">Failed: {execution.error}</span>
            )}
          </div>
        )}

        {!execution && !isCurrent && (
          <p className="mt-0.5 text-xs text-muted-foreground">Pending</p>
        )}

        {/* Expanded output */}
        {expanded && hasOutput && (
          <div className="mt-2 rounded border bg-muted/50 p-2">
            <pre className="overflow-auto text-xs text-muted-foreground whitespace-pre-wrap">
              {JSON.stringify(execution?.output, null, 2)}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}

export function WorkflowStepTimeline({
  steps,
  currentStepId,
  definitionSteps,
}: WorkflowStepTimelineProps) {
  if (definitionSteps.length === 0 && steps.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No steps available.</p>
    );
  }

  // Build display list from definition steps, annotated with execution data
  const execByStepId = new Map(steps.map((s) => [s.step_id, s]));

  const displaySteps =
    definitionSteps.length > 0
      ? definitionSteps
      : steps.map((s) => ({ id: s.step_id, name: s.step_name, type: s.step_type }));

  return (
    <div className="space-y-0">
      {displaySteps.map((defStep, idx) => (
        <StepRow
          key={defStep.id}
          step={defStep}
          execution={execByStepId.get(defStep.id)}
          isCurrent={defStep.id === currentStepId}
          isLast={idx === displaySteps.length - 1}
        />
      ))}
    </div>
  );
}
