'use client';

import { useCallback } from 'react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import type { ConnectionStatus } from '@/types/models';

interface LiveIndicatorProps {
  status: ConnectionStatus;
  onReconnect?: () => void;
}

export function LiveIndicator({ status, onReconnect }: LiveIndicatorProps) {
  const handleClick = useCallback(() => {
    if ((status === 'disconnected' || status === 'failed') && onReconnect) {
      onReconnect();
    }
  }, [status, onReconnect]);

  const tooltipText: Record<ConnectionStatus, string> = {
    connected: 'Real-time updates active',
    connecting: 'Connecting...',
    reconnecting: 'Reconnecting...',
    disconnected: 'Disconnected. Click to reconnect.',
    failed: 'Connection failed. Refresh page to retry.',
  };

  const isClickable = status === 'disconnected' || status === 'failed';

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          onClick={handleClick}
          disabled={!isClickable}
          aria-label={tooltipText[status]}
          className={cn(
            'relative flex h-2 w-2 shrink-0 rounded-full',
            isClickable && 'cursor-pointer',
            !isClickable && 'cursor-default',
          )}
        >
          <span
            className={cn(
              'absolute inline-flex h-full w-full rounded-full',
              status === 'connected' && 'animate-ping bg-green-400 opacity-75',
              status === 'reconnecting' && 'animate-ping bg-yellow-400 opacity-75',
            )}
          />
          <span
            className={cn(
              'relative inline-flex h-2 w-2 rounded-full',
              status === 'connected' && 'bg-green-500',
              status === 'connecting' && 'bg-yellow-400',
              status === 'reconnecting' && 'bg-yellow-500',
              status === 'disconnected' && 'bg-red-500',
              status === 'failed' && 'bg-red-600',
            )}
          />
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <p className="text-xs">{tooltipText[status]}</p>
      </TooltipContent>
    </Tooltip>
  );
}
