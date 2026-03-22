'use client';

import { useState, useMemo } from 'react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { cn } from '@/lib/utils';
import { Zap } from 'lucide-react';
import { FindingDetailPanel } from './finding-detail-panel';
import { useAssetNames } from '../_hooks/use-asset-names';
import type { CTEMFinding } from '@/types/cyber';

/** Status keys match backend CTEMFindingStatus values exactly */
const STATUS_STYLES: Record<string, string> = {
  open: 'bg-red-100 text-red-800 dark:bg-red-950/30 dark:text-red-400',
  in_remediation: 'bg-blue-100 text-blue-800 dark:bg-blue-950/30 dark:text-blue-400',
  remediated: 'bg-green-100 text-green-800 dark:bg-green-950/30 dark:text-green-400',
  accepted_risk: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
  false_positive: 'bg-muted text-muted-foreground',
  deferred: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-950/30 dark:text-yellow-400',
};

/** Format finding status for display */
function formatStatus(status: string): string {
  return status.replace(/_/g, ' ');
}

/** Derive "exploit available" from exploitability_score (backend has no explicit field) */
function isExploitAvailable(finding: CTEMFinding): boolean {
  return finding.exploitability_score >= 0.7;
}

/** Get asset display — resolve primary asset name or fall back to count */
function assetDisplay(finding: CTEMFinding, assetNames: Record<string, string>): string {
  if (finding.primary_asset_id && assetNames[finding.primary_asset_id]) {
    const extra = finding.affected_asset_count > 1 ? ` +${finding.affected_asset_count - 1}` : '';
    return `${assetNames[finding.primary_asset_id]}${extra}`;
  }
  if (finding.affected_asset_count > 0) return `${finding.affected_asset_count} asset${finding.affected_asset_count > 1 ? 's' : ''}`;
  return '—';
}

interface FindingTableProps {
  findings: CTEMFinding[];
  onStatusUpdated?: () => void;
}

export function FindingTable({ findings, onStatusUpdated }: FindingTableProps) {
  const [selected, setSelected] = useState<CTEMFinding | null>(null);

  // Collect primary asset IDs for name resolution
  const primaryAssetIds = useMemo(
    () => findings.map((f) => f.primary_asset_id).filter((id): id is string => !!id),
    [findings],
  );
  const assetNames = useAssetNames(primaryAssetIds);

  if (findings.length === 0) {
    return <p className="py-8 text-center text-sm text-muted-foreground">No findings in this assessment.</p>;
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <div className={cn('overflow-hidden rounded-xl border', selected ? 'lg:col-span-2' : 'lg:col-span-3')}>
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/30">
            <tr>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Severity</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Finding</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Assets</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Status</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Priority</th>
            </tr>
          </thead>
          <tbody>
            {findings.map((finding) => (
              <tr
                key={finding.id}
                className={cn(
                  'cursor-pointer border-b last:border-0 transition-colors hover:bg-muted/20',
                  selected?.id === finding.id && 'bg-primary/5',
                )}
                onClick={() => setSelected(selected?.id === finding.id ? null : finding)}
              >
                <td className="px-4 py-3">
                  <SeverityIndicator severity={finding.severity} />
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{finding.title}</span>
                    {isExploitAvailable(finding) && (
                      <span title="High exploitability"><Zap className="h-3.5 w-3.5 text-red-500" /></span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-3 hidden md:table-cell text-muted-foreground text-xs">
                  {assetDisplay(finding, assetNames)}
                </td>
                <td className="px-4 py-3">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLES[finding.status] ?? ''}`}>
                    {formatStatus(finding.status)}
                  </span>
                </td>
                <td className="px-4 py-3 hidden lg:table-cell">
                  <div className="flex items-center gap-2">
                    <div className="h-1.5 w-16 rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-orange-500"
                        style={{ width: `${finding.priority_score}%` }}
                      />
                    </div>
                    <span className="text-xs tabular-nums">{Math.round(finding.priority_score)}</span>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="lg:col-span-1">
          <FindingDetailPanel finding={selected} onClose={() => setSelected(null)} onStatusUpdated={onStatusUpdated} assetNames={assetNames} />
        </div>
      )}
    </div>
  );
}
