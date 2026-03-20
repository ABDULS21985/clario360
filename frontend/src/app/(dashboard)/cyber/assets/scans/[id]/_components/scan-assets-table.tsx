'use client';

import { AlertCircle } from 'lucide-react';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { EmptyState } from '@/components/common/empty-state';

export interface ScannedAsset {
  id: string;
  name: string;
  type: string;
  ip_address?: string;
  status: string;
  criticality: string;
}

interface Props {
  assets: ScannedAsset[];
  isLoading: boolean;
}

const CRITICALITY_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  high: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
  medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  low: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
};

export function ScanAssetsTable({ assets, isLoading }: Props) {
  if (isLoading) {
    return <LoadingSkeleton variant="table-row" />;
  }

  if (assets.length === 0) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="No assets discovered"
        description="This scan did not discover any assets, or they have not been loaded yet."
      />
    );
  }

  return (
    <div className="rounded-xl border bg-card">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                Name
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                Type
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                IP Address
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                Status
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                Criticality
              </th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {assets.map((asset) => (
              <tr key={asset.id} className="transition-colors hover:bg-muted/30">
                <td className="px-4 py-3 font-medium">{asset.name}</td>
                <td className="px-4 py-3">
                  <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">
                    {asset.type.replace(/_/g, ' ')}
                  </span>
                </td>
                <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                  {asset.ip_address ?? '—'}
                </td>
                <td className="px-4 py-3">
                  <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">
                    {asset.status}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${
                      CRITICALITY_COLORS[asset.criticality] ?? 'bg-muted text-muted-foreground'
                    }`}
                  >
                    {asset.criticality}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
