'use client';

import { useState } from 'react';
import Link from 'next/link';
import { CheckSquare } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, cn } from '@/lib/utils';
import { isAfter, parseISO } from 'date-fns';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Badge } from '@/components/ui/badge';
import type { PaginatedResponse } from '@/types/api';
import type { WorkflowTask } from '@/types/models';

export default function WorkflowTasksPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['workflows', 'tasks', page],
    queryFn: () =>
      apiGet<PaginatedResponse<WorkflowTask>>(API_ENDPOINTS.WORKFLOWS_TASKS, {
        page,
        per_page: 25,
        sort: 'created_at',
        order: 'desc',
      }),
    refetchInterval: 30000,
  });

  return (
    <div className="space-y-6">
      <PageHeader title="My Tasks" description="Tasks assigned to you" />

      {isLoading ? (
        <LoadingSkeleton variant="table-row" count={10} />
      ) : isError ? (
        <ErrorState message="Failed to load tasks" onRetry={() => refetch()} />
      ) : !data || data.data.length === 0 ? (
        <EmptyState icon={CheckSquare} title="No tasks" description="You have no assigned tasks." />
      ) : (
        <div className="rounded-lg border bg-card overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/30">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Task</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Workflow</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Due</th>
              </tr>
            </thead>
            <tbody>
              {data.data.map((task) => {
                const isOverdue = task.due_at ? isAfter(new Date(), parseISO(task.due_at)) : false;
                return (
                  <tr key={task.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3">
                      <Link href={`/workflows/tasks/${task.id}`} className="font-medium hover:underline">
                        {task.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground hidden md:table-cell">{task.workflow_name}</td>
                    <td className="px-4 py-3">
                      <Badge variant={
                        isOverdue ? 'destructive' :
                        task.status === 'claimed' ? 'default' : 'outline'
                      }>
                        {isOverdue ? 'overdue' : task.status}
                      </Badge>
                    </td>
                    <td className={cn('px-4 py-3 hidden lg:table-cell text-xs', isOverdue && 'text-destructive font-medium')}>
                      {task.due_at ? formatDate(task.due_at) : '—'}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {data.meta.total_pages > 1 && (
            <div className="flex items-center justify-between border-t px-4 py-3">
              <p className="text-xs text-muted-foreground">Page {page} of {data.meta.total_pages}</p>
              <div className="flex gap-2">
                <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}
                  className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent">Previous</button>
                <button disabled={page >= data.meta.total_pages} onClick={() => setPage((p) => p + 1)}
                  className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent">Next</button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
