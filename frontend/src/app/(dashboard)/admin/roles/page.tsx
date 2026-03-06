'use client';

import { KeyRound } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Badge } from '@/components/ui/badge';
import type { PaginatedResponse } from '@/types/api';
import type { Role } from '@/types/models';

export default function AdminRolesPage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'roles'],
    queryFn: () => apiGet<PaginatedResponse<Role>>('/api/v1/roles'),
  });

  return (
    <PermissionRedirect permission="roles:read">
      <div className="space-y-6">
        <PageHeader title="Roles" description="Manage roles and permissions" />

        {isLoading ? (
          <LoadingSkeleton variant="table-row" count={8} />
        ) : isError ? (
          <ErrorState message="Failed to load roles" onRetry={() => refetch()} />
        ) : !data || data.data.length === 0 ? (
          <EmptyState icon={KeyRound} title="No roles found" description="No roles configured." />
        ) : (
          <div className="rounded-lg border bg-card overflow-hidden">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/30">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Role</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Permissions</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Type</th>
                </tr>
              </thead>
              <tbody>
                {data.data.map((role) => (
                  <tr key={role.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3">
                      <p className="font-medium">{role.name}</p>
                      <p className="text-xs text-muted-foreground">{role.description}</p>
                    </td>
                    <td className="px-4 py-3 hidden md:table-cell">
                      <div className="flex flex-wrap gap-1 max-w-md">
                        {role.permissions.slice(0, 5).map((p) => (
                          <Badge key={p} variant="outline" className="text-xs font-mono">{p}</Badge>
                        ))}
                        {role.permissions.length > 5 && (
                          <Badge variant="secondary">+{role.permissions.length - 5}</Badge>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={role.is_system ? 'default' : 'outline'}>
                        {role.is_system ? 'System' : 'Custom'}
                      </Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </PermissionRedirect>
  );
}
