'use client';

import { useState, useEffect } from 'react';
import { WifiOff, RefreshCw, X, CheckCircle2 } from 'lucide-react';
import { useNotificationStore } from '@/stores/notification-store';
import { cn } from '@/lib/utils';

const BANNER_DISMISSED_KEY = 'clario360_banner_dismissed';

export function ConnectionBanner() {
  const connectionStatus = useNotificationStore((s) => s.connectionStatus);
  const [dismissed, setDismissed] = useState(false);
  const [showRestored, setShowRestored] = useState(false);
  const prevStatusRef = useState<string | null>(null);

  // Show "Connection restored" briefly when transitioning back to connected
  useEffect(() => {
    const prev = prevStatusRef[0];
    if (
      connectionStatus === 'connected' &&
      prev !== null &&
      prev !== 'connected' &&
      prev !== 'connecting'
    ) {
      setShowRestored(true);
      const timer = setTimeout(() => setShowRestored(false), 3000);
      return () => clearTimeout(timer);
    }
    prevStatusRef[1](connectionStatus);
  }, [connectionStatus, prevStatusRef]);

  // Dismiss resets on new disconnect cycle
  useEffect(() => {
    if (connectionStatus !== 'connected' && connectionStatus !== 'connecting') {
      const wasDismissed = sessionStorage.getItem(BANNER_DISMISSED_KEY) === '1';
      setDismissed(wasDismissed);
    }
  }, [connectionStatus]);

  const handleDismiss = () => {
    setDismissed(true);
    sessionStorage.setItem(BANNER_DISMISSED_KEY, '1');
  };

  if (showRestored) {
    return (
      <div
        role="status"
        aria-live="polite"
        className="flex h-10 items-center justify-center gap-2 bg-green-600 px-4 text-sm font-medium text-white"
      >
        <CheckCircle2 className="h-4 w-4" />
        Connection restored.
      </div>
    );
  }

  if (connectionStatus === 'connected' || connectionStatus === 'connecting') {
    return null;
  }

  if (dismissed) return null;

  const isReconnecting = connectionStatus === 'reconnecting';
  const isFailed = connectionStatus === 'failed';

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(
        'sticky top-0 z-50 flex h-10 items-center justify-center gap-3 px-4 text-sm font-medium',
        isReconnecting && 'bg-amber-500 text-white',
        isFailed && 'bg-destructive text-destructive-foreground',
      )}
    >
      <WifiOff className="h-4 w-4 shrink-0" />
      <span className="flex-1 text-center">
        {isReconnecting
          ? 'Connection lost. Attempting to reconnect...'
          : 'Unable to establish real-time connection. Some features may not update automatically.'}
      </span>
      {isFailed && (
        <div className="flex items-center gap-2">
          <button
            onClick={() => window.location.reload()}
            className="flex items-center gap-1 rounded px-2 py-0.5 text-xs hover:bg-white/20"
          >
            <RefreshCw className="h-3 w-3" />
            Refresh page
          </button>
          <button
            onClick={handleDismiss}
            aria-label="Dismiss banner"
            className="rounded p-0.5 hover:bg-white/20"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  );
}
