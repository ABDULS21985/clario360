'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showSuccess, showApiError } from '@/lib/toast';
import type { PaginatedResponse } from '@/types/api';
import type {
  NotificationWebhook,
  CreateWebhookRequest,
  CreateWebhookResponse,
  WebhookDelivery,
} from '@/types/models';
import type { FetchParams } from '@/types/table';

const WEBHOOKS_KEY = 'notification-webhooks';

function buildParams(params?: FetchParams): Record<string, unknown> {
  if (!params) return {};
  const result: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
  };
  if (params.sort) result.sort = params.sort;
  if (params.order) result.order = params.order;
  if (params.search) result.search = params.search;
  if (params.filters) {
    for (const [key, value] of Object.entries(params.filters)) {
      result[key] = value;
    }
  }
  return result;
}

export function useNotificationWebhooks(params?: FetchParams) {
  return useQuery({
    queryKey: [WEBHOOKS_KEY, params],
    queryFn: () =>
      apiGet<PaginatedResponse<NotificationWebhook>>(
        API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS,
        buildParams(params),
      ),
    staleTime: 30_000,
  });
}

export function useNotificationWebhook(webhookId: string) {
  return useQuery({
    queryKey: [WEBHOOKS_KEY, webhookId],
    queryFn: () =>
      apiGet<NotificationWebhook>(
        `${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}`,
      ),
    enabled: Boolean(webhookId),
  });
}

export function useCreateWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateWebhookRequest) =>
      apiPost<CreateWebhookResponse>(API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [WEBHOOKS_KEY] });
      showSuccess('Webhook created successfully');
    },
    onError: (error) => showApiError(error),
  });
}

export function useUpdateWebhook(webhookId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<CreateWebhookRequest>) =>
      apiPut<NotificationWebhook>(
        `${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}`,
        data,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [WEBHOOKS_KEY] });
      showSuccess('Webhook updated successfully');
    },
    onError: (error) => showApiError(error),
  });
}

export function useDeleteWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (webhookId: string) =>
      apiDelete(`${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [WEBHOOKS_KEY] });
      showSuccess('Webhook deleted');
    },
    onError: (error) => showApiError(error),
  });
}

export function useTestWebhook() {
  return useMutation({
    mutationFn: (webhookId: string) =>
      apiPost<{ success: boolean; response_status: number; response_body: string }>(
        `${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}/test`,
      ),
    onError: (error) => showApiError(error),
  });
}

export function useRotateWebhookSecret() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (webhookId: string) =>
      apiPost<{ secret: string }>(
        `${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}/rotate`,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [WEBHOOKS_KEY] });
    },
    onError: (error) => showApiError(error),
  });
}

export function useWebhookDeliveries(
  webhookId: string,
  params?: FetchParams,
) {
  return useQuery({
    queryKey: [WEBHOOKS_KEY, webhookId, 'deliveries', params],
    queryFn: () =>
      apiGet<PaginatedResponse<WebhookDelivery>>(
        `${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}/deliveries`,
        buildParams(params),
      ),
    enabled: Boolean(webhookId),
    staleTime: 15_000,
  });
}
