'use client';

import { useState, useCallback, useEffect } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useQueries } from '@tanstack/react-query';
import { Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { NotificationList } from '@/components/notifications/notification-list';
import { NotificationCategoryTabs } from '@/components/notifications/notification-category-tabs';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useNotificationActions } from '@/hooks/use-notification-actions';
import { useNotificationStore } from '@/stores/notification-store';
import { useRealtimeStore } from '@/stores/realtime-store';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { Notification } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { showNewDataToast } from '@/components/realtime/new-data-toast';
import { isNotificationRead } from '@/lib/notification-utils';

const MAX_PAGES = 5;
const PAGE_SIZE = 20;
const MAX_ITEMS = MAX_PAGES * PAGE_SIZE;

export default function NotificationsPage() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const activeTab = searchParams.get('tab') ?? 'all';
  const unreadCount = useNotificationStore((state) => state.unreadCount);
  const { markAsRead, markAllAsRead, deleteNotification } = useNotificationActions();
  const [markAllLoading, setMarkAllLoading] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [newIds, setNewIds] = useState<string[]>([]);
  const notificationEvent = useRealtimeStore((state) => state.topicEvents['notification.new']);
  const notificationReadEvent = useRealtimeStore((state) => state.topicEvents['notification.read']);

  const fetchFn = useCallback(
    (page: number) => {
      const params: Record<string, unknown> = {
        page,
        per_page: PAGE_SIZE,
        sort: 'created_at',
        order: 'desc',
      };

      if (activeTab === 'unread') {
        params.read = 'false';
      } else if (activeTab !== 'all') {
        params.category = activeTab;
      }

      return apiGet<PaginatedResponse<Notification>>(API_ENDPOINTS.NOTIFICATIONS, params);
    },
    [activeTab],
  );

  const {
    items: notifications,
    isLoading,
    isLoadingMore,
    hasMore,
    limitReached,
    error,
    loadMore,
    mutate,
    sentinelRef,
    updateItems,
  } = useInfiniteScroll<Notification>(fetchFn, { maxPages: MAX_PAGES });

  const countQueries = useQueries({
    queries: [
      { queryKey: ['notification-count', 'all'], queryFn: () => fetchNotificationCount() },
      { queryKey: ['notification-count', 'security'], queryFn: () => fetchNotificationCount('security') },
      { queryKey: ['notification-count', 'workflow'], queryFn: () => fetchNotificationCount('workflow') },
      { queryKey: ['notification-count', 'data'], queryFn: () => fetchNotificationCount('data') },
      { queryKey: ['notification-count', 'system'], queryFn: () => fetchNotificationCount('system') },
    ],
  });

  const counts = {
    all: countQueries[0]?.data ?? 0,
    unread: unreadCount,
    security: countQueries[1]?.data ?? 0,
    workflow: countQueries[2]?.data ?? 0,
    data: countQueries[3]?.data ?? 0,
    system: countQueries[4]?.data ?? 0,
  };

  const handleTabChange = (tab: string) => {
    const nextParams = new URLSearchParams(searchParams.toString());
    if (tab === 'all') {
      nextParams.delete('tab');
    } else {
      nextParams.set('tab', tab);
    }
    router.push(nextParams.toString() ? `${pathname}?${nextParams.toString()}` : pathname);
  };

  const handleMarkAllRead = async () => {
    setMarkAllLoading(true);
    try {
      await markAllAsRead();
      updateItems((items) =>
        items.map((notification) => ({
          ...notification,
          read: true,
          read_at: notification.read_at ?? new Date().toISOString(),
        })),
      );
    } finally {
      setMarkAllLoading(false);
      setConfirmOpen(false);
    }
  };

  const handleMarkRead = async (id: string) => {
    await markAsRead(id);
    updateItems((items) =>
      items.map((notification) =>
        notification.id === id
          ? {
              ...notification,
              read: true,
              read_at: notification.read_at ?? new Date().toISOString(),
            }
          : notification,
      ),
    );
  };

  const handleDelete = async (id: string) => {
    await deleteNotification(id);
    updateItems((items) => items.filter((notification) => notification.id !== id));
  };

  useEffect(() => {
    if (!notificationEvent?.count) {
      return;
    }

    const notification = normalizeNotification(notificationEvent.payload);
    if (!matchesTab(notification, activeTab)) {
      return;
    }

    updateItems((items) => [notification, ...items.filter((item) => item.id !== notification.id)].slice(0, MAX_ITEMS));
    setNewIds((ids) => [notification.id, ...ids.filter((id) => id !== notification.id)]);
    showNewDataToast({
      title: notification.title,
      description: notification.body,
    });

    const timeout = window.setTimeout(() => {
      setNewIds((ids) => ids.filter((id) => id !== notification.id));
    }, 3000);

    return () => window.clearTimeout(timeout);
  }, [activeTab, notificationEvent, updateItems]);

  useEffect(() => {
    if (!notificationReadEvent?.count) {
      return;
    }

    const payload = notificationReadEvent.payload as { id?: string };
    if (!payload.id) {
      return;
    }

    updateItems((items) =>
      items.map((notification) =>
        notification.id === payload.id
          ? {
              ...notification,
              read: true,
              read_at: notification.read_at ?? new Date().toISOString(),
            }
          : notification,
      ),
    );
  }, [notificationReadEvent, updateItems]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Notifications"
        description="Stay up to date with activity across the platform."
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setConfirmOpen(true)}
              disabled={markAllLoading || unreadCount === 0}
            >
              {markAllLoading ? 'Marking...' : 'Mark All Read'}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => router.push('/settings/notifications')}
              aria-label="Notification settings"
            >
              <Settings className="h-4 w-4" />
            </Button>
          </div>
        }
      />

      <NotificationCategoryTabs
        activeTab={activeTab}
        onTabChange={handleTabChange}
        counts={counts}
      />

      <div className="rounded-lg border bg-card">
        <NotificationList
          notifications={notifications}
          isLoading={isLoading}
          isLoadingMore={isLoadingMore}
          hasMore={hasMore}
          limitReached={limitReached}
          error={error?.message ?? null}
          onLoadMore={loadMore}
          onMarkRead={handleMarkRead}
          onDelete={handleDelete}
          onNavigate={(url) => router.push(url)}
          sentinelRef={sentinelRef}
          newIds={newIds}
          category={activeTab}
        />
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Mark All Read"
        description={`Mark all ${unreadCount} unread notifications as read?`}
        confirmLabel="Mark all read"
        onConfirm={handleMarkAllRead}
        loading={markAllLoading}
      />
    </div>
  );
}

async function fetchNotificationCount(category?: string): Promise<number> {
  const response = await apiGet<PaginatedResponse<Notification>>(API_ENDPOINTS.NOTIFICATIONS, {
    per_page: 1,
    page: 1,
    sort: 'created_at',
    order: 'desc',
    ...(category ? { category } : {}),
  });

  return response.meta.total;
}

function normalizeNotification(payload: unknown): Notification {
  if (!payload || typeof payload !== 'object') {
    return {
      id: crypto.randomUUID(),
      title: 'New notification',
      body: '',
      category: 'system',
      priority: 'low',
      read: false,
      created_at: new Date().toISOString(),
    };
  }

  const notification = payload as Partial<Notification>;
  return {
    id: notification.id ?? crypto.randomUUID(),
    type: notification.type,
    title: notification.title ?? 'New notification',
    body: notification.body ?? '',
    category: notification.category ?? 'system',
    priority: notification.priority ?? 'low',
    data: notification.data ?? null,
    action_url: notification.action_url ?? null,
    read: notification.read ?? Boolean(notification.read_at),
    read_at: notification.read_at ?? null,
    created_at: notification.created_at ?? new Date().toISOString(),
  };
}

function matchesTab(notification: Notification, tab: string): boolean {
  if (tab === 'all') {
    return true;
  }
  if (tab === 'unread') {
    return !isNotificationRead(notification);
  }
  return notification.category === tab;
}
