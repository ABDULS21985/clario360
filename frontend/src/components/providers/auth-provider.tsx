'use client';

import React, { useEffect, useRef } from 'react';
import { useAuthStore } from '@/stores/auth-store';
import { getAccessToken, isTokenExpired } from '@/lib/auth';
import { Spinner } from '@/components/ui/spinner';

interface AuthProviderProps {
  children: React.ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const isHydrated = useAuthStore((s) => s.isHydrated);
  const refreshSession = useAuthStore((s) => s.refreshSession);
  const setSessionExpired = useAuthStore((s) => s.setSessionExpired);
  const refreshingRef = useRef(false);

  // Hydrate session on mount
  useEffect(() => {
    refreshSession();
  }, [refreshSession]);

  // Listen for session-expired events fired by the Axios interceptor
  useEffect(() => {
    const handler = () => setSessionExpired(true);
    window.addEventListener('clario360:session-expired', handler);
    return () => window.removeEventListener('clario360:session-expired', handler);
  }, [setSessionExpired]);

  // Silent refresh on window focus / visibility change
  useEffect(() => {
    const silentRefresh = async () => {
      if (refreshingRef.current) return;
      const token = getAccessToken();
      if (token && isTokenExpired(token)) {
        refreshingRef.current = true;
        try {
          await refreshSession();
        } finally {
          refreshingRef.current = false;
        }
      }
    };

    const onFocus = () => void silentRefresh();
    const onVisible = () => {
      if (document.visibilityState === 'visible') void silentRefresh();
    };

    window.addEventListener('focus', onFocus);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      window.removeEventListener('focus', onFocus);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [refreshSession]);

  if (!isHydrated) {
    return (
      <div className="fixed inset-0 flex flex-col items-center justify-center bg-brand-teal-dark">
        <div className="mb-6 text-center">
          <h1 className="text-3xl font-bold text-white">Clario 360</h1>
          <p className="mt-1 text-sm text-brand-gold">Enterprise AI Platform</p>
        </div>
        <Spinner size="lg" className="text-white" />
      </div>
    );
  }

  return <>{children}</>;
}
