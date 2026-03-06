'use client';

import { useState } from 'react';
import { Wifi, WifiOff, X } from 'lucide-react';
import { useNotificationStore } from '@/stores/notification-store';
import { cn } from '@/lib/utils';

export function ConnectionStatusBanner() {
  const connectionStatus = useNotificationStore((s) => s.connectionStatus);
  const [dismissed, setDismissed] = useState(false);

  if (connectionStatus === 'connected' || connectionStatus === 'disconnected') {
    return null;
  }
  if (connectionStatus === 'failed' && dismissed) {
    return null;
  }

  const isReconnecting = connectionStatus === 'reconnecting' || connectionStatus === 'connecting';
  const isFailed = connectionStatus === 'failed';

  return (
    <div
      role="alert"
      className={cn(
        'flex items-center gap-3 px-4 py-2 text-sm font-medium',
        isReconnecting && 'bg-amber-500 text-white',
        isFailed && 'bg-destructive text-destructive-foreground',
      )}
    >
      {isReconnecting ? (
        <Wifi className="h-4 w-4 animate-pulse" />
      ) : (
        <WifiOff className="h-4 w-4" />
      )}
      <span className="flex-1">
        {isReconnecting
          ? 'Connection lost. Reconnecting...'
          : 'Unable to connect to real-time updates. Refresh the page to try again.'}
      </span>
      {isFailed && (
        <button
          onClick={() => setDismissed(true)}
          aria-label="Dismiss"
          className="rounded p-0.5 hover:bg-white/20"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
