'use client';

import Link from 'next/link';
import { useRef, useState } from 'react';
import { Shield, ArrowRight } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo, cn } from '@/lib/utils';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import type { PaginatedResponse } from '@/types/api';
import type { Alert, Notification } from '@/types/models';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { Button } from '@/components/ui/button';
import { HighlightAnimation } from '@/components/realtime/highlight-animation';

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#EF4444',
  high: '#F97316',
  medium: '#F59E0B',
  low: '#3B82F6',
  info: '#94A3B8',
};

function severityClass(severity: string): string {
  return cn(
    'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold',
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
    <motion.div
      ref={tableRef}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.3 }}
      className="flex flex-col rounded-2xl border border-border/60"
      style={{
        background: 'rgba(255, 255, 255, 0.6)',
        backdropFilter: 'blur(24px)',
        WebkitBackdropFilter: 'blur(24px)',
      }}
    >
      <div className="flex items-center justify-between border-b border-border/60 px-5 py-4">
        <div className="flex items-center gap-2.5">
          <Shield className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold">Recent Alerts</h3>
        </div>
        <Link
          href="/cyber/alerts"
          className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
        >
          View all
          <ArrowRight className="h-3 w-3" />
        </Link>
      </div>

      <AnimatePresence>
        {pendingAlert && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="overflow-hidden"
          >
            <div className="flex items-center justify-between gap-3 border-b border-border/60 bg-amber-50/60 px-5 py-2 text-sm">
              <span className="font-medium text-amber-800">New alert detected</span>
              <Button variant="ghost" size="sm" onClick={() => void handleShow()}>
                Show
              </Button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {isLoading ? (
        <div className="p-4">
          <LoadingSkeleton variant="table-row" count={5} />
        </div>
      ) : error ? (
        <ErrorState message="Failed to load alerts" onRetry={() => void mutate()} />
      ) : !data || data.data.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 px-4 py-10">
          <div
            className="flex h-12 w-12 items-center justify-center rounded-full"
            style={{ backgroundColor: 'rgba(34, 197, 94, 0.08)' }}
          >
            <Shield className="h-[22px] w-[22px] text-green-500" />
          </div>
          <div className="text-center">
            <p className="text-sm font-medium text-muted-foreground">No alerts found</p>
            <p className="mt-0.5 text-xs text-muted-foreground/70">No recent alerts to display.</p>
          </div>
        </div>
      ) : (
        <div className="overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b border-border/40 bg-muted/20">
              <tr>
                <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">Severity</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Title</th>
                <th className="hidden px-4 py-2.5 text-left text-xs font-medium text-muted-foreground sm:table-cell">Status</th>
                <th className="hidden px-4 py-2.5 text-left text-xs font-medium text-muted-foreground md:table-cell">Time</th>
              </tr>
            </thead>
            <tbody>
              {data.data.map((alert) => (
                <tr
                  key={alert.id}
                  className="border-b border-border/30 last:border-0 transition-colors hover:bg-muted/20"
                >
                  <td colSpan={4} className="p-0">
                    <HighlightAnimation
                      highlight={highlightedAlertId === alert.id}
                      highlightKey={highlightedAlertId === alert.id ? alert.id : null}
                    >
                      <div className="grid cursor-pointer grid-cols-[140px_1fr] sm:grid-cols-[140px_1fr_160px] md:grid-cols-[140px_1fr_160px_120px]">
                        <div className="flex items-center gap-2 px-5 py-2.5">
                          {/* Severity dot */}
                          <div
                            className="h-2.5 w-2.5 flex-shrink-0 rounded-full"
                            style={{
                              backgroundColor: SEVERITY_COLORS[alert.severity] ?? '#94A3B8',
                              boxShadow: alert.severity === 'critical'
                                ? `0 0 8px ${SEVERITY_COLORS.critical}80`
                                : 'none',
                            }}
                          />
                          <span className={severityClass(alert.severity)}>{alert.severity}</span>
                        </div>
                        <div className="px-4 py-2.5">
                          <Link
                            href={`/cyber/alerts/${alert.id}`}
                            className="block max-w-[240px] truncate font-medium hover:underline"
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
    </motion.div>
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
