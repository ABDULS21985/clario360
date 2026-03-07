'use client';

import { useEffect, useRef, useCallback, useMemo, useState } from 'react';
import { useQuery, type UseQueryResult } from '@tanstack/react-query';
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

  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const queryKey = useMemo(() => (params ? [url, params] : [url]), [url, params]);
  const queryKeyString = JSON.stringify(queryKey);

  const query: UseQueryResult<T, ApiError> = useQuery<T, ApiError>({
    queryKey,
    queryFn: () => apiGet<T>(url, params),
    refetchInterval: pollInterval > 0 ? pollInterval : undefined,
    refetchOnWindowFocus: options.refreshOnFocus !== false,
    enabled,
  });

  const { register, unregister } = useRealtimeStore();
  const queryEvent = useRealtimeStore((state) => state.queryEvents[queryKeyString]);

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

  useEffect(() => {
    if (!queryEvent || !enabled) {
      return;
    }

    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    debounceTimerRef.current = setTimeout(() => {
      void query.refetch();
      setLastUpdate(new Date(queryEvent.timestamp));
      if (onNewItem && isNotificationPayload(queryEvent.payload)) {
        onNewItem(queryEvent.payload);
      }
    }, 500);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [enabled, onNewItem, query, queryEvent]);

  const mutate = useCallback(async () => {
    await query.refetch();
    setLastUpdate(new Date());
  }, [query]);

  return {
    data: query.data,
    error: query.error ?? undefined,
    isLoading: query.isLoading,
    isValidating: query.isFetching,
    mutate,
    lastUpdate,
  };
}

function isNotificationPayload(payload: unknown): payload is Notification {
  if (!payload || typeof payload !== 'object') {
    return false;
  }

  return 'id' in payload && 'title' in payload && 'body' in payload && 'category' in payload;
}
