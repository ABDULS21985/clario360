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
        data,
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
    }) =>
      apiPost<HumanTask>(
        `${API_ENDPOINTS.WORKFLOWS_TASKS}/${taskId}/complete`,
        data,
      ),
    onSuccess: (_data, variables) => {
      showSuccess('Task completed.');
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
      queryClient.invalidateQueries({
        queryKey: [TASKS_KEY, variables.taskId],
      });
    },
    onError: (error) => showApiError(error),
  });
}
