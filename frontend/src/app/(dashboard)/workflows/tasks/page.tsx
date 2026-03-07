'use client';

import { useMemo, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
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
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { SearchInput } from '@/components/shared/forms/search-input';
import { taskFilters, fetchRoleFilterOptions } from '@/components/workflows/task-filters';
import type { HumanTask, TaskCounts } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

const TAB_PARAMS: Record<string, Record<string, string>> = {
  all: { sort: 'created_at', order: 'desc' },
  pending: { status: 'pending' },
  claimed: { status: 'claimed' },
  completed: { status: 'completed' },
  overdue: { status: 'pending,claimed', sla_breached: 'true' },
};

export default function WorkflowTasksPage() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const activeTab = searchParams.get('tab') ?? 'all';
  const [delegateTask, setDelegateTask] = useState<HumanTask | null>(null);

  const {
    data: counts,
    error: countsError,
    mutate: refetchCounts,
  } = useRealtimeData<TaskCounts>(API_ENDPOINTS.WORKFLOWS_TASKS_COUNT, {
    wsTopics: ['task.assigned', 'task.escalated', 'task.overdue', 'workflow.task.created'],
    pollInterval: 30000,
  });

  const { data: roleOptions = [] } = useQuery({
    queryKey: ['task-filter-roles'],
    queryFn: fetchRoleFilterOptions,
    staleTime: 60_000,
  });

  const filters = useMemo(
    () =>
      taskFilters.map((filter) =>
        filter.key === 'assignee_role'
          ? { ...filter, options: roleOptions }
          : filter,
      ),
    [roleOptions],
  );

  const taskTable = useDataTable<HumanTask>({
    queryKey: 'tasks',
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['task.assigned', 'task.escalated', 'task.overdue', 'workflow.task.created'],
    fetchFn: async (params) => {
      const filtersMap = params.filters ?? {};
      const queryParams: Record<string, unknown> = {
        ...TAB_PARAMS[activeTab],
        page: params.page,
        per_page: params.per_page,
        sort: params.sort ?? 'created_at',
        order: params.order ?? 'desc',
      };

      if (params.search) {
        queryParams.search = params.search;
      }

      for (const [key, value] of Object.entries(filtersMap)) {
        queryParams[key] = Array.isArray(value) ? value.join(',') : value;
      }

      return apiGet<PaginatedResponse<HumanTask>>(API_ENDPOINTS.WORKFLOWS_TASKS, queryParams);
    },
  });

  const columns = getTaskColumns({
    onOpen: (task) => router.push(`/workflows/tasks/${task.id}`),
    onClaim: (task) => router.push(`/workflows/tasks/${task.id}`),
    onDelegate: (task) => setDelegateTask(task),
    onViewWorkflow: (task) => router.push(`/workflows/${task.instance_id}`),
    currentUser: user,
  });

  const handleTabChange = (tab: string) => {
    const nextParams = new URLSearchParams(searchParams.toString());
    if (tab === 'all') {
      nextParams.delete('tab');
    } else {
      nextParams.set('tab', tab);
    }
    nextParams.set('page', '1');
    router.push(`${pathname}?${nextParams.toString()}`);
  };

  if (taskTable.error || countsError) {
    return (
      <div className="space-y-6">
        <PageHeader title="My Tasks" description="Tasks assigned to you across all workflows." />
        <ErrorState
          message="Failed to load tasks"
          onRetry={() => {
            void taskTable.refetch();
            void refetchCounts();
          }}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="My Tasks"
        description="Tasks assigned to you across all workflows."
      />

      <TaskStatusTabs activeTab={activeTab} onTabChange={handleTabChange} counts={counts} />

      <DataTable
        columns={columns}
        filters={filters}
        searchSlot={
          <SearchInput
            value={taskTable.searchValue}
            onChange={taskTable.setSearch}
            placeholder="Search tasks..."
          />
        }
        {...taskTable.tableProps}
        onRowClick={(row) => router.push(`/workflows/tasks/${row.id}`)}
      />

      {delegateTask && (
        <TaskDelegateDialog
          task={delegateTask}
          open={Boolean(delegateTask)}
          onOpenChange={(open) => {
            if (!open) {
              setDelegateTask(null);
            }
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
