'use client';

import Link from 'next/link';
import { useState } from 'react';
import { CheckCircle } from 'lucide-react';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, cn } from '@/lib/utils';
import { isAfter, parseISO } from 'date-fns';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import type { PaginatedResponse } from '@/types/api';
import type { WorkflowTask } from '@/types/models';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { HighlightAnimation } from '@/components/realtime/highlight-animation';
import { showNewDataToast } from '@/components/realtime/new-data-toast';

function statusBadge(status: string): string {
  return cn(
    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
    status === 'pending' && 'bg-muted text-muted-foreground',
    status === 'claimed' && 'bg-blue-100 text-blue-800',
    status === 'overdue' && 'bg-destructive/10 text-destructive',
  );
}

export function MyTasksList() {
  const [highlightedTaskId, setHighlightedTaskId] = useState<string | null>(null);
  const { data, isLoading, error, mutate } = useRealtimeData<PaginatedResponse<WorkflowTask>>(
    API_ENDPOINTS.WORKFLOWS_TASKS,
    {
      params: { status: 'pending,claimed', per_page: 5 },
      wsTopics: [
        'task.assigned',
        'task.completed',
        'task.escalated',
        'workflow.task.created',
        'workflow.task.completed',
        'workflow.task.escalated',
      ],
      onNewItem: (notification) => {
        const taskId = getTaskId(notification.action_url);
        if (taskId) {
          setHighlightedTaskId(taskId);
          window.setTimeout(() => setHighlightedTaskId(null), 3000);
        }
        showNewDataToast({
          title: notification.title,
          description: notification.body,
        });
      },
    },
  );

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
      ) : error ? (
        <ErrorState message="Failed to load tasks" onRetry={() => void mutate()} />
      ) : !data || data.data.length === 0 ? (
        <EmptyState
          icon={CheckCircle}
          title="No pending tasks"
          description="You're all caught up!"
        />
      ) : (
        <div className="divide-y">
          {data.data.map((task) => {
            const isOverdue = task.due_at ? isAfter(new Date(), parseISO(task.due_at)) : false;
            return (
              <HighlightAnimation
                key={task.id}
                highlight={highlightedTaskId === task.id}
                highlightKey={highlightedTaskId === task.id ? task.id : null}
              >
                <Link
                  href={`/workflows/tasks/${task.id}`}
                  className="flex items-start gap-3 px-4 py-3 transition-colors hover:bg-muted/30"
                >
                  <CheckCircle className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{task.name}</p>
                    <p className="truncate text-xs text-muted-foreground">{task.workflow_name}</p>
                    {task.due_at && (
                      <p
                        className={cn(
                          'mt-0.5 text-xs',
                          isOverdue ? 'font-medium text-destructive' : 'text-muted-foreground',
                        )}
                      >
                        Due {isOverdue ? '(overdue) ' : ''}
                        {formatDate(task.due_at)}
                      </p>
                    )}
                  </div>
                  <span className={statusBadge(isOverdue ? 'overdue' : task.status)}>
                    {isOverdue ? 'overdue' : task.status}
                  </span>
                </Link>
              </HighlightAnimation>
            );
          })}
        </div>
      )}
    </div>
  );
}

function getTaskId(actionUrl?: string | null): string | null {
  if (!actionUrl) {
    return null;
  }
  const parts = actionUrl.split('/');
  return parts[parts.length - 1] ?? null;
}
