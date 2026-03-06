'use client';

import { useEffect, useRef, useCallback, useState } from 'react';
import { useQuery, useQueryClient, type UseQueryResult } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { useRealtimeStore } from '@/stores/realtime-store';
import type { ApiError } from '@/types/api';
import type { Notification } from '@/types/models';

interface UseRealtimeDataOptions {
  wsTopics?: string[];
  onNewItem?: (notification: Notification) => void;
  params?: Record<string, unknown>;
  refreshOnFocus?: boolean;
  pollInterval?: number;
  enabled?: boolean;
}

interface UseRealtimeDataResult<T> {
  data: T | undefined;
  error: ApiError | undefined;
  isLoading: boolean;
  isValidating: boolean;
  mutate: () => Promise<void>;
  lastUpdate: Date | null;
}

export function useRealtimeData<T>(
  url: string,
  options: UseRealtimeDataOptions = {},
): UseRealtimeDataResult<T> {
  const {
    wsTopics = [],
    params,
    pollInterval = 0,
    enabled = true,
  } = options;

  const queryClient = useQueryClient();
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Build stable query key
  const queryKey = params ? [url, params] : [url];
  const queryKeyString = JSON.stringify(queryKey);

  const query: UseQueryResult<T, ApiError> = useQuery<T, ApiError>({
    queryKey,
    queryFn: () => apiGet<T>(url, params),
    refetchInterval: pollInterval > 0 ? pollInterval : undefined,
    refetchOnWindowFocus: options.refreshOnFocus !== false,
    enabled,
  });

  // Debounced revalidation to batch rapid WS messages
  const scheduleRevalidation = useCallback(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      queryClient.invalidateQueries({ queryKey });
      setLastUpdate(new Date());
    }, 500);
  }, [queryClient, queryKey]);

  // Register this hook's query key with the realtime store for each topic
  const { register, unregister } = useRealtimeStore();

  useEffect(() => {
    if (!enabled || wsTopics.length === 0) return;

    for (const topic of wsTopics) {
      register(topic, queryKeyString);
    }

    return () => {
      for (const topic of wsTopics) {
        unregister(topic, queryKeyString);
      }
    };
  }, [wsTopics, queryKeyString, register, unregister, enabled]);

  // Listen for query invalidations triggered by the WS handler
  // (The WS handler calls queryClient.invalidateQueries directly via the realtime store)
  // We track last update time from our scheduled revalidations
  useEffect(() => {
    void scheduleRevalidation; // reference to prevent unused warning
  }, [scheduleRevalidation]);

  const mutate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey });
    setLastUpdate(new Date());
  }, [queryClient, queryKey]);

  return {
    data: query.data,
    error: query.error ?? undefined,
    isLoading: query.isLoading,
    isValidating: query.isFetching,
    mutate,
    lastUpdate,
  };
}
