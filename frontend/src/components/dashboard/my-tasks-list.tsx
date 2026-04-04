'use client';

import Link from 'next/link';
import { useState } from 'react';
import { CheckCircle, ArrowRight, CheckCircle2 } from 'lucide-react';
import { motion } from 'framer-motion';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, cn } from '@/lib/utils';
import { isAfter, parseISO } from 'date-fns';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import type { PaginatedResponse } from '@/types/api';
import type { WorkflowTask } from '@/types/models';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { HighlightAnimation } from '@/components/realtime/highlight-animation';
import { showNewDataToast } from '@/components/realtime/new-data-toast';

const STATUS_COLORS: Record<string, string> = {
  pending: '#94A3B8',
  claimed: '#3B82F6',
  overdue: '#EF4444',
};

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
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.35 }}
      className="flex flex-col rounded-2xl border border-border/60"
      style={{
        background: 'rgba(255, 255, 255, 0.6)',
        backdropFilter: 'blur(24px)',
        WebkitBackdropFilter: 'blur(24px)',
      }}
    >
      <div className="flex items-center justify-between border-b border-border/60 px-5 py-4">
        <div className="flex items-center gap-2.5">
          <CheckSquareIcon className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold">My Tasks</h3>
          {data && data.data.length > 0 && (
            <span className="inline-flex min-w-[20px] items-center justify-center rounded-full bg-blue-100 px-1.5 py-0.5 text-[11px] font-bold text-blue-700">
              {data.data.length}
            </span>
          )}
        </div>
        <Link
          href="/workflows/tasks"
          className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
        >
          View all
          <ArrowRight className="h-3 w-3" />
        </Link>
      </div>

      {isLoading ? (
        <div className="p-4">
          <LoadingSkeleton variant="list-item" count={5} />
        </div>
      ) : error ? (
        <ErrorState message="Failed to load tasks" onRetry={() => void mutate()} />
      ) : !data || data.data.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 px-4 py-10">
          <div
            className="flex h-12 w-12 items-center justify-center rounded-full"
            style={{ backgroundColor: 'rgba(34, 197, 94, 0.08)' }}
          >
            <CheckCircle2 className="h-[22px] w-[22px] text-green-500" />
          </div>
          <div className="text-center">
            <p className="text-sm font-medium text-muted-foreground">All caught up!</p>
            <p className="mt-0.5 text-xs text-muted-foreground/70">No pending tasks.</p>
          </div>
        </div>
      ) : (
        <div className="divide-y divide-border/30">
          {data.data.map((task) => {
            const isOverdue = task.due_at ? isAfter(new Date(), parseISO(task.due_at)) : false;
            const effectiveStatus = isOverdue ? 'overdue' : task.status;
            const dotColor = STATUS_COLORS[effectiveStatus] ?? '#94A3B8';

            return (
              <HighlightAnimation
                key={task.id}
                highlight={highlightedTaskId === task.id}
                highlightKey={highlightedTaskId === task.id ? task.id : null}
              >
                <Link
                  href={`/workflows/tasks/${task.id}`}
                  className="flex items-start gap-3 px-5 py-3 transition-colors hover:bg-muted/20"
                >
                  {/* Severity/status dot */}
                  <div className="mt-1.5 flex-shrink-0">
                    <div
                      className="h-2.5 w-2.5 rounded-full"
                      style={{
                        backgroundColor: dotColor,
                        boxShadow: isOverdue ? `0 0 8px ${dotColor}80` : 'none',
                      }}
                    />
                  </div>
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
                  <span className={statusBadge(effectiveStatus)}>
                    {effectiveStatus}
                  </span>
                </Link>
              </HighlightAnimation>
            );
          })}
        </div>
      )}
    </motion.div>
  );
}

function CheckSquareIcon({ className }: { className?: string }) {
  return <CheckCircle className={className} />;
}

function getTaskId(actionUrl?: string | null): string | null {
  if (!actionUrl) {
    return null;
  }
  const parts = actionUrl.split('/');
  return parts[parts.length - 1] ?? null;
}
