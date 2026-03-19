'use client';

import { useQuery, useMutation } from '@tanstack/react-query';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showApiError } from '@/lib/toast';
import type { DeliveryStats, TestNotificationRequest, RetryFailedRequest } from '@/types/models';

interface DeliveryStatsParams {
  period?: '7d' | '30d' | '90d';
  channel?: 'email' | 'in_app' | 'websocket' | 'webhook';
}

export function useDeliveryStats(params?: DeliveryStatsParams) {
  return useQuery({
    queryKey: ['delivery-stats', params],
    queryFn: () =>
      apiGet<DeliveryStats>(API_ENDPOINTS.NOTIFICATIONS_DELIVERY_STATS, params as Record<string, unknown>),
    staleTime: 60_000,
  });
}

export function useSendTestNotification() {
  return useMutation({
    mutationFn: (data: TestNotificationRequest) =>
      apiPost<{ success: boolean; message: string }>(API_ENDPOINTS.NOTIFICATIONS_TEST, data),
    onError: (error) => showApiError(error),
  });
}

export function useRetryFailedDeliveries() {
  return useMutation({
    mutationFn: (data: RetryFailedRequest) =>
      apiPost<{ retried: number; message: string }>(API_ENDPOINTS.NOTIFICATIONS_RETRY_FAILED, data),
    onError: (error) => showApiError(error),
  });
}
