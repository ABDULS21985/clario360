'use client';

import {
  Cloud,
  Database,
  Key,
  Monitor,
  MoreHorizontal,
  RefreshCw,
  Settings,
  Shield,
  TicketCheck,
  Eye,
  Unplug,
  Trash2,
  type LucideIcon,
} from 'lucide-react';

import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { integrationStatusConfig, integrationHealthConfig } from '@/lib/status-configs';
import { formatRelativeTime, formatCompactNumber, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { VCISOIntegration, CyberIntegrationType } from '@/types/cyber';

// ─── Type-to-Icon Map ───────────────────────────────────────────────────────

const TYPE_ICON_MAP: Record<CyberIntegrationType, LucideIcon> = {
  ticketing: TicketCheck,
  cloud_security: Cloud,
  asset_management: Database,
  data_protection: Shield,
  siem: Monitor,
  iam: Key,
};

const TYPE_COLOR_MAP: Record<CyberIntegrationType, string> = {
  ticketing: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  cloud_security: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400',
  asset_management: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  data_protection: 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400',
  siem: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400',
  iam: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
};

const HEALTH_DOT_MAP: Record<string, string> = {
  healthy: 'bg-green-500',
  degraded: 'bg-yellow-500',
  unavailable: 'bg-red-500',
};

// ─── Props ───────────────────────────────────────────────────────────────────

interface IntegrationCardProps {
  integration: VCISOIntegration;
  onViewDetails: (integration: VCISOIntegration) => void;
  onConfigure: (integration: VCISOIntegration) => void;
  onSyncNow: (integration: VCISOIntegration) => void;
  onDisconnect: (integration: VCISOIntegration) => void;
  onRemove: (integration: VCISOIntegration) => void;
  /** ID of the integration currently being synced (null if none). */
  syncingId?: string | null;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function IntegrationCard({
  integration,
  onViewDetails,
  onConfigure,
  onSyncNow,
  onDisconnect,
  onRemove,
  syncingId = null,
}: IntegrationCardProps) {
  const TypeIcon = TYPE_ICON_MAP[integration.type] ?? Monitor;
  const typeColor = TYPE_COLOR_MAP[integration.type] ?? TYPE_COLOR_MAP.siem;
  const healthDot = HEALTH_DOT_MAP[integration.health_status] ?? HEALTH_DOT_MAP.unavailable;
  const isSyncing = syncingId === integration.id;
  const isDisconnected = integration.status === 'disconnected';

  return (
    <Card
      className={cn(
        'group relative overflow-hidden transition-shadow hover:shadow-md cursor-pointer',
        integration.status === 'error' && 'border-red-200 dark:border-red-800/40',
      )}
      onClick={() => onViewDetails(integration)}
    >
      <CardHeader className="flex flex-row items-start justify-between gap-3 pb-3 space-y-0">
        <div className="flex items-center gap-3 min-w-0">
          <div
            className={cn(
              'flex h-10 w-10 shrink-0 items-center justify-center rounded-lg',
              typeColor,
            )}
          >
            <TypeIcon className="h-5 w-5" aria-hidden />
          </div>
          <div className="min-w-0 flex-1">
            <h3 className="text-sm font-semibold leading-tight truncate">
              {integration.name}
            </h3>
            <p className="text-xs text-muted-foreground mt-0.5 truncate">
              {integration.provider}
            </p>
          </div>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
              onClick={(e) => e.stopPropagation()}
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
                onViewDetails(integration);
              }}
            >
              <Eye className="mr-2 h-3.5 w-3.5" />
              View Details
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
                onConfigure(integration);
              }}
            >
              <Settings className="mr-2 h-3.5 w-3.5" />
              Configure
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={isSyncing || isDisconnected}
              onClick={(e) => {
                e.stopPropagation();
                onSyncNow(integration);
              }}
            >
              <RefreshCw className={cn('mr-2 h-3.5 w-3.5', isSyncing && 'animate-spin')} />
              {isSyncing ? 'Syncing...' : 'Sync Now'}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-amber-600 dark:text-amber-400"
              disabled={isDisconnected}
              onClick={(e) => {
                e.stopPropagation();
                onDisconnect(integration);
              }}
            >
              <Unplug className="mr-2 h-3.5 w-3.5" />
              Disconnect
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-destructive"
              onClick={(e) => {
                e.stopPropagation();
                onRemove(integration);
              }}
            >
              <Trash2 className="mr-2 h-3.5 w-3.5" />
              Remove
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </CardHeader>

      <CardContent className="space-y-3 pt-0">
        {/* Status and Health */}
        <div className="flex items-center gap-2 flex-wrap">
          <StatusBadge
            status={integration.status}
            config={integrationStatusConfig}
            size="sm"
          />
          <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
            <span className={cn('h-1.5 w-1.5 rounded-full shrink-0', healthDot)} aria-hidden />
            {integrationHealthConfig[integration.health_status]?.label ?? titleCase(integration.health_status)}
          </span>
        </div>

        {/* Error message */}
        {integration.status === 'error' && integration.error_message && (
          <p className="text-xs text-red-600 dark:text-red-400 line-clamp-2 leading-relaxed">
            {integration.error_message}
          </p>
        )}

        {/* Metrics row */}
        <div className="flex items-center justify-between text-xs text-muted-foreground border-t pt-3">
          <div className="flex flex-col">
            <span className="text-foreground font-semibold tabular-nums">
              {formatCompactNumber(integration.items_synced)}
            </span>
            <span>items synced</span>
          </div>
          <div className="flex flex-col items-center">
            <Badge variant="outline" className="text-[10px] px-1.5 py-0">
              {titleCase(integration.sync_frequency)}
            </Badge>
            <span className="mt-0.5">frequency</span>
          </div>
          <div className="flex flex-col items-end">
            <span className="text-foreground font-medium">
              {integration.last_sync_at
                ? formatRelativeTime(integration.last_sync_at)
                : 'Never'}
            </span>
            <span>last sync</span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Re-export the icon map for use in other components
export { TYPE_ICON_MAP, TYPE_COLOR_MAP };
