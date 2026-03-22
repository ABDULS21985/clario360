'use client';

import {
  AlertCircle,
  Calendar,
  Clock,
  Database,
  Hash,
  RefreshCw,
  Settings,
  Trash2,
  Unplug,
} from 'lucide-react';

import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { integrationStatusConfig, integrationHealthConfig } from '@/lib/status-configs';
import { formatDate, formatDateTime, formatCompactNumber, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { VCISOIntegration } from '@/types/cyber';

import { TYPE_ICON_MAP, TYPE_COLOR_MAP } from './integration-card';

// ─── Props ───────────────────────────────────────────────────────────────────

interface IntegrationDetailPanelProps {
  integration: VCISOIntegration | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSyncNow?: (integration: VCISOIntegration) => void;
  onConfigure?: (integration: VCISOIntegration) => void;
  onDisconnect?: (integration: VCISOIntegration) => void;
  onRemove?: (integration: VCISOIntegration) => void;
  /** ID of the integration currently being synced (null if none). */
  syncingId?: string | null;
}

// ─── Helper ──────────────────────────────────────────────────────────────────

function redactConfigValue(key: string, value: unknown): string {
  const sensitiveKeys = ['key', 'secret', 'password', 'token', 'credential'];
  const lowerKey = key.toLowerCase();
  if (sensitiveKeys.some((s) => lowerKey.includes(s)) && typeof value === 'string' && value.length > 0) {
    return value.slice(0, 4) + '****';
  }
  if (typeof value === 'object' && value !== null) {
    return JSON.stringify(value);
  }
  return String(value);
}

// ─── Component ───────────────────────────────────────────────────────────────

export function IntegrationDetailPanel({
  integration,
  open,
  onOpenChange,
  onSyncNow,
  onConfigure,
  onDisconnect,
  onRemove,
  syncingId = null,
}: IntegrationDetailPanelProps) {
  if (!integration) return null;

  const TypeIcon = TYPE_ICON_MAP[integration.type] ?? Database;
  const typeColor = TYPE_COLOR_MAP[integration.type] ?? TYPE_COLOR_MAP.siem;
  const configEntries = Object.entries(integration.config);
  const isSyncing = syncingId === integration.id;
  const isDisconnected = integration.status === 'disconnected';

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title={integration.name}
      description={`${integration.provider} - ${titleCase(integration.type)}`}
      width="lg"
    >
      <div className="space-y-6">
        {/* Header badges */}
        <div className="flex items-center gap-2 flex-wrap">
          <div
            className={cn(
              'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
              typeColor,
            )}
          >
            <TypeIcon className="h-4 w-4" aria-hidden />
          </div>
          <StatusBadge
            status={integration.status}
            config={integrationStatusConfig}
          />
          <StatusBadge
            status={integration.health_status}
            config={integrationHealthConfig}
            variant="outline"
          />
          <Badge variant="outline" className="text-xs capitalize">
            {titleCase(integration.type)}
          </Badge>
        </div>

        {/* Error message */}
        {integration.status === 'error' && integration.error_message && (
          <div className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-900/10 p-3">
            <div className="flex items-start gap-2">
              <AlertCircle className="h-4 w-4 text-red-600 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-red-800 dark:text-red-300">Error</p>
                <p className="text-xs text-red-700 dark:text-red-400 mt-0.5 leading-relaxed">
                  {integration.error_message}
                </p>
              </div>
            </div>
          </div>
        )}

        <Separator />

        {/* Sync details */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
            <RefreshCw className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
            Sync Information
          </h4>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="rounded-lg border p-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <Database className="h-3 w-3" />
                Items Synced
              </p>
              <p className="text-lg font-semibold mt-0.5 tabular-nums">
                {formatCompactNumber(integration.items_synced)}
              </p>
            </div>
            <div className="rounded-lg border p-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <Clock className="h-3 w-3" />
                Sync Frequency
              </p>
              <p className="text-sm font-medium mt-0.5">
                {titleCase(integration.sync_frequency)}
              </p>
            </div>
            <div className="rounded-lg border p-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <Calendar className="h-3 w-3" />
                Last Sync
              </p>
              <p className="text-sm font-medium mt-0.5">
                {integration.last_sync_at
                  ? formatDateTime(integration.last_sync_at)
                  : 'Never'}
              </p>
            </div>
            <div className="rounded-lg border p-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <Hash className="h-3 w-3" />
                Provider
              </p>
              <p className="text-sm font-medium mt-0.5">{integration.provider}</p>
            </div>
          </div>
        </div>

        <Separator />

        {/* Configuration */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
            <Settings className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
            Configuration ({configEntries.length} {configEntries.length === 1 ? 'parameter' : 'parameters'})
          </h4>
          {configEntries.length > 0 ? (
            <div className="space-y-1.5 rounded-lg border p-3">
              {configEntries.map(([key, value]) => (
                <div key={key} className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground font-mono text-xs">{key}</span>
                  <span className="font-medium font-mono text-xs truncate max-w-[120px] sm:max-w-[200px]">
                    {redactConfigValue(key, value)}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">No configuration parameters set</p>
          )}
        </div>

        <Separator />

        {/* Timestamps */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
            Timeline
          </h4>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="rounded-lg border p-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <Calendar className="h-3 w-3" />
                Created
              </p>
              <p className="text-sm font-medium mt-0.5">{formatDate(integration.created_at)}</p>
            </div>
            <div className="rounded-lg border p-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <Calendar className="h-3 w-3" />
                Updated
              </p>
              <p className="text-sm font-medium mt-0.5">{formatDate(integration.updated_at)}</p>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-col gap-2">
          <Button
            className="w-full"
            disabled={isSyncing || isDisconnected}
            onClick={() => onSyncNow?.(integration)}
          >
            <RefreshCw className={cn('mr-1.5 h-4 w-4', isSyncing && 'animate-spin')} />
            {isSyncing ? 'Syncing...' : 'Sync Now'}
          </Button>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <Button
              variant="outline"
              onClick={() => onConfigure?.(integration)}
            >
              <Settings className="mr-1.5 h-4 w-4" />
              Configure
            </Button>
            <Button
              variant="outline"
              className="text-amber-600 hover:text-amber-600 dark:text-amber-400 dark:hover:text-amber-400"
              disabled={isDisconnected}
              onClick={() => onDisconnect?.(integration)}
            >
              <Unplug className="mr-1.5 h-4 w-4" />
              Disconnect
            </Button>
            <Button
              variant="outline"
              className="text-destructive hover:text-destructive"
              onClick={() => onRemove?.(integration)}
            >
              <Trash2 className="mr-1.5 h-4 w-4" />
              Remove
            </Button>
          </div>
        </div>
      </div>
    </DetailPanel>
  );
}
