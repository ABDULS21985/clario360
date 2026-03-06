'use client';

import { useEffect, useRef, useState } from 'react';
import { apiGet } from '@/lib/api';
import type { BadgeConfig } from '@/config/navigation';
import { useSidebarStore } from '@/stores/sidebar-store';

type BadgeMap = Map<string, number | undefined>;

export function useBadgeCounts(configs: BadgeConfig[]): BadgeMap {
  const [counts, setCounts] = useState<BadgeMap>(new Map());
  const collapsed = useSidebarStore((s) => s.collapsed);
  const intervalsRef = useRef<Map<string, ReturnType<typeof setInterval>>>(new Map());

  useEffect(() => {
    const uniqueConfigs = new Map<string, BadgeConfig>();
    for (const cfg of configs) {
      if (!uniqueConfigs.has(cfg.endpoint)) {
        uniqueConfigs.set(cfg.endpoint, cfg);
      }
    }

    // Fetch once immediately
    for (const [endpoint, cfg] of uniqueConfigs) {
      const fetch = async () => {
        try {
          const resp = await apiGet<Record<string, number>>(endpoint);
          const val = resp[cfg.key];
          if (typeof val === 'number') {
            setCounts((prev) => {
              const next = new Map(prev);
              next.set(endpoint, val);
              return next;
            });
          }
        } catch {
          // Error tolerance: hide badge on failure
        }
      };
      fetch();

      const interval = collapsed ? cfg.pollIntervalMs * 2 : cfg.pollIntervalMs;
      const id = setInterval(fetch, interval);
      intervalsRef.current.set(endpoint, id);
    }

    return () => {
      for (const id of intervalsRef.current.values()) clearInterval(id);
      intervalsRef.current.clear();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [configs.map((c) => c.endpoint).join(','), collapsed]);

  return counts;
}
