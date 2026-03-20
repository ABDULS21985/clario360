'use client';

import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { cn } from '@/lib/utils';
import type { AlertStats } from '@/types/cyber';

interface AlertStatBarProps {
  onFilterBySeverity?: (severity: string) => void;
}

const SEVERITY_CONFIG = [
  { key: 'critical', label: 'Critical', color: 'bg-red-500', textColor: 'text-red-600', hoverBg: 'hover:bg-red-50 dark:hover:bg-red-950/30' },
  { key: 'high', label: 'High', color: 'bg-orange-500', textColor: 'text-orange-600', hoverBg: 'hover:bg-orange-50 dark:hover:bg-orange-950/30' },
  { key: 'medium', label: 'Medium', color: 'bg-yellow-500', textColor: 'text-yellow-600', hoverBg: 'hover:bg-yellow-50 dark:hover:bg-yellow-950/30' },
  { key: 'low', label: 'Low', color: 'bg-blue-500', textColor: 'text-blue-600', hoverBg: 'hover:bg-blue-50 dark:hover:bg-blue-950/30' },
];

export function AlertStatBar({ onFilterBySeverity }: AlertStatBarProps) {
  const { data: envelope } = useRealtimeData<{ data: AlertStats }>(
    API_ENDPOINTS.CYBER_ALERTS_STATS,
    { pollInterval: 60000 },
  );

  const stats = envelope?.data;
  if (!stats) return null;

  const severityMap = Object.fromEntries(stats.by_severity.map((s) => [s.name, s.count]));

  return (
    <div className="flex flex-wrap gap-2 rounded-lg border bg-card p-3">
      {SEVERITY_CONFIG.map(({ key, label, color, textColor, hoverBg }) => {
        const count = severityMap[key] ?? 0;
        return (
          <button
            key={key}
            onClick={() => onFilterBySeverity?.(key)}
            className={cn(
              'flex cursor-pointer items-center gap-2 rounded-md px-3 py-1.5 transition-colors',
              hoverBg,
            )}
          >
            <div className={cn('h-2 w-2 rounded-full', color)} />
            <span className="text-xs text-muted-foreground">{label}</span>
            <span className={cn('text-sm font-bold tabular-nums', textColor)}>{count}</span>
          </button>
        );
      })}
      <div className="ml-auto flex items-center gap-4 pr-1">
        <div className="text-center">
          <p className="text-xs text-muted-foreground">Open</p>
          <p className="text-sm font-bold">{stats.open_count}</p>
        </div>
        <div className="text-center">
          <p className="text-xs text-muted-foreground">Resolved</p>
          <p className="text-sm font-bold text-green-600">{stats.resolved_count}</p>
        </div>
      </div>
    </div>
  );
}
