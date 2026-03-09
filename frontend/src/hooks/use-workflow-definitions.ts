'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showSuccess, showApiError } from '@/lib/toast';
import type {
  WorkflowDefinition,
  WorkflowDefinitionVersion,
} from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

const DEFINITIONS_KEY = 'workflow-definitions';

export function useWorkflowDefinitions(params?: Record<string, unknown>) {
  return useQuery({
    queryKey: [DEFINITIONS_KEY, params],
    queryFn: () =>
      apiGet<PaginatedResponse<WorkflowDefinition>>(
        API_ENDPOINTS.WORKFLOWS_DEFINITIONS,
        params,
      ),
  });
}

export function useWorkflowDefinition(defId: string) {
  return useQuery({
    queryKey: [DEFINITIONS_KEY, defId],
    queryFn: () =>
      apiGet<WorkflowDefinition>(
        `${API_ENDPOINTS.WORKFLOWS_DEFINITIONS}/${defId}`,
      ),
    enabled: !!defId,
  });
}

export function useWorkflowDefinitionVersions(defId: string) {
  return useQuery({
    queryKey: [DEFINITIONS_KEY, defId, 'versions'],
    queryFn: () =>
      apiGet<{ versions: WorkflowDefinitionVersion[] }>(
        `${API_ENDPOINTS.WORKFLOWS_DEFINITIONS}/${defId}/versions`,
      ),
    enabled: !!defId,
  });
}

export function useCreateWorkflowDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<WorkflowDefinition>) =>
      apiPost<WorkflowDefinition>(API_ENDPOINTS.WORKFLOWS_DEFINITIONS, data),
    onSuccess: () => {
      showSuccess('Workflow definition created.');
      queryClient.invalidateQueries({ queryKey: [DEFINITIONS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useUpdateWorkflowDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      defId,
      data,
    }: {
      defId: string;
      data: Partial<WorkflowDefinition>;
    }) =>
      apiPut<WorkflowDefinition>(
        `${API_ENDPOINTS.WORKFLOWS_DEFINITIONS}/${defId}`,
        data,
      ),
    onSuccess: (_data, variables) => {
      showSuccess('Workflow definition updated.');
      queryClient.invalidateQueries({ queryKey: [DEFINITIONS_KEY] });
      queryClient.invalidateQueries({
        queryKey: [DEFINITIONS_KEY, variables.defId],
      });
    },
    onError: (error) => showApiError(error),
  });
}

export function useDeleteWorkflowDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (defId: string) =>
      apiDelete(`${API_ENDPOINTS.WORKFLOWS_DEFINITIONS}/${defId}`),
    onSuccess: () => {
      showSuccess('Workflow definition deleted.');
      queryClient.invalidateQueries({ queryKey: [DEFINITIONS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function usePublishWorkflowDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (defId: string) =>
      apiPost<WorkflowDefinition>(
        `${API_ENDPOINTS.WORKFLOWS_DEFINITIONS}/${defId}/publish`,
      ),
    onSuccess: (_data, defId) => {
      showSuccess('Workflow definition published.');
      queryClient.invalidateQueries({ queryKey: [DEFINITIONS_KEY] });
      queryClient.invalidateQueries({
        queryKey: [DEFINITIONS_KEY, defId],
      });
    },
    onError: (error) => showApiError(error),
  });
}

export function useArchiveWorkflowDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (defId: string) =>
      apiPost<WorkflowDefinition>(
        `${API_ENDPOINTS.WORKFLOWS_DEFINITIONS}/${defId}/archive`,
      ),
    onSuccess: (_data, defId) => {
      showSuccess('Workflow definition archived.');
      queryClient.invalidateQueries({ queryKey: [DEFINITIONS_KEY] });
      queryClient.invalidateQueries({
        queryKey: [DEFINITIONS_KEY, defId],
      });
    },
    onError: (error) => showApiError(error),
  });
}

export function useCloneWorkflowDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (defId: string) =>
      apiPost<WorkflowDefinition>(
        `${API_ENDPOINTS.WORKFLOWS_DEFINITIONS}/${defId}/clone`,
      ),
    onSuccess: () => {
      showSuccess('Workflow definition cloned.');
      queryClient.invalidateQueries({ queryKey: [DEFINITIONS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}
