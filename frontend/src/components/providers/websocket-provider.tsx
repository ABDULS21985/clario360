'use client';

import { useEffect } from 'react';
import { useAuth } from '@/hooks/use-auth';
import { useWebSocket } from '@/hooks/use-websocket';
import { useNotificationStore } from '@/stores/notification-store';

function WebSocketInner() {
  useWebSocket();
  const { fetchInitialData } = useNotificationStore();

  useEffect(() => {
    fetchInitialData();
  }, [fetchInitialData]);

  return null;
}

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isHydrated } = useAuth();

  return (
    <>
      {isHydrated && isAuthenticated && <WebSocketInner />}
      {children}
    </>
  );
}
