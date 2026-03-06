'use client';

import { useQueryClient } from '@tanstack/react-query';
import { apiPut, apiDelete } from '@/lib/api';
import { useNotificationStore } from '@/stores/notification-store';
import { showError } from '@/lib/toast';
import type { Notification } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

const NOTIFICATIONS_KEY = '/api/v1/notifications';

export function useNotificationActions() {
  const queryClient = useQueryClient();
  const { markAsRead: storeMarkAsRead, markAllAsRead: storeMarkAllAsRead, decrementUnreadCount } =
    useNotificationStore();

  const markAsRead = async (id: string): Promise<void> => {
    // Optimistic update: update cache immediately
    queryClient.setQueriesData<PaginatedResponse<Notification>>(
      { queryKey: [NOTIFICATIONS_KEY], exact: false },
      (old) => {
        if (!old) return old;
        return {
          ...old,
          data: old.data.map((n) => (n.id === id ? { ...n, read: true } : n)),
        };
      },
    );
    storeMarkAsRead(id);
    decrementUnreadCount();

    try {
      await apiPut(`/api/v1/notifications/${id}/read`);
    } catch {
      // Revert on error
      queryClient.invalidateQueries({ queryKey: [NOTIFICATIONS_KEY] });
      showError('Failed to mark notification as read');
    }
  };

  const markAllAsRead = async (): Promise<void> => {
    try {
      await apiPut('/api/v1/notifications/read-all');
      // Update cache
      queryClient.setQueriesData<PaginatedResponse<Notification>>(
        { queryKey: [NOTIFICATIONS_KEY], exact: false },
        (old) => {
          if (!old) return old;
          return { ...old, data: old.data.map((n) => ({ ...n, read: true })) };
        },
      );
      storeMarkAllAsRead();
    } catch {
      showError('Failed to mark all notifications as read');
      queryClient.invalidateQueries({ queryKey: [NOTIFICATIONS_KEY] });
    }
  };

  const deleteNotification = async (id: string): Promise<void> => {
    // Optimistic remove
    queryClient.setQueriesData<PaginatedResponse<Notification>>(
      { queryKey: [NOTIFICATIONS_KEY], exact: false },
      (old) => {
        if (!old) return old;
        return { ...old, data: old.data.filter((n) => n.id !== id) };
      },
    );

    try {
      await apiDelete(`/api/v1/notifications/${id}`);
    } catch {
      // Revert on error
      queryClient.invalidateQueries({ queryKey: [NOTIFICATIONS_KEY] });
      showError('Failed to delete notification');
    }
  };

  return { markAsRead, markAllAsRead, deleteNotification };
}
