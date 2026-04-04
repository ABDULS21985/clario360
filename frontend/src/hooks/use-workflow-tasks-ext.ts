'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showSuccess, showApiError } from '@/lib/toast';
import type {
  HumanTask,
  CompleteTaskRequest,
  AssignTaskRequest,
  TaskComment,
} from '@/types/models';

const TASKS_KEY = 'workflow-tasks';

export function useWorkflowTask(taskId: string) {
  return useQuery({
    queryKey: [TASKS_KEY, taskId],
    queryFn: () =>
      apiGet<HumanTask>(`${API_ENDPOINTS.WORKFLOWS_TASKS}/${taskId}`),
    enabled: !!taskId,
  });
}

export function useAssignTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      taskId,
      data,
    }: {
      taskId: string;
      data: AssignTaskRequest;
    }) =>
      apiPost<HumanTask>(
        `${API_ENDPOINTS.WORKFLOWS_TASKS}/${taskId}/assign`,
        { user_id: data.user_id },
      ),
    onSuccess: (_data, variables) => {
      showSuccess('Task assigned.');
      queryClient.invalidateQueries({
        queryKey: [TASKS_KEY, variables.taskId],
      });
      queryClient.invalidateQueries({ queryKey: [TASKS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useCompleteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      taskId,
      data,
    }: {
      taskId: string;
      data: CompleteTaskRequest;
    }) => {
      // Route reject actions to the dedicated /reject endpoint.
      if (data.action === 'reject') {
        return apiPost<HumanTask>(
          `${API_ENDPOINTS.WORKFLOWS_TASKS}/${taskId}/reject`,
          { reason: data.comment || 'Rejected' },
        );
      }
      // All other actions (approve, complete, escalate) go to /complete.
      return apiPost<HumanTask>(
        `${API_ENDPOINTS.WORKFLOWS_TASKS}/${taskId}/complete`,
        { form_data: data.form_data ?? {} },
      );
    },
    onSuccess: (_data, variables) => {
      const msg = variables.data.action === 'reject' ? 'Task rejected.' : 'Task completed.';
      showSuccess(msg);
      queryClient.invalidateQueries({
        queryKey: [TASKS_KEY, variables.taskId],
      });
      queryClient.invalidateQueries({ queryKey: [TASKS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useRejectTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      taskId,
      reason,
    }: {
      taskId: string;
      reason: string;
    }) =>
      apiPost<HumanTask>(
        `${API_ENDPOINTS.WORKFLOWS_TASKS}/${taskId}/reject`,
        { reason },
      ),
    onSuccess: (_data, variables) => {
      showSuccess('Task rejected.');
      queryClient.invalidateQueries({
        queryKey: [TASKS_KEY, variables.taskId],
      });
      queryClient.invalidateQueries({ queryKey: [TASKS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useAddTaskComment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      taskId,
      content,
    }: {
      taskId: string;
      content: string;
    }) =>
      apiPost<TaskComment>(
        `${API_ENDPOINTS.WORKFLOWS_TASKS}/${taskId}/comment`,
        { content },
      ),
    onSuccess: (_data, variables) => {
      showSuccess('Comment added.');
      // Invalidate the task detail query key used by TaskDetailPageClient.
      queryClient.invalidateQueries({ queryKey: ['task', variables.taskId] });
      // Also refresh the task list cache.
      queryClient.invalidateQueries({ queryKey: [TASKS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}
