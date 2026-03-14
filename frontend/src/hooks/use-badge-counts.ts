'use client';

import { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import { apiGet } from '@/lib/api';
import type { BadgeConfig } from '@/config/navigation';
import { useSidebarStore } from '@/stores/sidebar-store';

type BadgeMap = Map<string, number | undefined>;

const BASE_POLL_MS = 120_000; // 2 min base (was per-badge 30-60s intervals)

export function useBadgeCounts(configs: BadgeConfig[]): BadgeMap {
  const [counts, setCounts] = useState<BadgeMap>(new Map());
  const collapsed = useSidebarStore((s) => s.collapsed);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const isMountedRef = useRef(true);

  // Stable endpoint signature to avoid re-running effect on every render
  const endpointSignature = useMemo(
    () => configs.map((c) => c.endpoint).join(','),
    [configs],
  );

  // Deduplicate configs by endpoint
  const uniqueConfigs = useMemo(() => {
    const map = new Map<string, BadgeConfig>();
    for (const cfg of configs) {
      if (!map.has(cfg.endpoint)) {
        map.set(cfg.endpoint, cfg);
      }
    }
    return map;
  }, [endpointSignature]); // eslint-disable-line react-hooks/exhaustive-deps

  // Single batched fetch for all badge endpoints
  const fetchAll = useCallback(async () => {
    if (!isMountedRef.current) return;

    const entries = Array.from(uniqueConfigs.entries());
    if (entries.length === 0) return;

    const results = await Promise.allSettled(
      entries.map(async ([endpoint, cfg]) => {
        const resp = await apiGet<Record<string, number>>(endpoint);
        return { endpoint, value: resp[cfg.key] } as const;
      }),
    );

    if (!isMountedRef.current) return;

    setCounts((prev) => {
      const next = new Map(prev);
      let changed = false;
      for (const result of results) {
        if (result.status === 'fulfilled' && typeof result.value.value === 'number') {
          if (prev.get(result.value.endpoint) !== result.value.value) {
            next.set(result.value.endpoint, result.value.value);
            changed = true;
          }
        }
      }
      return changed ? next : prev;
    });
  }, [uniqueConfigs]);

  useEffect(() => {
    isMountedRef.current = true;

    // Fetch once immediately
    void fetchAll();

    // Single interval for all badges; double interval when sidebar collapsed
    const interval = collapsed ? BASE_POLL_MS * 2 : BASE_POLL_MS;
    intervalRef.current = setInterval(() => void fetchAll(), interval);

    return () => {
      isMountedRef.current = false;
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [fetchAll, collapsed]);

  return counts;
}
