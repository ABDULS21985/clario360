'use client';

import { useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { NotificationList } from '@/components/notifications/notification-list';
import { NotificationCategoryTabs } from '@/components/notifications/notification-category-tabs';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useNotificationActions } from '@/hooks/use-notification-actions';
import { useNotificationStore } from '@/stores/notification-store';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { Notification } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

const MAX_PAGES = 5;

export default function NotificationsPage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('all');
  const [tabKey, setTabKey] = useState(0); // force reset on tab change
  const unreadCount = useNotificationStore((s) => s.unreadCount);
  const { markAsRead, markAllAsRead, deleteNotification } = useNotificationActions();
  const [markAllLoading, setMarkAllLoading] = useState(false);

  const fetchFn = useCallback(
    (page: number) => {
      const params: Record<string, unknown> = {
        page,
        per_page: 20,
        sort: 'created_at',
        order: 'desc',
      };
      if (activeTab === 'unread') params.read = 'false';
      else if (activeTab !== 'all') params.category = activeTab;

      return apiGet<PaginatedResponse<Notification>>(API_ENDPOINTS.NOTIFICATIONS, params);
    },
    [activeTab],
  );

  const {
    items: notifications,
    isLoading,
    isLoadingMore,
    hasMore,
    onLoadMore,
    sentinelRef,
    reset,
  } = useInfiniteScroll<Notification>(fetchFn, { maxPages: MAX_PAGES });

  const handleTabChange = (tab: string) => {
    setActiveTab(tab);
    setTabKey((k) => k + 1);
    reset();
  };

  const handleMarkAllRead = async () => {
    setMarkAllLoading(true);
    try {
      await markAllAsRead();
    } finally {
      setMarkAllLoading(false);
    }
  };

  void tabKey;

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
              onClick={handleMarkAllRead}
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
        unreadCount={unreadCount}
      />

      <div className="rounded-lg border bg-card">
        <NotificationList
          notifications={notifications}
          isLoading={isLoading}
          isLoadingMore={isLoadingMore}
          hasMore={hasMore}
          onLoadMore={onLoadMore}
          onMarkRead={(id) => markAsRead(id)}
          onDelete={(id) => deleteNotification(id)}
          sentinelRef={sentinelRef}
          category={activeTab}
        />
      </div>
    </div>
  );
}
