'use client';

import { Clock, LogIn, Upload, AlertTriangle, CheckSquare, Settings, FileText } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo } from '@/lib/utils';
import { subDays, formatISO } from 'date-fns';
import { useAuth } from '@/hooks/use-auth';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import type { PaginatedResponse } from '@/types/api';
import type { AuditLog } from '@/types/models';

function getActionIcon(action: string) {
  if (action.includes('login')) return LogIn;
  if (action.includes('upload') || action.includes('file')) return Upload;
  if (action.includes('alert')) return AlertTriangle;
  if (action.includes('task') || action.includes('workflow')) return CheckSquare;
  if (action.includes('settings') || action.includes('update')) return Settings;
  if (action.includes('document') || action.includes('contract')) return FileText;
  return Clock;
}

function formatAction(log: AuditLog): string {
  const action = log.action.replace(/_/g, ' ').replace(/\./g, ' ');
  if (log.resource_id) {
    return `${action}: ${log.resource_type} ${log.resource_id.slice(0, 8)}`;
  }
  return action;
}

export function ActivityTimeline() {
  const { user } = useAuth();
  const sevenDaysAgo = formatISO(subDays(new Date(), 7));

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['dashboard', 'activity'],
    queryFn: () =>
      apiGet<PaginatedResponse<AuditLog>>(API_ENDPOINTS.AUDIT_LOGS, {
        user_id: user?.id,
        per_page: 20,
        date_from: sevenDaysAgo,
      }),
    enabled: !!user?.id,
    refetchInterval: 120000,
  });

  return (
    <div className="rounded-lg border bg-card">
      <div className="border-b px-4 py-3">
        <h3 className="text-sm font-semibold">Recent Activity</h3>
      </div>

      {isLoading ? (
        <div className="p-4">
          <LoadingSkeleton variant="list-item" count={5} />
        </div>
      ) : isError ? (
        <ErrorState message="Failed to load activity" onRetry={() => refetch()} />
      ) : !data || data.data.length === 0 ? (
        <EmptyState icon={Clock} title="No recent activity" description="No activity in the last 7 days." />
      ) : (
        <div className="relative px-4 py-3">
          <div className="space-y-3">
            {data.data.map((log, idx) => {
              const Icon = getActionIcon(log.action);
              return (
                <div key={log.id} className="flex items-start gap-3">
                  <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted">
                    <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm capitalize truncate">{formatAction(log)}</p>
                    <p className="text-xs text-muted-foreground">{timeAgo(log.created_at)}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
