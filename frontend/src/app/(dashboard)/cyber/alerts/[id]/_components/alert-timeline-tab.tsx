'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Clock } from 'lucide-react';
import { timeAgo, cn } from '@/lib/utils';
import type { AlertTimelineEntry } from '@/types/cyber';

const ACTION_COLORS: Record<string, string> = {
  created: 'bg-blue-500',
  assigned: 'bg-purple-500',
  status_changed: 'bg-amber-500',
  escalated: 'bg-orange-500',
  resolved: 'bg-green-500',
  commented: 'bg-slate-400',
  false_positive: 'bg-gray-500',
};

interface AlertTimelineTabProps {
  alertId: string;
}

export function AlertTimelineTab({ alertId }: AlertTimelineTabProps) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['alert-timeline', alertId],
    queryFn: () =>
      apiGet<{ data: AlertTimelineEntry[] }>(
        `${API_ENDPOINTS.CYBER_ALERTS}/${alertId}/timeline`,
      ),
  });

  if (isLoading) return <LoadingSkeleton variant="list-item" count={6} />;
  if (error) return <ErrorState message="Failed to load timeline" onRetry={() => refetch()} />;

  const entries = data?.data ?? [];

  if (entries.length === 0) {
    return (
      <EmptyState icon={Clock} title="No timeline events" description="No activity has been recorded for this alert." />
    );
  }

  return (
    <div className="relative pl-6">
      {/* Vertical line */}
      <div className="absolute left-2.5 top-0 bottom-0 w-px bg-border" />

      <div className="space-y-4">
        {entries.map((entry, i) => {
          const dotColor = ACTION_COLORS[entry.action] ?? 'bg-slate-400';
          return (
            <div key={entry.id} className="relative flex items-start gap-4">
              {/* Dot */}
              <div className={cn('absolute -left-6 mt-1 h-3 w-3 rounded-full border-2 border-background', dotColor)} />

              <div className="flex-1 rounded-xl border bg-card p-3">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-xs font-semibold capitalize">{entry.action.replace(/_/g, ' ')}</p>
                  <span className="text-xs text-muted-foreground">{timeAgo(entry.created_at)}</span>
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{entry.description}</p>
                {entry.actor_name && (
                  <p className="mt-1 text-xs text-muted-foreground">by {entry.actor_name}</p>
                )}
                {(entry.old_value || entry.new_value) && (
                  <div className="mt-2 flex items-center gap-2 text-xs">
                    {entry.old_value && (
                      <span className="rounded bg-red-100 px-1.5 py-0.5 font-mono text-red-700 line-through dark:bg-red-950/30 dark:text-red-400">
                        {entry.old_value}
                      </span>
                    )}
                    {entry.new_value && (
                      <span className="rounded bg-green-100 px-1.5 py-0.5 font-mono text-green-700 dark:bg-green-950/30 dark:text-green-400">
                        {entry.new_value}
                      </span>
                    )}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
