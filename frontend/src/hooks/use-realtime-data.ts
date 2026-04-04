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
    onNewItem,
    params,
    pollInterval = 0,
    enabled = true,
  } = options;

  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onNewItemRef = useRef(onNewItem);
  onNewItemRef.current = onNewItem;
  const topicSignature = wsTopics.join('|');
  const stableTopics = useMemo(
    () => (topicSignature ? topicSignature.split('|') : []),
    [topicSignature],
  );

  const queryKey = useMemo(() => (params ? [url, params] : [url]), [url, params]);
  const queryKeyString = JSON.stringify(queryKey);

  const query: UseQueryResult<T, ApiError> = useQuery<T, ApiError>({
    queryKey,
    queryFn: () => apiGet<T>(url, params),
    refetchInterval: pollInterval > 0 ? pollInterval : undefined,
    refetchOnWindowFocus: options.refreshOnFocus === true, // opt-in, not opt-out
    enabled,
  });

  const { register, unregister } = useRealtimeStore();
  const queryEvent = useRealtimeStore((state) => state.queryEvents[queryKeyString]);
  const { refetch } = query;

  useEffect(() => {
    if (!enabled || stableTopics.length === 0) return;

    for (const topic of stableTopics) {
      register(topic, queryKeyString);
    }

    return () => {
      for (const topic of stableTopics) {
        unregister(topic, queryKeyString);
      }
    };
  }, [stableTopics, queryKeyString, register, unregister, enabled]);

  useEffect(() => {
    if (!queryEvent || !enabled) {
      return;
    }

    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    debounceTimerRef.current = setTimeout(() => {
      void refetch();
      setLastUpdate(new Date(queryEvent.timestamp));
      if (onNewItemRef.current && isNotificationPayload(queryEvent.payload)) {
        onNewItemRef.current(queryEvent.payload);
      }
    }, 500);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [enabled, refetch, queryEvent]);

  const mutate = useCallback(async () => {
    await refetch();
    setLastUpdate(new Date());
  }, [refetch]);

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
