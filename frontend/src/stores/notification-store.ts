'use client';

import { create } from 'zustand';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { Notification, ConnectionStatus } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

interface NotificationState {
  unreadCount: number;
  recentNotifications: Notification[];
  connectionStatus: ConnectionStatus;
  isInitialized: boolean;

  setUnreadCount: (count: number) => void;
  incrementUnreadCount: () => void;
  decrementUnreadCount: () => void;
  addNotification: (notif: Notification) => void;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  setConnectionStatus: (status: ConnectionStatus) => void;
  fetchInitialData: () => Promise<void>;
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
  unreadCount: 0,
  recentNotifications: [],
  connectionStatus: 'disconnected',
  isInitialized: false,

  setUnreadCount: (count) => set({ unreadCount: Math.max(0, count) }),
  incrementUnreadCount: () => set((s) => ({ unreadCount: s.unreadCount + 1 })),
  decrementUnreadCount: () => set((s) => ({ unreadCount: Math.max(0, s.unreadCount - 1) })),

  addNotification: (notif) =>
    set((s) => ({
      recentNotifications: [notif, ...s.recentNotifications].slice(0, 10),
      unreadCount: s.unreadCount + 1,
    })),

  markAsRead: (id) =>
    set((s) => ({
      recentNotifications: s.recentNotifications.map((n) =>
        n.id === id ? { ...n, read: true } : n,
      ),
      unreadCount: Math.max(0, s.unreadCount - 1),
    })),

  markAllAsRead: () =>
    set((s) => ({
      recentNotifications: s.recentNotifications.map((n) => ({ ...n, read: true })),
      unreadCount: 0,
    })),

  setConnectionStatus: (status) => set({ connectionStatus: status }),

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
        set({
          recentNotifications: listResp.value.data,
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
