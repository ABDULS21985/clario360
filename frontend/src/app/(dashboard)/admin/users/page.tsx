'use client';

import { useState } from 'react';
import { Users } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { formatDate, timeAgo } from '@/lib/utils';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Badge } from '@/components/ui/badge';
import type { PaginatedResponse } from '@/types/api';
import type { User } from '@/types/models';

function statusVariant(status: string) {
  const map: Record<string, 'default' | 'destructive' | 'warning' | 'success'> = {
    active: 'success',
    suspended: 'warning',
    deactivated: 'destructive',
    pending_verification: 'default',
  };
  return map[status] ?? 'default';
}

export default function AdminUsersPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'users', page],
    queryFn: () =>
      apiGet<PaginatedResponse<User>>('/api/v1/users', { page, per_page: 20 }),
  });

  return (
    <PermissionRedirect permission="users:read">
      <div className="space-y-6">
        <PageHeader title="Users" description="Manage tenant users and access" />

        {isLoading ? (
          <LoadingSkeleton variant="table-row" count={10} />
        ) : isError ? (
          <ErrorState message="Failed to load users" onRetry={() => refetch()} />
        ) : !data || data.data.length === 0 ? (
          <EmptyState icon={Users} title="No users found" description="No users in this tenant." />
        ) : (
          <div className="rounded-lg border bg-card overflow-hidden">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/30">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">User</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Roles</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Last login</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">MFA</th>
                </tr>
              </thead>
              <tbody>
                {data.data.map((user) => (
                  <tr key={user.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3">
                      <p className="font-medium">{`${user.first_name} ${user.last_name}`.trim() || '—'}</p>
                      <p className="text-xs text-muted-foreground">{user.email}</p>
                    </td>
                    <td className="px-4 py-3 hidden md:table-cell">
                      <div className="flex flex-wrap gap-1">
                        {user.roles.map((r) => (
                          <Badge key={r.id} variant="outline" className="text-xs">{r.name}</Badge>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={statusVariant(user.status)}>{user.status.replace('_', ' ')}</Badge>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground hidden lg:table-cell">
                      {user.last_login_at ? timeAgo(user.last_login_at) : 'Never'}
                    </td>
                    <td className="px-4 py-3 hidden lg:table-cell">
                      <Badge variant={user.mfa_enabled ? 'success' : 'outline'}>
                        {user.mfa_enabled ? 'Enabled' : 'Disabled'}
                      </Badge>
                    </td>
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
