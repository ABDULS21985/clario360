'use client';

import { useCallback } from 'react';
import { X } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import type { ConnectionStatus } from '@/types/models';

interface LiveIndicatorProps {
  status: ConnectionStatus;
  attempt?: number;
  onReconnect?: () => void;
}

export function LiveIndicator({ status, attempt, onReconnect }: LiveIndicatorProps) {
  const handleClick = useCallback(() => {
    if ((status === 'disconnected' || status === 'failed') && onReconnect) {
      onReconnect();
    }
  }, [status, onReconnect]);

  const tooltipText: Record<ConnectionStatus, string> = {
    connected: 'Real-time updates active',
    connecting: 'Connecting...',
    reconnecting: `Reconnecting...${attempt ? ` (attempt ${attempt})` : ''}`,
    disconnected: 'Disconnected. Click to reconnect.',
    failed: 'Connection failed. Refresh page to retry.',
  };

  const isClickable = status === 'disconnected' || status === 'failed';

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          onClick={handleClick}
          role={isClickable ? 'button' : undefined}
          tabIndex={isClickable ? 0 : undefined}
          aria-label={tooltipText[status]}
          className={cn(
            'relative flex h-2.5 w-2.5 shrink-0 items-center justify-center rounded-full',
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
          {status === 'failed' && (
            <X className="absolute h-2 w-2 text-white" />
          )}
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <p className="text-xs">{tooltipText[status]}</p>
      </TooltipContent>
    </Tooltip>
  );
}
