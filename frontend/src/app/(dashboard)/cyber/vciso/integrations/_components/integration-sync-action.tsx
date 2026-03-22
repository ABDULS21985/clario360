'use client';

import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { cn } from '@/lib/utils';
import type { VCISOIntegration } from '@/types/cyber';

// ─── Props ───────────────────────────────────────────────────────────────────

interface IntegrationSyncActionProps {
  integration: VCISOIntegration;
  variant?: 'default' | 'outline' | 'ghost';
  size?: 'default' | 'sm' | 'icon';
  className?: string;
  onSynced?: () => void;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function IntegrationSyncAction({
  integration,
  variant = 'outline',
  size = 'sm',
  className,
  onSynced,
}: IntegrationSyncActionProps) {
  const { mutate: triggerSync, isPending: syncing } = useApiMutation<
    unknown,
    { id: string }
  >(
    'post',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS}/${variables.id}/sync`,
    {
      successMessage: 'Sync triggered successfully',
      invalidateKeys: [API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS],
      onSuccess: () => {
        onSynced?.();
      },
    },
  );

  const isDisabled = syncing || integration.status === 'disconnected';

  return (
    <Button
      variant={variant}
      size={size}
      className={className}
      disabled={isDisabled}
      onClick={(e) => {
        e.stopPropagation();
        triggerSync({ id: integration.id });
      }}
    >
      <RefreshCw
        className={cn('h-3.5 w-3.5', size !== 'icon' && 'mr-1.5', syncing && 'animate-spin')}
      />
      {size !== 'icon' && (syncing ? 'Syncing...' : 'Sync Now')}
    </Button>
  );
}

// ─── Hook for parent usage ────────────────────────────────────────────────────
// Returns a function to trigger sync for a specific integration and tracks
// which integration ID is currently syncing (not a global flag).

export function useSyncIntegration(onSynced?: () => void) {
  const { mutate, isPending, variables } = useApiMutation<unknown, { id: string }>(
    'post',
    (vars) => `${API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS}/${vars.id}/sync`,
    {
      successMessage: 'Sync triggered successfully',
      invalidateKeys: [API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS],
      onSuccess: () => {
        onSynced?.();
      },
    },
  );

  return {
    triggerSync: (integration: VCISOIntegration) => mutate({ id: integration.id }),
    syncing: isPending,
    syncingId: isPending ? (variables?.id ?? null) : null,
  };
}
