'use client';

import Link from 'next/link';
import { CheckCircle } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, timeAgo, cn } from '@/lib/utils';
import { isAfter, parseISO } from 'date-fns';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import type { PaginatedResponse } from '@/types/api';
import type { WorkflowTask } from '@/types/models';

function statusBadge(status: string): string {
  return cn(
    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
    status === 'pending' && 'bg-muted text-muted-foreground',
    status === 'claimed' && 'bg-blue-100 text-blue-800',
    status === 'overdue' && 'bg-destructive/10 text-destructive',
  );
}

export function MyTasksList() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['dashboard', 'my-tasks'],
    queryFn: () =>
      apiGet<PaginatedResponse<WorkflowTask>>(API_ENDPOINTS.WORKFLOWS_TASKS, {
        status: 'pending,claimed',
        per_page: 5,
      }),
    refetchInterval: 30000,
  });

  return (
    <div className="rounded-lg border bg-card">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <h3 className="text-sm font-semibold">My Tasks</h3>
        <Link href="/workflows/tasks" className="text-xs text-primary hover:underline">
          View all →
        </Link>
      </div>

      {isLoading ? (
        <div className="p-4">
          <LoadingSkeleton variant="list-item" count={5} />
        </div>
      ) : isError ? (
        <ErrorState
          message="Failed to load tasks"
          onRetry={() => refetch()}
        />
      ) : !data || data.data.length === 0 ? (
        <EmptyState
          icon={CheckCircle}
          title="No pending tasks"
          description="You're all caught up!"
        />
      ) : (
        <div className="divide-y">
          {data.data.map((task) => {
            const isOverdue =
              task.due_at ? isAfter(new Date(), parseISO(task.due_at)) : false;
            return (
              <Link
                key={task.id}
                href={`/workflows/tasks/${task.id}`}
                className="flex items-start gap-3 px-4 py-3 hover:bg-muted/30 transition-colors"
              >
                <CheckCircle className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                <div className="flex-1 min-w-0">
                  <p className="truncate text-sm font-medium">{task.name}</p>
                  <p className="text-xs text-muted-foreground truncate">{task.workflow_name}</p>
                  {task.due_at && (
                    <p className={cn('text-xs mt-0.5', isOverdue ? 'text-destructive font-medium' : 'text-muted-foreground')}>
                      Due {isOverdue ? '(overdue) ' : ''}{formatDate(task.due_at)}
                    </p>
                  )}
                </div>
                <span className={statusBadge(isOverdue ? 'overdue' : task.status)}>
                  {isOverdue ? 'overdue' : task.status}
                </span>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
