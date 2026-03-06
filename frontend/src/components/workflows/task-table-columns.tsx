'use client';

import { useRouter } from 'next/navigation';
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
import { formatSLAStatus, PRIORITY_LABELS, PRIORITY_COLORS } from '@/lib/workflow-utils';
import { useAuth } from '@/hooks/use-auth';
import type { HumanTask } from '@/types/models';

interface TaskColumnOptions {
  onOpen: (task: HumanTask) => void;
  onClaim: (task: HumanTask) => void;
  onDelegate: (task: HumanTask) => void;
  currentUserId?: string;
}

function PriorityCell({ priority }: { priority: number }) {
  const label = PRIORITY_LABELS[priority] ?? 'Normal';
  const color = PRIORITY_COLORS[priority] ?? 'bg-blue-400';
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn('block h-2.5 w-2.5 rounded-full', color)} aria-label={label} />
      </TooltipTrigger>
      <TooltipContent><p className="text-xs">{label}</p></TooltipContent>
    </Tooltip>
  );
}

function SLACell({ task }: { task: HumanTask }) {
  const { text, color } = formatSLAStatus(task);
  if (task.sla_breached) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge variant="destructive" className="text-xs">
            {text}
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p className="text-xs">
            Deadline: {task.sla_deadline ? new Date(task.sla_deadline).toLocaleString() : '—'}
          </p>
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
        <p className="text-xs">
          Deadline: {task.sla_deadline ? new Date(task.sla_deadline).toLocaleString() : 'None'}
        </p>
      </TooltipContent>
    </Tooltip>
  );
}

function AssignedCell({
  task,
  currentUserId,
  onClaim,
}: {
  task: HumanTask;
  currentUserId?: string;
  onClaim: (task: HumanTask) => void;
}) {
  const { hasPermission } = useAuth();
  const isMe = task.claimed_by === currentUserId;

  if (!task.claimed_by) {
    const canClaim =
      !task.assignee_role || hasPermission(`${task.assignee_role}:*`) || hasPermission('*:*');
    return (
      <div className="flex items-center gap-2">
        <Badge variant="outline" className="text-xs text-orange-600 border-orange-300">
          Unassigned
        </Badge>
        {canClaim && (
          <Button
            size="sm"
            variant="ghost"
            className="h-5 px-1 text-xs"
            onClick={(e) => {
              e.stopPropagation();
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

  return <span className="text-xs text-muted-foreground">{task.claimed_by_name ?? task.claimed_by}</span>;
}

export function getTaskColumns(options: TaskColumnOptions): ColumnDef<HumanTask>[] {
  const { onOpen, onClaim, onDelegate, currentUserId } = options;

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
      header: 'Task',
      cell: ({ row }) => {
        const task = row.original;
        return (
          <div
            className="cursor-pointer"
            onClick={() => onOpen(task)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => e.key === 'Enter' && onOpen(task)}
          >
            <span className="block font-medium text-sm">{task.name}</span>
            {task.description && (
              <span className="block text-xs text-muted-foreground">
                {task.description.length > 80
                  ? `${task.description.slice(0, 80)}...`
                  : task.description}
              </span>
            )}
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'workflow_name',
      accessorKey: 'workflow_name',
      header: 'Workflow',
      cell: ({ getValue }) => {
        const val = getValue() as string | null;
        if (!val) return <span className="text-muted-foreground">—</span>;
        return <Badge variant="outline" className="text-xs">{val}</Badge>;
      },
      size: 160,
    },
    statusColumn<HumanTask>('status', 'Status', taskStatusConfig),
    {
      id: 'sla_deadline',
      accessorKey: 'sla_deadline',
      header: 'Due (SLA)',
      cell: ({ row }) => <SLACell task={row.original} />,
      enableSorting: true,
      size: 130,
    },
    {
      id: 'claimed_by',
      accessorKey: 'claimed_by',
      header: 'Assigned',
      cell: ({ row }) => (
        <AssignedCell
          task={row.original}
          currentUserId={currentUserId}
          onClaim={onClaim}
        />
      ),
      size: 140,
    },
    dateColumn<HumanTask>('created_at', 'Created', { relative: true }),
    actionsColumn<HumanTask>((task) => [
      {
        label: 'Open Task',
        onClick: () => onOpen(task),
      },
      ...(task.claimed_by === null
        ? [{ label: 'Claim', onClick: () => onClaim(task) }]
        : []),
      ...(task.claimed_by === currentUserId
        ? [{ label: 'Delegate', onClick: () => onDelegate(task) }]
        : []),
      {
        label: 'View Workflow',
        onClick: () => {
          if (typeof window !== 'undefined') {
            window.location.href = `/workflows/${task.instance_id}`;
          }
        },
      },
    ]),
  ];
}

// Re-export so pages can use this component
export function TaskTableColumnWrapper({
  task,
  onOpen,
  onClaim,
  onDelegate,
  currentUserId,
}: {
  task: HumanTask;
} & TaskColumnOptions) {
  const router = useRouter();
  void router;
  void task;
  void onOpen;
  void onClaim;
  void onDelegate;
  void currentUserId;
  return null;
}
