'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import {
  dateColumn,
  statusColumn,
  actionsColumn,
} from '@/components/shared/data-table/columns/common-columns';
import { workflowStatusConfig } from '@/lib/status-configs';
import { formatDuration } from '@/lib/format';
import type { WorkflowInstance } from '@/types/models';

interface WorkflowInstanceColumnOptions {
  onView: (instance: WorkflowInstance) => void;
  onCancel: (instance: WorkflowInstance) => void;
  onRetry: (instance: WorkflowInstance) => void;
}

function CurrentStepCell({ instance }: { instance: WorkflowInstance }) {
  if (instance.status === 'completed') {
    return (
      <span className="text-sm text-green-700">
        Completed ({instance.total_steps} steps)
      </span>
    );
  }
  if (instance.status === 'failed') {
    return (
      <span className="text-sm text-destructive">
        Failed at: {instance.current_step_name ?? 'Unknown step'}
      </span>
    );
  }
  if (instance.current_step_name) {
    const stepNum = instance.completed_steps + 1;
    const total = instance.total_steps;
    return (
      <div>
        <span className="text-sm font-medium">{instance.current_step_name}</span>
        <span className="ml-1.5 text-xs text-muted-foreground">
          Step {stepNum} of {total}
        </span>
      </div>
    );
  }
  return <span className="text-muted-foreground text-sm">—</span>;
}

function DurationCell({ instance }: { instance: WorkflowInstance }) {
  const startTime = new Date(instance.started_at).getTime();
  const endTime = instance.completed_at
    ? new Date(instance.completed_at).getTime()
    : Date.now();
  const seconds = Math.floor((endTime - startTime) / 1000);
  return <span className="text-sm text-muted-foreground">{formatDuration(seconds)}</span>;
}

function StartedByCell({ instance }: { instance: WorkflowInstance }) {
  if (!instance.started_by) {
    return <Badge variant="secondary" className="text-xs">System</Badge>;
  }
  return <span className="text-sm">{instance.started_by_name ?? instance.started_by}</span>;
}

export function getWorkflowInstanceColumns(
  options: WorkflowInstanceColumnOptions,
): ColumnDef<WorkflowInstance>[] {
  const { onView, onCancel, onRetry } = options;

  return [
    {
      id: 'definition_name',
      accessorKey: 'definition_name',
      header: 'Workflow',
      cell: ({ getValue, row }) => {
        const name = getValue() as string;
        return (
          <button
            onClick={() => onView(row.original)}
            className="text-sm font-medium text-left hover:underline"
          >
            {name}
          </button>
        );
      },
      enableSorting: true,
    },
    {
      id: 'current_step',
      header: 'Current Step',
      cell: ({ row }) => <CurrentStepCell instance={row.original} />,
      size: 200,
      enableSorting: false,
    },
    statusColumn<WorkflowInstance>('status', 'Status', workflowStatusConfig),
    dateColumn<WorkflowInstance>('started_at', 'Started', { relative: true }),
    {
      id: 'duration',
      header: 'Duration',
      cell: ({ row }) => <DurationCell instance={row.original} />,
      size: 100,
      enableSorting: false,
    },
    {
      id: 'started_by',
      header: 'Started By',
      cell: ({ row }) => <StartedByCell instance={row.original} />,
      size: 140,
      enableSorting: false,
    },
    actionsColumn<WorkflowInstance>((instance) => [
      { label: 'View Details', onClick: () => onView(instance) },
      ...(instance.status === 'running'
        ? [{ label: 'Cancel', onClick: () => onCancel(instance), variant: 'destructive' as const }]
        : []),
      ...(instance.status === 'failed'
        ? [{ label: 'Retry', onClick: () => onRetry(instance) }]
        : []),
    ]),
  ];
}
