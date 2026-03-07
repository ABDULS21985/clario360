'use client';

import { useState } from 'react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { ChevronRight, Zap } from 'lucide-react';
import { FindingDetailPanel } from './finding-detail-panel';
import type { CTEMFinding } from '@/types/cyber';

const STATUS_STYLES: Record<string, string> = {
  open: 'bg-red-100 text-red-800',
  in_remediation: 'bg-blue-100 text-blue-800',
  resolved: 'bg-green-100 text-green-800',
  accepted: 'bg-gray-100 text-gray-800',
  false_positive: 'bg-muted text-muted-foreground',
};

interface FindingTableProps {
  findings: CTEMFinding[];
}

export function FindingTable({ findings }: FindingTableProps) {
  const [selected, setSelected] = useState<CTEMFinding | null>(null);

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
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Asset</th>
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
                    {finding.exploit_available && (
                      <span title="Exploit available"><Zap className="h-3.5 w-3.5 text-red-500" /></span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-3 hidden md:table-cell text-muted-foreground text-xs">
                  {finding.asset_name ?? '—'}
                </td>
                <td className="px-4 py-3">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLES[finding.status] ?? ''}`}>
                    {finding.status.replace('_', ' ')}
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
                    <span className="text-xs tabular-nums">{finding.priority_score}</span>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="lg:col-span-1">
          <FindingDetailPanel finding={selected} onClose={() => setSelected(null)} />
        </div>
      )}
    </div>
  );
}
