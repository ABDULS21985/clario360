'use client';

import Link from 'next/link';
import { Shield } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo, cn } from '@/lib/utils';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import type { PaginatedResponse } from '@/types/api';
import type { Alert } from '@/types/models';

function severityClass(s: string): string {
  return cn(
    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
    s === 'critical' && 'bg-red-100 text-red-800',
    s === 'high' && 'bg-orange-100 text-orange-800',
    s === 'medium' && 'bg-yellow-100 text-yellow-800',
    s === 'low' && 'bg-blue-100 text-blue-800',
    s === 'info' && 'bg-gray-100 text-gray-800',
  );
}

function statusClass(s: string): string {
  return cn(
    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
    s === 'new' && 'bg-destructive/10 text-destructive',
    s === 'acknowledged' && 'bg-amber-100 text-amber-800',
    s === 'investigating' && 'bg-blue-100 text-blue-800',
    s === 'resolved' && 'bg-green-100 text-green-800',
    s === 'false_positive' && 'bg-muted text-muted-foreground',
  );
}

export function RecentAlertsTable() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['dashboard', 'recent-alerts'],
    queryFn: () =>
      apiGet<PaginatedResponse<Alert>>(API_ENDPOINTS.CYBER_ALERTS, {
        sort: 'created_at',
        order: 'desc',
        per_page: 5,
      }),
    refetchInterval: 60000,
  });

  return (
    <div className="rounded-lg border bg-card">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <h3 className="text-sm font-semibold">Recent Alerts</h3>
        <Link href="/cyber/alerts" className="text-xs text-primary hover:underline">
          View all →
        </Link>
      </div>

      {isLoading ? (
        <div className="p-4">
          <LoadingSkeleton variant="table-row" count={5} />
        </div>
      ) : isError ? (
        <ErrorState
          message="Failed to load alerts"
          onRetry={() => refetch()}
        />
      ) : !data || data.data.length === 0 ? (
        <EmptyState
          icon={Shield}
          title="No alerts found"
          description="No recent alerts to display."
        />
      ) : (
        <div className="overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/30">
              <tr>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Severity</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Title</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden sm:table-cell">Status</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Time</th>
              </tr>
            </thead>
            <tbody>
              {data.data.map((alert) => (
                <tr
                  key={alert.id}
                  className="border-b last:border-0 hover:bg-muted/30 cursor-pointer"
                >
                  <td className="px-4 py-2.5">
                    <span className={severityClass(alert.severity)}>{alert.severity}</span>
                  </td>
                  <td className="px-4 py-2.5">
                    <Link
                      href={`/cyber/alerts/${alert.id}`}
                      className="hover:underline font-medium truncate block max-w-[200px]"
                    >
                      {alert.title}
                    </Link>
                  </td>
                  <td className="px-4 py-2.5 hidden sm:table-cell">
                    <span className={statusClass(alert.status)}>{alert.status.replace('_', ' ')}</span>
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground hidden md:table-cell">
                    {timeAgo(alert.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
