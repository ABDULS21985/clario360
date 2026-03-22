'use client';

import { useEffect, useRef, useCallback } from 'react';
import { getAccessToken } from '@/lib/auth';
import { useNotificationStore } from '@/stores/notification-store';
import { useRealtimeStore } from '@/stores/realtime-store';
import type { Notification } from '@/types/models';
import { isNotificationRead } from '@/lib/notification-utils';

const MAX_RECONNECT_ATTEMPTS = 10;
const BACKOFF_DELAYS = [1000, 2000, 4000, 8000, 16000, 30000];

const TOAST_TYPES = new Set([
  'alert.created',
  'alert.escalated',
  'task.assigned',
  'task.escalated',
  'workflow.task.created',
  'workflow.task.escalated',
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
  const index = Math.min(attempt, BACKOFF_DELAYS.length - 1);
  return BACKOFF_DELAYS[index];
}

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);
  const intentionalCloseRef = useRef(false);
  const reconnectToken = useNotificationStore((state) => state.reconnectToken);

  const connect = useCallback(() => {
    const token = getAccessToken();
    if (!token) {
      return;
    }

    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
    const wsProtocol = apiUrl.startsWith('https') ? 'wss' : 'ws';
    const wsBase = apiUrl.replace(/^https?/, wsProtocol);
    const wsUrl = `${wsBase}/ws/v1/notifications?token=${token}`;
    const notificationStore = useNotificationStore.getState();

    notificationStore.setConnectionStatus('connecting');
    notificationStore.setReconnectState(0, null);

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      const store = useNotificationStore.getState();
      store.setConnectionStatus('connected');
      store.setReconnectState(0, null);
      attemptRef.current = 0;
    };

    ws.onmessage = (event: MessageEvent) => {
      try {
        const message: WSMessage = JSON.parse(event.data as string);
        const realtimeStore = useRealtimeStore.getState();
        realtimeStore.publish(message.type, message.data, message.timestamp);

        switch (message.type) {
          case 'notification.new': {
            const notification = normalizeNotification(message.data);
            useNotificationStore.getState().addNotification(notification);
            for (const topic of normalizeTopics(notification.type)) {
              realtimeStore.publish(topic, notification, message.timestamp);
            }

            if (notification.type && TOAST_TYPES.has(notification.type)) {
              import('@/lib/toast')
                .then(({ showNotificationToast }) => {
                  showNotificationToast(notification);
                })
                .catch(() => undefined);
            }
            break;
          }
          case 'notification.read':
            useNotificationStore.getState().markAsRead((message.data as { id: string }).id);
            break;
          case 'unread.count':
            useNotificationStore.getState().setUnreadCount(
              Number((message.data as { count: number }).count ?? 0),
            );
            break;
          case 'connection.ack':
            break;
          default:
            break;
        }
      } catch {
        // Ignore malformed websocket payloads.
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (intentionalCloseRef.current) {
        return;
      }

      const store = useNotificationStore.getState();
      if (attemptRef.current < MAX_RECONNECT_ATTEMPTS) {
        const delay = getBackoffDelay(attemptRef.current);
        const nextAttempt = attemptRef.current + 1;
        attemptRef.current = nextAttempt;
        store.setConnectionStatus('reconnecting');
        store.setReconnectState(nextAttempt, Date.now() + delay);
        reconnectTimerRef.current = setTimeout(() => connect(), delay);
      } else {
        store.setConnectionStatus('failed');
        store.setReconnectState(MAX_RECONNECT_ATTEMPTS, null);
      }
    };

    ws.onerror = () => {
      useNotificationStore.getState().setConnectionStatus('disconnected');
    };
  }, []);

  useEffect(() => {
    connect();

    return () => {
      intentionalCloseRef.current = true;
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmount');
        wsRef.current = null;
      }
    };
  }, [connect]);

  useEffect(() => {
    if (reconnectToken === 0) {
      return;
    }

    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (wsRef.current) {
      intentionalCloseRef.current = true;
      wsRef.current.close(1000, 'Manual reconnect');
      wsRef.current = null;
    }
    attemptRef.current = 0;
    intentionalCloseRef.current = false;
    connect();
  }, [connect, reconnectToken]);
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
    read: notification.read ?? isNotificationRead(notification as Notification),
    read_at: notification.read_at ?? null,
    created_at: notification.created_at ?? new Date().toISOString(),
  };
}

function normalizeTopics(type: string | undefined): string[] {
  if (!type) {
    return [];
  }

  const aliases: Record<string, string[]> = {
    'alert.created': ['alert.created'],
    'alert.escalated': ['alert.escalated'],
    'task.assigned': ['task.assigned', 'workflow.task.created'],
    'task.completed': ['task.completed', 'workflow.task.completed'],
    'task.overdue': ['task.overdue', 'workflow.task.escalated'],
    'task.escalated': ['task.escalated', 'workflow.task.escalated'],
    'workflow.task.created': ['workflow.task.created', 'task.assigned'],
    'workflow.task.completed': ['workflow.task.completed', 'task.completed'],
    'workflow.task.escalated': ['workflow.task.escalated', 'task.escalated', 'task.overdue'],
    'pipeline.failed': ['pipeline.failed'],
    'pipeline.completed': ['pipeline.completed'],
    'data_quality.issue_detected': ['data_quality.issue_detected'],
    'remediation.approval_required': ['remediation.approval_required'],
    'security.incident': ['security.incident'],
    'cyber.rule.created': ['cyber.rule.created', 'cyber.rule.updated'],
    'cyber.rule.updated': ['cyber.rule.updated'],
    'cyber.rule.toggled': ['cyber.rule.toggled', 'cyber.rule.updated'],
    'cyber.rule.deleted': ['cyber.rule.deleted', 'cyber.rule.updated'],
  };

  return aliases[type] ?? [type];
}
