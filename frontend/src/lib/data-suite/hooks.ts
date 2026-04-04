'use client';

import { useEffect, useRef, useState } from 'react';

interface UsePollingOperationOptions<T> {
  enabled: boolean;
  intervalMs?: number;
  fetcher: () => Promise<T>;
  isDone: (value: T) => boolean;
  onData?: (value: T) => void;
  onError?: (error: Error) => void;
}

interface UsePollingOperationResult<T> {
  data: T | null;
  error: string | null;
  isPolling: boolean;
  start: () => void;
  stop: () => void;
}

export function usePollingOperation<T>({
  enabled,
  intervalMs = 3000,
  fetcher,
  isDone,
  onData,
  onError,
}: UsePollingOperationOptions<T>): UsePollingOperationResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isPolling, setIsPolling] = useState(enabled);
  const intervalRef = useRef<number | null>(null);
  const pendingRef = useRef(false);

  useEffect(() => {
    setIsPolling(enabled);
  }, [enabled]);

  useEffect(() => {
    if (!isPolling) {
      return;
    }

    const tick = async () => {
      if (pendingRef.current) {
        return;
      }
      pendingRef.current = true;
      try {
        const next = await fetcher();
        setData(next);
        setError(null);
        onData?.(next);
        if (isDone(next)) {
          setIsPolling(false);
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Operation failed';
        setError(message);
        onError?.(err instanceof Error ? err : new Error(message));
        setIsPolling(false);
      } finally {
        pendingRef.current = false;
      }
    };

    void tick();
    intervalRef.current = window.setInterval(() => {
      void tick();
    }, intervalMs);

    return () => {
      if (intervalRef.current !== null) {
        window.clearInterval(intervalRef.current);
      }
    };
  }, [fetcher, intervalMs, isDone, isPolling, onData, onError]);

  return {
    data,
    error,
    isPolling,
    start: () => {
      setError(null);
      setIsPolling(true);
    },
    stop: () => setIsPolling(false),
  };
}
