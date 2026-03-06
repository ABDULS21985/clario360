'use client';

import { useState, useEffect, useRef, useCallback } from 'react';

interface UseCountdownReturn {
  seconds: number;
  isRunning: boolean;
  start: (durationSeconds: number) => void;
  reset: () => void;
}

export function useCountdown(): UseCountdownReturn {
  const [seconds, setSeconds] = useState(0);
  const [isRunning, setIsRunning] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const stop = useCallback(() => {
    if (intervalRef.current !== null) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    setIsRunning(false);
  }, []);

  const start = useCallback(
    (durationSeconds: number) => {
      stop();
      setSeconds(durationSeconds);
      setIsRunning(true);
      intervalRef.current = setInterval(() => {
        setSeconds((prev) => {
          if (prev <= 1) {
            stop();
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    },
    [stop],
  );

  const reset = useCallback(() => {
    stop();
    setSeconds(0);
  }, [stop]);

  useEffect(() => {
    return () => {
      if (intervalRef.current !== null) clearInterval(intervalRef.current);
    };
  }, []);

  return { seconds, isRunning, start, reset };
}
