'use client';

import { useQuery } from '@tanstack/react-query';
import {
  ArrowUpCircle,
  CheckCircle2,
  Clock3,
  GitMerge,
  MessageSquareText,
  Search,
  ShieldAlert,
  UserCheck,
} from 'lucide-react';
import { Timeline } from '@/components/shared/timeline';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDateTime } from '@/lib/utils';
import type { AlertTimelineEntry } from '@/types/cyber';

interface AlertTimelineProps {
  alertId: string;
}

export function AlertTimeline({ alertId }: AlertTimelineProps) {
  const timelineQuery = useQuery({
    queryKey: ['alert-timeline', alertId],
    queryFn: () => apiGet<{ data: AlertTimelineEntry[] }>(API_ENDPOINTS.CYBER_ALERT_TIMELINE(alertId)),
  });

  const entries = timelineQuery.data?.data ?? [];

  if (timelineQuery.isLoading) {
    return <LoadingSkeleton variant="list-item" count={5} />;
  }

  if (timelineQuery.error) {
    return <ErrorState message="Failed to load alert timeline" onRetry={() => void timelineQuery.refetch()} />;
  }

  if (entries.length === 0) {
    return (
      <div className="rounded-[26px] border border-dashed bg-card p-8 text-center text-muted-foreground">
        No activity has been recorded for this alert yet.
      </div>
    );
  }

  return (
    <div className="rounded-[26px] border bg-card p-5 shadow-sm">
      <Timeline
        items={entries.map((entry) => ({
          id: entry.id,
          icon: timelineConfig(entry.action).icon,
          title: entry.description,
          description: timelineDescription(entry),
          timestamp: formatDateTime(entry.created_at),
          variant: timelineConfig(entry.action).variant,
        }))}
      />
    </div>
  );
}

function timelineDescription(entry: AlertTimelineEntry): string {
  const parts = [];

  if (entry.actor_name) {
    parts.push(`Actor: ${entry.actor_name}`);
  }
  if (entry.old_value || entry.new_value) {
    parts.push(`Change: ${entry.old_value ?? '∅'} -> ${entry.new_value ?? '∅'}`);
  }

  return parts.join(' • ');
}

function timelineConfig(action: string): {
  icon: typeof Clock3;
  variant: 'default' | 'success' | 'warning' | 'error';
} {
  switch (action) {
    case 'assigned':
      return { icon: UserCheck, variant: 'default' };
    case 'escalated':
      return { icon: ArrowUpCircle, variant: 'warning' };
    case 'commented':
      return { icon: MessageSquareText, variant: 'default' };
    case 'false_positive':
      return { icon: ShieldAlert, variant: 'warning' };
    case 'merged':
      return { icon: GitMerge, variant: 'default' };
    case 'status_changed':
      return { icon: Search, variant: 'warning' };
    case 'resolved':
      return { icon: CheckCircle2, variant: 'success' };
    default:
      return { icon: Clock3, variant: 'default' };
  }
}
