'use client';

import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { getAccessToken } from '@/lib/auth';
import { useNotificationStore } from '@/stores/notification-store';
import { useRealtimeStore } from '@/stores/realtime-store';
import type { ConnectionStatus, Notification } from '@/types/models';

const MAX_RECONNECT_ATTEMPTS = 10;
const BACKOFF_DELAYS = [1000, 2000, 4000, 8000, 16000, 30000];

// Toast-worthy event types (only show toast for critical events)
const TOAST_TYPES = new Set([
  'alert.created',
  'alert.escalated',
  'task.assigned',
  'task.escalated',
  'remediation.approval_required',
  'security.incident',
  'pipeline.failed',
]);

interface WSMessage {
  type: string;
  data: unknown;
  timestamp: string;
}

function getBackoffDelay(attempt: number): number {
  const idx = Math.min(attempt, BACKOFF_DELAYS.length - 1);
  return BACKOFF_DELAYS[idx];
}

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);
  const intentionalCloseRef = useRef(false);
  const queryClient = useQueryClient();

  const { setConnectionStatus, addNotification, markAsRead, setUnreadCount } =
    useNotificationStore.getState();

  const invalidateForTopic = useCallback(
    (topic: string) => {
      const keys = useRealtimeStore.getState().getKeysForTopic(topic);
      for (const key of keys) {
        try {
          const parsed = JSON.parse(key) as unknown[];
          queryClient.invalidateQueries({ queryKey: parsed });
        } catch {
          // ignore parse errors
        }
      }
    },
    [queryClient],
  );

  const connect = useCallback(() => {
    const token = getAccessToken();
    if (!token) return;

    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
    const wsProtocol = apiUrl.startsWith('https') ? 'wss' : 'ws';
    const wsBase = apiUrl.replace(/^https?/, wsProtocol);
    const wsUrl = `${wsBase}/ws/v1/notifications?token=${token}`;

    setConnectionStatus('connecting');

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnectionStatus('connected');
      attemptRef.current = 0;
    };

    ws.onmessage = (event: MessageEvent) => {
      try {
        const msg: WSMessage = JSON.parse(event.data as string);

        // 1. Dispatch to notification store
        switch (msg.type) {
          case 'notification.new':
            addNotification(msg.data as Notification);
            // Show toast for critical events
            if (TOAST_TYPES.has((msg.data as Notification)?.category ?? '')) {
              // Lazy import to avoid circular deps
              import('@/lib/toast').then(({ showNotificationToast }) => {
                showNotificationToast(msg.data as Notification);
              }).catch(() => undefined);
            }
            break;
          case 'notification.read':
            markAsRead((msg.data as { id: string }).id);
            break;
          case 'unread.count':
            setUnreadCount((msg.data as { count: number }).count);
            break;
          case 'connection.ack':
            break;
        }

        // 2. Dispatch to realtime store → invalidate react-query caches
        invalidateForTopic(msg.type);
      } catch {
        // Ignore malformed messages
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (intentionalCloseRef.current) return;

      if (attemptRef.current < MAX_RECONNECT_ATTEMPTS) {
        setConnectionStatus('reconnecting');
        const delay = getBackoffDelay(attemptRef.current);
        attemptRef.current += 1;
        reconnectTimerRef.current = setTimeout(() => connect(), delay);
      } else {
        setConnectionStatus('failed');
      }
    };

    ws.onerror = () => {
      setConnectionStatus('error' as ConnectionStatus);
    };
  }, [setConnectionStatus, addNotification, markAsRead, setUnreadCount, invalidateForTopic]);

  useEffect(() => {
    connect();

    return () => {
      intentionalCloseRef.current = true;
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmount');
        wsRef.current = null;
      }
    };
  }, [connect]);
}
