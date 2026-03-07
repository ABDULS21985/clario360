'use client';

import Link from 'next/link';
import { useRef, useState } from 'react';
import { Shield } from 'lucide-react';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo, cn } from '@/lib/utils';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import type { PaginatedResponse } from '@/types/api';
import type { Alert, Notification } from '@/types/models';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { Button } from '@/components/ui/button';
import { HighlightAnimation } from '@/components/realtime/highlight-animation';

function severityClass(severity: string): string {
  return cn(
    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
    severity === 'critical' && 'bg-red-100 text-red-800',
    severity === 'high' && 'bg-orange-100 text-orange-800',
    severity === 'medium' && 'bg-yellow-100 text-yellow-800',
    severity === 'low' && 'bg-blue-100 text-blue-800',
    severity === 'info' && 'bg-gray-100 text-gray-800',
  );
}

function statusClass(status: string): string {
  return cn(
    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
    status === 'new' && 'bg-destructive/10 text-destructive',
    status === 'acknowledged' && 'bg-amber-100 text-amber-800',
    status === 'investigating' && 'bg-blue-100 text-blue-800',
    status === 'resolved' && 'bg-green-100 text-green-800',
    status === 'false_positive' && 'bg-muted text-muted-foreground',
  );
}

export function RecentAlertsTable() {
  const [pendingAlert, setPendingAlert] = useState<Notification | null>(null);
  const [highlightedAlertId, setHighlightedAlertId] = useState<string | null>(null);
  const tableRef = useRef<HTMLDivElement>(null);
  const {
    data,
    isLoading,
    error,
    mutate,
  } = useRealtimeData<PaginatedResponse<Alert>>(API_ENDPOINTS.CYBER_ALERTS, {
    params: {
      sort: 'created_at',
      order: 'desc',
      per_page: 5,
    },
    wsTopics: ['alert.created'],
    onNewItem: (notification) => {
      setPendingAlert(notification);
    },
  });

  const handleShow = async () => {
    await mutate();
    const alertId = getAlertId(pendingAlert);
    if (alertId) {
      setHighlightedAlertId(alertId);
      window.setTimeout(() => setHighlightedAlertId(null), 3000);
    }
    setPendingAlert(null);
    tableRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <div ref={tableRef} className="rounded-lg border bg-card">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <h3 className="text-sm font-semibold">Recent Alerts</h3>
        <Link href="/cyber/alerts" className="text-xs text-primary hover:underline">
          View all →
        </Link>
      </div>

      {pendingAlert && (
        <div className="flex items-center justify-between gap-3 border-b bg-yellow-50 px-4 py-2 text-sm">
          <span>New alert detected.</span>
          <Button variant="ghost" size="sm" onClick={() => void handleShow()}>
            Show
          </Button>
        </div>
      )}

      {isLoading ? (
        <div className="p-4">
          <LoadingSkeleton variant="table-row" count={5} />
        </div>
      ) : error ? (
        <ErrorState message="Failed to load alerts" onRetry={() => void mutate()} />
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
                <th className="hidden px-4 py-2.5 text-left text-xs font-medium text-muted-foreground sm:table-cell">Status</th>
                <th className="hidden px-4 py-2.5 text-left text-xs font-medium text-muted-foreground md:table-cell">Time</th>
              </tr>
            </thead>
            <tbody>
              {data.data.map((alert) => (
                <tr
                  key={alert.id}
                  className="border-b last:border-0 hover:bg-muted/30"
                >
                  <td colSpan={4} className="p-0">
                    <HighlightAnimation
                      highlight={highlightedAlertId === alert.id}
                      highlightKey={highlightedAlertId === alert.id ? alert.id : null}
                    >
                      <div className="grid cursor-pointer grid-cols-[140px_1fr] sm:grid-cols-[140px_1fr_160px] md:grid-cols-[140px_1fr_160px_120px]">
                        <div className="px-4 py-2.5">
                          <span className={severityClass(alert.severity)}>{alert.severity}</span>
                        </div>
                        <div className="px-4 py-2.5">
                          <Link
                            href={`/cyber/alerts/${alert.id}`}
                            className="block max-w-[200px] truncate font-medium hover:underline"
                          >
                            {alert.title}
                          </Link>
                        </div>
                        <div className="hidden px-4 py-2.5 sm:block">
                          <span className={statusClass(alert.status)}>{alert.status.replace('_', ' ')}</span>
                        </div>
                        <div className="hidden px-4 py-2.5 text-muted-foreground md:block">
                          {timeAgo(alert.created_at)}
                        </div>
                      </div>
                    </HighlightAnimation>
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

function getAlertId(notification: Notification | null): string | null {
  if (!notification) {
    return null;
  }
  if (typeof notification.data?.id === 'string') {
    return notification.data.id;
  }
  if (typeof notification.action_url === 'string') {
    const parts = notification.action_url.split('/');
    return parts[parts.length - 1] ?? null;
  }
  return null;
}
