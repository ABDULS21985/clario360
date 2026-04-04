'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { getAccessToken } from '@/lib/auth';
import { useCTIStore } from '@/stores/cti-store';
import type { CTIThreatEvent, CTIWebSocketMessage } from '@/types/cti';

type CTIWebSocketStatus = 'idle' | 'connecting' | 'connected' | 'error' | 'closed';

function buildWebSocketUrl(token: string): string {
  const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
  const protocol = baseUrl.startsWith('https') ? 'wss' : 'ws';
  const host = baseUrl.replace(/^https?:\/\//, '');
  return `${protocol}://${host}/ws/v1/cyber/cti/ws?token=${encodeURIComponent(token)}`;
}

export function useCTIWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const refreshRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [status, setStatus] = useState<CTIWebSocketStatus>('idle');
  const pushLiveEvent = useCTIStore((state) => state.pushLiveEvent);
  const refreshExecutiveSnapshot = useCTIStore((state) => state.refreshExecutiveSnapshot);

  const scheduleRefresh = useCallback(() => {
    if (refreshRef.current) {
      clearTimeout(refreshRef.current);
    }
    refreshRef.current = setTimeout(() => {
      void refreshExecutiveSnapshot();
    }, 400);
  }, [refreshExecutiveSnapshot]);

  const connect = useCallback(() => {
    const token = getAccessToken();
    if (!token || typeof window === 'undefined') {
      setStatus('closed');
      return;
    }

    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      return;
    }

    setStatus('connecting');
    const socket = new WebSocket(buildWebSocketUrl(token));
    wsRef.current = socket;

    socket.onopen = () => {
      setStatus('connected');
      window.console.info('[CTI WS] connected');
    };

    socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as CTIWebSocketMessage<CTIThreatEvent>;
        const type = message.type ?? '';

        if (type.includes('threat-event.created') || type.includes('threat-event.updated')) {
          pushLiveEvent(message.data);
          scheduleRefresh();
          return;
        }

        if (
          type.includes('campaign.created') ||
          type.includes('campaign.updated') ||
          type.includes('campaign.status-changed') ||
          type.includes('campaign.event-linked') ||
          type.includes('brand-abuse.detected') ||
          type.includes('brand-abuse.updated') ||
          type.includes('brand-abuse.takedown-changed')
        ) {
          scheduleRefresh();
        }
      } catch (error) {
        window.console.error('[CTI WS] message parse failed', error);
      }
    };

    socket.onerror = (error) => {
      setStatus('error');
      window.console.error('[CTI WS] connection error', error);
    };

    socket.onclose = (event) => {
      setStatus('closed');
      window.console.info('[CTI WS] disconnected', { code: event.code, clean: event.wasClean });
      if (!event.wasClean) {
        reconnectRef.current = setTimeout(connect, 3000);
      }
    };
  }, [pushLiveEvent, scheduleRefresh]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectRef.current) {
        clearTimeout(reconnectRef.current);
      }
      if (refreshRef.current) {
        clearTimeout(refreshRef.current);
      }
      wsRef.current?.close();
    };
  }, [connect]);

  return {
    ws: wsRef.current,
    status,
  };
}
