'use client';

import { EmptyState } from '@/components/common/empty-state';
import { Settings2 } from 'lucide-react';
import type { CyberAsset } from '@/types/cyber';

interface AssetConfigTabProps {
  asset: CyberAsset;
}

export function AssetConfigTab({ asset }: AssetConfigTabProps) {
  const metadata = asset.metadata;

  if (!metadata || Object.keys(metadata).length === 0) {
    return (
      <EmptyState
        icon={Settings2}
        title="No configuration data"
        description="Configuration metadata will appear here once a configuration scan has been run on this asset."
      />
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">Configuration metadata collected during discovery or last scan.</p>
      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/30">
            <tr>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground w-1/3">Key</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Value</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(metadata).map(([key, value]) => (
              <tr key={key} className="border-b last:border-0 hover:bg-muted/20">
                <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{key}</td>
                <td className="px-4 py-2.5 font-mono text-xs break-all">
                  {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
