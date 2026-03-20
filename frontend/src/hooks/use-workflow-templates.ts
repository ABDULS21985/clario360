'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showSuccess, showApiError } from '@/lib/toast';
import type {
  WorkflowTemplate,
  WorkflowDefinition,
  CreateFromTemplateRequest,
} from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

const TEMPLATES_KEY = 'workflow-templates';
const DEFINITIONS_KEY = 'workflow-definitions';

export function useWorkflowTemplates(params?: Record<string, unknown>) {
  return useQuery({
    queryKey: [TEMPLATES_KEY, params],
    queryFn: async () => {
      const resp = await apiGet<{ templates: WorkflowTemplate[]; total: number }>(
        API_ENDPOINTS.WORKFLOWS_TEMPLATES,
        params,
      );
      // Normalize to PaginatedResponse shape
      return {
        data: resp.templates ?? [],
        meta: { page: 1, per_page: resp.total || 0, total: resp.total || 0, total_pages: 1 },
      } as PaginatedResponse<WorkflowTemplate>;
    },
  });
}

export function useWorkflowTemplate(templateId: string) {
  return useQuery({
    queryKey: [TEMPLATES_KEY, templateId],
    queryFn: () =>
      apiGet<WorkflowTemplate>(
        `${API_ENDPOINTS.WORKFLOWS_TEMPLATES}/${templateId}`,
      ),
    enabled: !!templateId,
  });
}

export function useCreateWorkflowTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      definition_id: string;
      description: string;
      tags: string[];
    }) => apiPost<WorkflowTemplate>(API_ENDPOINTS.WORKFLOWS_TEMPLATES, data),
    onSuccess: () => {
      showSuccess('Template created.');
      queryClient.invalidateQueries({ queryKey: [TEMPLATES_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useUpdateWorkflowTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      templateId,
      data,
    }: {
      templateId: string;
      data: Partial<WorkflowTemplate>;
    }) =>
      apiPut<WorkflowTemplate>(
        `${API_ENDPOINTS.WORKFLOWS_TEMPLATES}/${templateId}`,
        data,
      ),
    onSuccess: (_data, variables) => {
      showSuccess('Template updated.');
      queryClient.invalidateQueries({ queryKey: [TEMPLATES_KEY] });
      queryClient.invalidateQueries({
        queryKey: [TEMPLATES_KEY, variables.templateId],
      });
    },
    onError: (error) => showApiError(error),
  });
}

export function useDeleteWorkflowTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (templateId: string) =>
      apiDelete(`${API_ENDPOINTS.WORKFLOWS_TEMPLATES}/${templateId}`),
    onSuccess: () => {
      showSuccess('Template deleted.');
      queryClient.invalidateQueries({ queryKey: [TEMPLATES_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useCreateDefinitionFromTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateFromTemplateRequest) =>
      apiPost<WorkflowDefinition>(
        `${API_ENDPOINTS.WORKFLOWS_TEMPLATES}/${data.template_id}/instantiate`,
        { name: data.name, description: data.description },
      ),
    onSuccess: () => {
      showSuccess('Workflow created from template.');
      queryClient.invalidateQueries({ queryKey: [DEFINITIONS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}
