'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/shared/status-badge';
import { actionItemStatusConfig } from '@/lib/status-configs';
import { daysOverdue } from '@/lib/enterprise';
import type { ActaActionItem } from '@/types/suites';

interface ActionItemColumnsOptions {
  onComplete: (item: ActaActionItem) => void;
  onExtend: (item: ActaActionItem) => void;
}

export function actionItemColumns({
  onComplete,
  onExtend,
}: ActionItemColumnsOptions): ColumnDef<ActaActionItem>[] {
  return [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Action Item',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.title}</p>
          <p className="text-xs text-muted-foreground">{row.original.assignee_name}</p>
        </div>
      ),
    },
    {
      id: 'meeting',
      header: 'Meeting',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{row.original.meeting_title ?? row.original.meeting_id.slice(0, 8)}</span>
      ),
    },
    {
      id: 'due_date',
      accessorKey: 'due_date',
      header: 'Due Date',
      enableSorting: true,
      cell: ({ row }) => {
        const item = row.original;
        const overdue = item.status === 'overdue' || daysOverdue(item.due_date) > 0;
        return (
          <div className={overdue ? 'text-destructive' : 'text-sm text-muted-foreground'}>
            <p>{item.due_date}</p>
            {overdue ? (
              <p className="text-xs">
                {daysOverdue(item.due_date)} day{daysOverdue(item.due_date) === 1 ? '' : 's'} overdue
              </p>
            ) : null}
          </div>
        );
      },
    },
    {
      id: 'priority',
      accessorKey: 'priority',
      header: 'Priority',
      cell: ({ row }) => <Badge variant="outline" className="capitalize">{row.original.priority}</Badge>,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => <StatusBadge status={row.original.status} config={actionItemStatusConfig} size="sm" />,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex justify-end gap-2">
          {row.original.status !== 'completed' ? (
            <Button variant="outline" size="sm" onClick={() => onComplete(row.original)}>
              Complete
            </Button>
          ) : null}
          <Button variant="ghost" size="sm" onClick={() => onExtend(row.original)}>
            Extend
          </Button>
        </div>
      ),
    },
  ];
}
