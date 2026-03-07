'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  dateColumn,
  actionsColumn,
  statusColumn,
} from '@/components/shared/data-table/columns/common-columns';
import { taskStatusConfig } from '@/lib/status-configs';
import { cn } from '@/lib/utils';
import {
  canClaimTask,
  canDelegateTask,
  formatSLAStatus,
  PRIORITY_LABELS,
  PRIORITY_COLORS,
} from '@/lib/workflow-utils';
import type { HumanTask, User } from '@/types/models';

interface TaskColumnOptions {
  onOpen: (task: HumanTask) => void;
  onClaim: (task: HumanTask) => void;
  onDelegate: (task: HumanTask) => void;
  onViewWorkflow: (task: HumanTask) => void;
  currentUser?: User | null;
}

function PriorityCell({ priority }: { priority: number }) {
  const label = PRIORITY_LABELS[priority] ?? 'Normal';
  const color = PRIORITY_COLORS[priority] ?? 'bg-blue-400';
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn('block h-2.5 w-2.5 rounded-full', color)} aria-label={label} />
      </TooltipTrigger>
      <TooltipContent>
        <p className="text-xs">{label}</p>
      </TooltipContent>
    </Tooltip>
  );
}

function SLACell({ task }: { task: HumanTask }) {
  const { text, color } = formatSLAStatus(task);
  const deadlineText = task.sla_deadline ? new Date(task.sla_deadline).toLocaleString() : 'None';

  if (task.sla_breached) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="space-y-1">
            <Badge variant="destructive" className="text-xs">
              Overdue
            </Badge>
            <p className="text-xs text-destructive">{text}</p>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <p className="text-xs">Deadline: {deadlineText}</p>
        </TooltipContent>
      </Tooltip>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn('text-xs', color)}>{text}</span>
      </TooltipTrigger>
      <TooltipContent>
        <p className="text-xs">Deadline: {deadlineText}</p>
      </TooltipContent>
    </Tooltip>
  );
}

function AssignedCell({
  task,
  currentUser,
  onClaim,
}: {
  task: HumanTask;
  currentUser?: User | null;
  onClaim: (task: HumanTask) => void;
}) {
  const isMe = task.claimed_by === currentUser?.id;

  if (!task.claimed_by) {
    const canClaim = canClaimTask(task, currentUser);
    return (
      <div className="flex items-center gap-2">
        <Badge variant="outline" className="border-orange-300 text-xs text-orange-600">
          Unassigned
        </Badge>
        {canClaim && (
          <Button
            size="sm"
            variant="ghost"
            className="h-5 px-1 text-xs"
            onClick={(event) => {
              event.stopPropagation();
              onClaim(task);
            }}
          >
            Claim
          </Button>
        )}
      </div>
    );
  }

  if (isMe) {
    return (
      <div className="flex items-center gap-1.5">
        <span className="h-1.5 w-1.5 rounded-full bg-green-500" />
        <span className="text-xs font-medium">You</span>
      </div>
    );
  }

  return (
    <span className="text-xs text-muted-foreground">
      {task.claimed_by_name ?? task.claimed_by}
    </span>
  );
}

export function getTaskColumns(options: TaskColumnOptions): ColumnDef<HumanTask>[] {
  const { onOpen, onClaim, onDelegate, onViewWorkflow, currentUser } = options;

  return [
    {
      id: 'priority',
      accessorKey: 'priority',
      header: '',
      cell: ({ row }) => <PriorityCell priority={row.original.priority} />,
      enableSorting: true,
      size: 50,
    },
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Task Name',
      cell: ({ row }) => {
        const task = row.original;
        return (
          <div
            className="cursor-pointer"
            onClick={() => onOpen(task)}
            role="button"
            tabIndex={0}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                onOpen(task);
              }
            }}
          >
            <span className="block text-sm font-medium">{task.name}</span>
            <span className="block text-xs text-muted-foreground">
              {task.description.length > 80
                ? `${task.description.slice(0, 80)}...`
                : task.description || '—'}
            </span>
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'workflow_name',
      accessorKey: 'workflow_name',
      header: 'Workflow',
      cell: ({ row }) => {
        const value = row.original.workflow_name || row.original.definition_name;
        if (!value) {
          return <span className="text-muted-foreground">—</span>;
        }
        return (
          <Badge variant="outline" className="text-xs">
            {value}
          </Badge>
        );
      },
      size: 160,
    },
    statusColumn<HumanTask>('status', 'Status', taskStatusConfig),
    {
      id: 'sla_deadline',
      accessorKey: 'sla_deadline',
      header: 'Due Date',
      cell: ({ row }) => <SLACell task={row.original} />,
      enableSorting: true,
      size: 130,
    },
    {
      id: 'claimed_by',
      accessorKey: 'claimed_by',
      header: 'Assigned',
      cell: ({ row }) => (
        <AssignedCell task={row.original} currentUser={currentUser} onClaim={onClaim} />
      ),
      size: 140,
    },
    dateColumn<HumanTask>('created_at', 'Created', { relative: true }),
    actionsColumn<HumanTask>((task) => [
      {
        label: 'Open Task',
        onClick: () => onOpen(task),
      },
      ...(canClaimTask(task, currentUser)
        ? [{ label: 'Claim', onClick: () => onClaim(task) }]
        : []),
      ...(canDelegateTask(task, currentUser)
        ? [{ label: 'Delegate', onClick: () => onDelegate(task) }]
        : []),
      {
        label: 'View Workflow',
        onClick: () => onViewWorkflow(task),
      },
    ]),
  ];
}
