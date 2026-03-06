'use client';

import { useState } from 'react';
import { Bell, Shield, Database, Workflow, Settings } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo, cn } from '@/lib/utils';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Button } from '@/components/ui/button';
import type { PaginatedResponse } from '@/types/api';
import type { Notification, NotificationCategory } from '@/types/models';

function getCategoryIcon(category: NotificationCategory) {
  const map: Record<NotificationCategory, React.ElementType> = {
    security: Shield,
    data: Database,
    workflow: Workflow,
    system: Settings,
    governance: Bell,
    legal: Bell,
  };
  return map[category] ?? Bell;
}

export default function NotificationsPage() {
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['notifications', page],
    queryFn: () =>
      apiGet<PaginatedResponse<Notification>>(API_ENDPOINTS.NOTIFICATIONS, {
        page,
        per_page: 25,
        sort: 'created_at',
        order: 'desc',
      }),
  });

  const markAllMutation = useMutation({
    mutationFn: () => apiPut('/api/v1/notifications/read-all'),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  });

  return (
    <div className="space-y-6">
      <PageHeader
        title="Notifications"
        description="All your notifications"
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={() => markAllMutation.mutate()}
            disabled={markAllMutation.isPending}
          >
            Mark all read
          </Button>
        }
      />

      {isLoading ? (
        <LoadingSkeleton variant="list-item" count={10} />
      ) : isError ? (
        <ErrorState message="Failed to load notifications" onRetry={() => refetch()} />
      ) : !data || data.data.length === 0 ? (
        <EmptyState icon={Bell} title="No notifications" description="You're all caught up!" />
      ) : (
        <div className="rounded-lg border bg-card overflow-hidden divide-y">
          {data.data.map((notif) => {
            const Icon = getCategoryIcon(notif.category);
            return (
              <div
                key={notif.id}
                className={cn(
                  'flex items-start gap-3 px-4 py-3',
                  !notif.read && 'bg-primary/5',
                )}
              >
                <Icon className={cn(
                  'mt-0.5 h-4 w-4 shrink-0',
                  notif.priority === 'critical' && 'text-red-500',
                  notif.priority === 'high' && 'text-orange-500',
                  notif.priority === 'medium' && 'text-blue-500',
                  notif.priority === 'low' && 'text-muted-foreground',
                )} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    {!notif.read && <span className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" />}
                    <p className={cn('text-sm truncate', !notif.read && 'font-medium')}>{notif.title}</p>
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">{notif.body}</p>
                  <p className="text-xs text-muted-foreground mt-1">{timeAgo(notif.created_at)}</p>
                </div>
              </div>
            );
          })}
          {data.meta.total_pages > 1 && (
            <div className="flex items-center justify-between px-4 py-3">
              <p className="text-xs text-muted-foreground">Page {page} of {data.meta.total_pages}</p>
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
  );
}
