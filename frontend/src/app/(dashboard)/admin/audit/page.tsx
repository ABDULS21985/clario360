'use client';

import { useState } from 'react';
import { ClipboardList } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDateTime } from '@/lib/utils';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import type { PaginatedResponse } from '@/types/api';
import type { AuditLog } from '@/types/models';

export default function AdminAuditPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'audit', page],
    queryFn: () =>
      apiGet<PaginatedResponse<AuditLog>>(API_ENDPOINTS.AUDIT_LOGS, {
        page,
        per_page: 25,
        sort: 'created_at',
        order: 'desc',
      }),
  });

  return (
    <PermissionRedirect permission="*:read">
      <div className="space-y-6">
        <PageHeader title="Audit Logs" description="Tenant-wide audit trail" />

        {isLoading ? (
          <LoadingSkeleton variant="table-row" count={10} />
        ) : isError ? (
          <ErrorState message="Failed to load audit logs" onRetry={() => refetch()} />
        ) : !data || data.data.length === 0 ? (
          <EmptyState icon={ClipboardList} title="No audit logs" description="No audit events recorded." />
        ) : (
          <div className="rounded-lg border bg-card overflow-hidden">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/30">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">User</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Action</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Resource</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">IP</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Time</th>
                </tr>
              </thead>
              <tbody>
                {data.data.map((log) => (
                  <tr key={log.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3 text-xs text-muted-foreground">{log.user_email}</td>
                    <td className="px-4 py-3 font-mono text-xs">{log.action}</td>
                    <td className="px-4 py-3 text-xs text-muted-foreground hidden md:table-cell">
                      {log.resource_type}{log.resource_id ? ` · ${log.resource_id.slice(0, 8)}` : ''}
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground hidden lg:table-cell font-mono">
                      {log.ip_address}
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">{formatDateTime(log.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {data.meta.total_pages > 1 && (
              <div className="flex items-center justify-between border-t px-4 py-3">
                <p className="text-xs text-muted-foreground">
                  Page {data.meta.page} of {data.meta.total_pages}
                </p>
                <div className="flex gap-2">
                  <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}
                    className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent">Previous</button>
                  <button disabled={page >= data.meta.total_pages} onClick={() => setPage((p) => p + 1)}
                    className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent">Next</button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </PermissionRedirect>
  );
}
