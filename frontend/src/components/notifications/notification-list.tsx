'use client';

import { groupNotificationsByDate } from '@/lib/notification-utils';
import { NotificationCard } from './notification-card';
import { NotificationEmpty } from './notification-empty';
import { Skeleton } from '@/components/ui/skeleton';
import type { Notification } from '@/types/models';

interface NotificationListProps {
  notifications: Notification[];
  isLoading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  limitReached?: boolean;
  error?: string | null;
  onLoadMore: () => void;
  onMarkRead: (id: string) => void;
  onDelete: (id: string) => void;
  onNavigate?: (url: string) => void;
  sentinelRef: (el: HTMLDivElement | null) => void;
  newIds?: string[];
  category?: string;
  isSelecting?: boolean;
  selectedIds?: Set<string>;
  onSelect?: (id: string) => void;
}

function NotificationSkeleton() {
  return (
    <div className="flex items-start gap-3 px-4 py-3">
      <Skeleton className="mt-1.5 h-2 w-2 rounded-full" />
      <Skeleton className="mt-0.5 h-4 w-4 rounded" />
      <div className="flex-1 space-y-1.5">
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-16" />
      </div>
    </div>
  );
}

export function NotificationList({
  notifications,
  isLoading,
  isLoadingMore,
  hasMore,
  limitReached = false,
  error,
  onLoadMore,
  onMarkRead,
  onDelete,
  onNavigate,
  sentinelRef,
  newIds = [],
  category = 'all',
  isSelecting = false,
  selectedIds = new Set(),
  onSelect,
}: NotificationListProps) {
  if (isLoading) {
    return (
      <div className="divide-y">
        {Array.from({ length: 5 }).map((_, i) => (
          <NotificationSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="px-4 py-8 text-center text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (notifications.length === 0) {
    return <NotificationEmpty category={category} />;
  }

  const groups = groupNotificationsByDate(notifications);

  return (
    <div>
      {Array.from(groups.entries()).map(([groupName, groupNotifs]) => (
        <div key={groupName}>
          <div className="sticky top-0 z-10 border-b bg-background px-4 py-2">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {groupName}
            </p>
          </div>
          <div className="divide-y">
            {groupNotifs.map((notif) => (
              <NotificationCard
                key={notif.id}
                notification={notif}
                onMarkRead={onMarkRead}
                onDelete={onDelete}
                onNavigate={onNavigate}
                isNew={newIds.includes(notif.id)}
                isSelecting={isSelecting}
                isSelected={selectedIds.has(notif.id)}
                onSelect={onSelect}
              />
            ))}
          </div>
        </div>
      ))}

      {/* Infinite scroll sentinel */}
      {hasMore && (
        <>
          <div ref={sentinelRef} className="h-4" aria-hidden />
          {isLoadingMore && (
            <div className="divide-y">
              {Array.from({ length: 3 }).map((_, i) => (
                <NotificationSkeleton key={i} />
              ))}
            </div>
          )}
        </>
      )}

      {!hasMore && notifications.length > 0 && (
        <div className="px-4 py-6 text-center">
          <p className="text-xs text-muted-foreground">
            {limitReached
              ? `Showing most recent ${notifications.length} notifications.`
              : 'No more notifications.'}
          </p>
        </div>
      )}

      {/* Fallback load more button */}
      {hasMore && !isLoadingMore && (
        <div className="px-4 py-3 text-center">
          <button
            onClick={onLoadMore}
            className="text-xs text-primary underline hover:no-underline"
          >
            Load more notifications
          </button>
        </div>
      )}
    </div>
  );
}
