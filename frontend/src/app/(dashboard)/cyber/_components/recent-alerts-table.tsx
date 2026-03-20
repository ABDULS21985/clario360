'use client';

import Link from 'next/link';
import { AlertTriangle, ShieldAlert, Minus, TrendingDown } from 'lucide-react';
import { cn, timeAgo } from '@/lib/utils';
import { EmptyState } from '@/components/common/empty-state';
import type { AlertSummary, CyberSeverity } from '@/types/cyber';

interface RecentAlertsTableProps {
  alerts: AlertSummary[];
}

const SEVERITY_CONFIG: Record<CyberSeverity, { icon: React.ElementType; color: string; bg: string }> = {
  critical: { icon: ShieldAlert, color: 'text-red-600', bg: 'bg-red-100' },
  high: { icon: AlertTriangle, color: 'text-orange-600', bg: 'bg-orange-100' },
  medium: { icon: AlertTriangle, color: 'text-yellow-600', bg: 'bg-yellow-100' },
  low: { icon: TrendingDown, color: 'text-blue-600', bg: 'bg-blue-100' },
  info: { icon: Minus, color: 'text-gray-500', bg: 'bg-gray-100' },
};

const STATUS_COLORS: Record<string, string> = {
  new: 'bg-red-100 text-red-800',
  acknowledged: 'bg-yellow-100 text-yellow-800',
  investigating: 'bg-blue-100 text-blue-800',
  in_progress: 'bg-purple-100 text-purple-800',
  resolved: 'bg-green-100 text-green-800',
  closed: 'bg-gray-100 text-gray-800',
  false_positive: 'bg-gray-100 text-gray-600',
  escalated: 'bg-red-200 text-red-900',
};

export function RecentAlertsTable({ alerts }: RecentAlertsTableProps) {
  if (alerts.length === 0) {
    return (
      <EmptyState
        icon={AlertTriangle}
        title="No recent alerts"
        description="No critical alerts in the last 24 hours."
      />
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <table className="w-full text-sm">
        <thead className="border-b bg-muted/30">
          <tr>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Sev</th>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Title</th>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Status</th>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Confidence</th>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Detected</th>
          </tr>
        </thead>
        <tbody>
          {alerts.map((alert) => {
            const sevConfig = SEVERITY_CONFIG[alert.severity] ?? SEVERITY_CONFIG.info;
            const SevIcon = sevConfig.icon;
            return (
              <tr key={alert.id} className="border-b last:border-0 hover:bg-muted/20">
                <td className="px-3 py-2">
                  <span className={cn('inline-flex items-center justify-center rounded-full p-1', sevConfig.bg)}>
                    <SevIcon className={cn('h-3 w-3', sevConfig.color)} />
                  </span>
                </td>
                <td className="px-3 py-2 max-w-[120px] sm:max-w-[200px]">
                  <Link
                    href={`/cyber/alerts/${alert.id}`}
                    className="font-medium hover:underline line-clamp-1"
                  >
                    {alert.title}
                  </Link>
                </td>
                <td className="px-3 py-2 hidden md:table-cell">
                  <span className={cn('inline-flex rounded-full px-2 py-0.5 text-xs font-medium', STATUS_COLORS[alert.status] ?? 'bg-gray-100 text-gray-700')}>
                    {alert.status.replace('_', ' ')}
                  </span>
                </td>
                <td className="px-3 py-2 hidden lg:table-cell">
                  {alert.confidence_score !== undefined ? (
                    <div className="flex items-center gap-1.5">
                      <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted">
                        <div
                          className={cn(
                            'h-full rounded-full',
                            alert.confidence_score > 0.8 ? 'bg-green-500' : alert.confidence_score > 0.5 ? 'bg-yellow-500' : 'bg-red-500',
                          )}
                          style={{ width: `${Math.round(alert.confidence_score * 100)}%` }}
                        />
                      </div>
                      <span className="text-xs text-muted-foreground">{Math.round(alert.confidence_score * 100)}%</span>
                    </div>
                  ) : (
                    <span className="text-xs text-muted-foreground">—</span>
                  )}
                </td>
                <td className="px-3 py-2 text-xs text-muted-foreground whitespace-nowrap">
                  {timeAgo(alert.created_at)}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
