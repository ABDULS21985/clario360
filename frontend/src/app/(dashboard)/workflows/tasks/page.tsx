'use client';

import { useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { DataTable } from '@/components/shared/data-table/data-table';
import { TaskStatusTabs } from '@/components/workflows/task-status-tabs';
import { TaskDelegateDialog } from '@/components/workflows/task-delegate-dialog';
import { getTaskColumns } from '@/components/workflows/task-table-columns';
import { ErrorState } from '@/components/common/error-state';
import { useAuth } from '@/hooks/use-auth';
import type { HumanTask, TaskCounts } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

const TAB_PARAMS: Record<string, Record<string, unknown>> = {
  all: { sort: 'created_at', order: 'desc' },
  pending: { status: 'pending' },
  claimed: { status: 'claimed' },
  completed: { status: 'completed' },
  overdue: { status: 'pending,claimed', sla_breached: 'true' },
};

export default function WorkflowTasksPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState('all');
  const [page, setPage] = useState(1);
  const [delegateTask, setDelegateTask] = useState<HumanTask | null>(null);

  const { data: counts } = useQuery({
    queryKey: ['tasks', 'count'],
    queryFn: () => apiGet<TaskCounts>(API_ENDPOINTS.WORKFLOWS_TASKS_COUNT),
    refetchInterval: 30000,
  });

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['tasks', 'list', activeTab, page],
    queryFn: () =>
      apiGet<PaginatedResponse<HumanTask>>(API_ENDPOINTS.WORKFLOWS_TASKS, {
        ...TAB_PARAMS[activeTab],
        page,
        per_page: 25,
      }),
    refetchInterval: 30000,
  });

  const handleTabChange = useCallback((tab: string) => {
    setActiveTab(tab);
    setPage(1);
  }, []);

  const columns = getTaskColumns({
    onOpen: (task) => router.push(`/workflows/tasks/${task.id}`),
    onClaim: (task) => router.push(`/workflows/tasks/${task.id}`),
    onDelegate: (task) => setDelegateTask(task),
    currentUserId: user?.id,
  });

  if (isError) {
    return (
      <div className="space-y-6">
        <PageHeader title="My Tasks" description="Tasks assigned to you across all workflows." />
        <ErrorState message="Failed to load tasks" onRetry={() => refetch()} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="My Tasks"
        description="Tasks assigned to you across all workflows."
      />

      <TaskStatusTabs
        activeTab={activeTab}
        onTabChange={handleTabChange}
        counts={counts}
      />

      <DataTable
        columns={columns}
        data={data?.data ?? []}
        totalRows={data?.meta.total ?? 0}
        page={page}
        pageSize={25}
        onPageChange={setPage}
        onPageSizeChange={() => undefined}
        onSortChange={() => undefined}
        isLoading={isLoading}
        onRowClick={(row) => router.push(`/workflows/tasks/${row.id}`)}
      />

      {delegateTask && (
        <TaskDelegateDialog
          task={delegateTask}
          open={!!delegateTask}
          onOpenChange={(open) => {
            if (!open) setDelegateTask(null);
          }}
          onSuccess={() => {
            setDelegateTask(null);
            queryClient.invalidateQueries({ queryKey: ['tasks'] });
          }}
        />
      )}
    </div>
  );
}
