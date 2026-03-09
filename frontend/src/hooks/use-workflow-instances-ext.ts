'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showSuccess, showApiError } from '@/lib/toast';
import type {
  WorkflowInstance,
  CreateInstanceRequest,
  StepExecution,
} from '@/types/models';

const INSTANCES_KEY = 'workflow-instances';

export function useWorkflowInstance(instanceId: string) {
  return useQuery({
    queryKey: [INSTANCES_KEY, instanceId],
    queryFn: () =>
      apiGet<WorkflowInstance>(
        `${API_ENDPOINTS.WORKFLOWS_INSTANCES}/${instanceId}`,
      ),
    enabled: !!instanceId,
  });
}

export function useWorkflowInstanceHistory(instanceId: string) {
  return useQuery({
    queryKey: [INSTANCES_KEY, instanceId, 'history'],
    queryFn: () =>
      apiGet<{ steps: StepExecution[] }>(
        `${API_ENDPOINTS.WORKFLOWS_INSTANCES}/${instanceId}/history`,
      ),
    enabled: !!instanceId,
  });
}

export function useCreateWorkflowInstance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateInstanceRequest) =>
      apiPost<WorkflowInstance>(API_ENDPOINTS.WORKFLOWS_INSTANCES, data),
    onSuccess: () => {
      showSuccess('Workflow started.');
      queryClient.invalidateQueries({ queryKey: [INSTANCES_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useUpdateWorkflowInstance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      instanceId,
      data,
    }: {
      instanceId: string;
      data: { variables?: Record<string, unknown>; context?: Record<string, unknown> };
    }) =>
      apiPut<WorkflowInstance>(
        `${API_ENDPOINTS.WORKFLOWS_INSTANCES}/${instanceId}`,
        data,
      ),
    onSuccess: (_data, variables) => {
      showSuccess('Instance updated.');
      queryClient.invalidateQueries({
        queryKey: [INSTANCES_KEY, variables.instanceId],
      });
    },
    onError: (error) => showApiError(error),
  });
}

export function useDeleteWorkflowInstance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (instanceId: string) =>
      apiDelete(`${API_ENDPOINTS.WORKFLOWS_INSTANCES}/${instanceId}`),
    onSuccess: () => {
      showSuccess('Instance deleted.');
      queryClient.invalidateQueries({ queryKey: [INSTANCES_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function usePauseWorkflowInstance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (instanceId: string) =>
      apiPost(`${API_ENDPOINTS.WORKFLOWS_INSTANCES}/${instanceId}/pause`),
    onSuccess: (_data, instanceId) => {
      showSuccess('Workflow paused.');
      queryClient.invalidateQueries({
        queryKey: [INSTANCES_KEY, instanceId],
      });
      queryClient.invalidateQueries({ queryKey: [INSTANCES_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useResumeWorkflowInstance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (instanceId: string) =>
      apiPost(`${API_ENDPOINTS.WORKFLOWS_INSTANCES}/${instanceId}/resume`),
    onSuccess: (_data, instanceId) => {
      showSuccess('Workflow resumed.');
      queryClient.invalidateQueries({
        queryKey: [INSTANCES_KEY, instanceId],
      });
      queryClient.invalidateQueries({ queryKey: [INSTANCES_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useRetryWorkflowInstance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (instanceId: string) =>
      apiPost(`${API_ENDPOINTS.WORKFLOWS_INSTANCES}/${instanceId}/retry`),
    onSuccess: (_data, instanceId) => {
      showSuccess('Workflow retry initiated.');
      queryClient.invalidateQueries({
        queryKey: [INSTANCES_KEY, instanceId],
      });
      queryClient.invalidateQueries({ queryKey: [INSTANCES_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}
