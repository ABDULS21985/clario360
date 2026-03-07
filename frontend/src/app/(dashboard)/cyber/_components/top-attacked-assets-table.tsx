'use client';

import Link from 'next/link';
import { Server, Monitor, Cloud, Router, Wifi, AppWindow, Database, Box, AlertTriangle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/components/common/empty-state';
import type { AssetAlertSummary, Criticality } from '@/types/cyber';

interface TopAttackedAssetsTableProps {
  assets: AssetAlertSummary[];
}

const TYPE_ICONS: Record<string, React.ElementType> = {
  server: Server,
  endpoint: Monitor,
  cloud_resource: Cloud,
  network_device: Router,
  iot_device: Wifi,
  application: AppWindow,
  database: Database,
  container: Box,
};

const CRITICALITY_COLORS: Record<Criticality, string> = {
  critical: 'bg-red-100 text-red-800',
  high: 'bg-orange-100 text-orange-800',
  medium: 'bg-yellow-100 text-yellow-800',
  low: 'bg-blue-100 text-blue-800',
};

function alertCountColor(count: number): string {
  if (count >= 10) return 'text-red-600 font-semibold';
  if (count >= 5) return 'text-orange-600 font-medium';
  if (count >= 1) return 'text-yellow-600';
  return 'text-muted-foreground';
}

export function TopAttackedAssetsTable({ assets }: TopAttackedAssetsTableProps) {
  if (assets.length === 0) {
    return (
      <EmptyState
        icon={AlertTriangle}
        title="No attacked assets"
        description="No assets with active alerts found."
      />
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <table className="w-full text-sm">
        <thead className="border-b bg-muted/30">
          <tr>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Asset</th>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Criticality</th>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Alerts</th>
          </tr>
        </thead>
        <tbody>
          {assets.map((asset) => {
            const Icon = TYPE_ICONS[asset.asset_type] ?? Server;
            return (
              <tr key={asset.asset_id} className="border-b last:border-0 hover:bg-muted/20">
                <td className="px-3 py-2">
                  <Link href={`/cyber/assets/${asset.asset_id}`} className="flex items-center gap-2 hover:underline">
                    <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                    <span className="font-medium truncate max-w-[140px]">{asset.asset_name}</span>
                  </Link>
                </td>
                <td className="px-3 py-2">
                  <span className={cn('inline-flex rounded-full px-1.5 py-0.5 text-xs font-medium capitalize', CRITICALITY_COLORS[asset.criticality as Criticality] ?? 'bg-gray-100 text-gray-700')}>
                    {asset.criticality}
                  </span>
                </td>
                <td className="px-3 py-2">
                  <span className={alertCountColor(asset.alert_count)}>{asset.alert_count}</span>
                  {asset.critical_open > 0 && (
                    <span className="ml-1 text-xs text-red-600">({asset.critical_open} crit)</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
