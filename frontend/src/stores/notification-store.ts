'use client';

import { create } from 'zustand';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { Notification, ConnectionStatus } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';
import { isNotificationRead } from '@/lib/notification-utils';

interface NotificationState {
  unreadCount: number;
  recentNotifications: Notification[];
  connectionStatus: ConnectionStatus;
  reconnectAttempt: number;
  nextRetryAt: number | null;
  reconnectToken: number;
  isInitialized: boolean;

  setUnreadCount: (count: number) => void;
  incrementUnreadCount: () => void;
  decrementUnreadCount: () => void;
  addNotification: (notif: Notification) => void;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  deleteNotification: (id: string) => void;
  setConnectionStatus: (status: ConnectionStatus) => void;
  setReconnectState: (attempt: number, nextRetryAt: number | null) => void;
  requestReconnect: () => void;
  fetchInitialData: () => Promise<void>;
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
  unreadCount: 0,
  recentNotifications: [],
  connectionStatus: 'disconnected',
  reconnectAttempt: 0,
  nextRetryAt: null,
  reconnectToken: 0,
  isInitialized: false,

  setUnreadCount: (count) => set({ unreadCount: Math.max(0, count) }),
  incrementUnreadCount: () => set((s) => ({ unreadCount: s.unreadCount + 1 })),
  decrementUnreadCount: () => set((s) => ({ unreadCount: Math.max(0, s.unreadCount - 1) })),

  addNotification: (notif) =>
    set((s) => {
      const existing = s.recentNotifications.filter((item) => item.id !== notif.id);
      const isUnread = !isNotificationRead(notif);
      return {
        recentNotifications: [notif, ...existing].slice(0, 10),
        unreadCount: isUnread ? s.unreadCount + 1 : s.unreadCount,
      };
    }),

  markAsRead: (id) =>
    set((s) => ({
      recentNotifications: s.recentNotifications.map((n) =>
        n.id === id ? { ...n, read: true, read_at: n.read_at ?? new Date().toISOString() } : n,
      ),
      unreadCount: Math.max(
        0,
        s.unreadCount -
          (s.recentNotifications.some((n) => n.id === id && !isNotificationRead(n)) ? 1 : 0),
      ),
    })),

  markAllAsRead: () =>
    set((s) => ({
      recentNotifications: s.recentNotifications.map((n) => ({
        ...n,
        read: true,
        read_at: n.read_at ?? new Date().toISOString(),
      })),
      unreadCount: 0,
    })),

  deleteNotification: (id) =>
    set((s) => {
      const notification = s.recentNotifications.find((item) => item.id === id);
      return {
        recentNotifications: s.recentNotifications.filter((item) => item.id !== id),
        unreadCount:
          notification && !isNotificationRead(notification)
            ? Math.max(0, s.unreadCount - 1)
            : s.unreadCount,
      };
    }),

  setConnectionStatus: (status) => set({ connectionStatus: status }),

  setReconnectState: (attempt, nextRetryAt) => set({ reconnectAttempt: attempt, nextRetryAt }),

  requestReconnect: () =>
    set((state) => ({
      reconnectToken: state.reconnectToken + 1,
      nextRetryAt: null,
    })),

  fetchInitialData: async () => {
    if (get().isInitialized) return;
    try {
      const [countResp, listResp] = await Promise.allSettled([
        apiGet<{ count: number }>(API_ENDPOINTS.NOTIFICATIONS_UNREAD_COUNT),
        apiGet<PaginatedResponse<Notification>>(API_ENDPOINTS.NOTIFICATIONS, {
          per_page: 10,
          sort: 'created_at',
          order: 'desc',
        }),
      ]);

      if (countResp.status === 'fulfilled') {
        set({ unreadCount: countResp.value.count });
      }
      if (listResp.status === 'fulfilled') {
        const notifications = listResp.value.data.map((notification) => ({
          ...notification,
          read:
            notification.read ??
            Boolean(notification.read_at),
        }));
        set({
          recentNotifications: notifications,
          isInitialized: true,
        });
      } else {
        set({ isInitialized: true });
      }
    } catch {
      set({ isInitialized: true });
    }
  },
}));
