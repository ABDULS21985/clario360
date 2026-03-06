'use client';

import { useState } from 'react';
import Link from 'next/link';
import { AlertTriangle } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo, cn } from '@/lib/utils';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import type { PaginatedResponse } from '@/types/api';
import type { Alert } from '@/types/models';

function SeverityBadge({ severity }: { severity: string }) {
  return (
    <span className={cn(
      'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
      severity === 'critical' && 'bg-red-100 text-red-800',
      severity === 'high' && 'bg-orange-100 text-orange-800',
      severity === 'medium' && 'bg-yellow-100 text-yellow-800',
      severity === 'low' && 'bg-blue-100 text-blue-800',
      severity === 'info' && 'bg-gray-100 text-gray-800',
    )}>
      {severity}
    </span>
  );
}

function StatusBadge({ status }: { status: string }) {
  return (
    <span className={cn(
      'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold',
      status === 'new' && 'bg-destructive/10 text-destructive',
      status === 'acknowledged' && 'bg-amber-100 text-amber-800',
      status === 'investigating' && 'bg-blue-100 text-blue-800',
      status === 'resolved' && 'bg-green-100 text-green-800',
      status === 'false_positive' && 'bg-muted text-muted-foreground',
    )}>
      {status.replace('_', ' ')}
    </span>
  );
}

export default function CyberAlertsPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['cyber', 'alerts', page],
    queryFn: () =>
      apiGet<PaginatedResponse<Alert>>(API_ENDPOINTS.CYBER_ALERTS, {
        sort: 'created_at',
        order: 'desc',
        per_page: 20,
        page,
      }),
    refetchInterval: 30000,
  });

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader title="Alerts" description="Monitor and manage security alerts" />

        {isLoading ? (
          <LoadingSkeleton variant="table-row" count={10} />
        ) : isError ? (
          <ErrorState message="Failed to load alerts" onRetry={() => refetch()} />
        ) : !data || data.data.length === 0 ? (
          <EmptyState
            icon={AlertTriangle}
            title="No alerts found"
            description="No security alerts match the current filter."
          />
        ) : (
          <div className="rounded-lg border bg-card overflow-hidden">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/30">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Severity</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Title</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Source</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Created</th>
                </tr>
              </thead>
              <tbody>
                {data.data.map((alert) => (
                  <tr key={alert.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3"><SeverityBadge severity={alert.severity} /></td>
                    <td className="px-4 py-3">
                      <Link href={`/cyber/alerts/${alert.id}`} className="font-medium hover:underline">
                        {alert.title}
                      </Link>
                      <p className="text-xs text-muted-foreground truncate max-w-xs">{alert.description}</p>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground hidden md:table-cell">{alert.source}</td>
                    <td className="px-4 py-3"><StatusBadge status={alert.status} /></td>
                    <td className="px-4 py-3 text-muted-foreground hidden lg:table-cell">{timeAgo(alert.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {data.meta.total_pages > 1 && (
              <div className="flex items-center justify-between border-t px-4 py-3">
                <p className="text-xs text-muted-foreground">
                  Page {data.meta.page} of {data.meta.total_pages} ({data.meta.total} total)
                </p>
                <div className="flex gap-2">
                  <button
                    disabled={page <= 1}
                    onClick={() => setPage((p) => p - 1)}
                    className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent"
                  >
                    Previous
                  </button>
                  <button
                    disabled={page >= data.meta.total_pages}
                    onClick={() => setPage((p) => p + 1)}
                    className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </PermissionRedirect>
  );
}
