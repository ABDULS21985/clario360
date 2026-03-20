'use client';

import { EmptyState } from '@/components/common/empty-state';
import { FileSearch } from 'lucide-react';
import type { AlertExplanation } from '@/types/cyber';

interface AlertEvidenceTabProps {
  explanation: AlertExplanation;
}

export function AlertEvidenceTab({ explanation }: AlertEvidenceTabProps) {
  const { evidence, indicator_matches } = explanation;
  const hasContent = (evidence?.length ?? 0) > 0 || (indicator_matches?.length ?? 0) > 0;

  if (!hasContent) {
    return (
      <EmptyState
        icon={FileSearch}
        title="No evidence collected"
        description="No structured evidence was captured for this alert."
      />
    );
  }

  return (
    <div className="space-y-6">
      {evidence?.length > 0 && (
        <div>
          <h4 className="mb-3 text-sm font-semibold">Forensic Evidence</h4>
          <div className="overflow-hidden rounded-xl border">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/30">
                <tr>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Field</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Value</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Description</th>
                </tr>
              </thead>
              <tbody>
                {evidence.map((ev, i) => (
                  <tr key={i} className="border-b last:border-0 hover:bg-muted/20">
                    <td className="px-4 py-2.5">
                      <div>
                        <p className="font-mono text-xs text-muted-foreground">{ev.field}</p>
                        <p className="text-xs font-medium">{ev.label}</p>
                      </div>
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs break-all">
                      {typeof ev.value === 'object' ? JSON.stringify(ev.value) : String(ev.value)}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-muted-foreground hidden lg:table-cell">
                      {ev.description}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {(indicator_matches?.length ?? 0) > 0 && (
        <div>
          <h4 className="mb-3 text-sm font-semibold">Threat Intelligence Indicators</h4>
          <div className="overflow-hidden rounded-xl border">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/30">
                <tr>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Type</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Value</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Source</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Confidence</th>
                </tr>
              </thead>
              <tbody>
                {indicator_matches!.map((ind, i) => (
                  <tr key={i} className="border-b last:border-0 hover:bg-muted/20">
                    <td className="px-4 py-2.5">
                      <span className="rounded bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900/30 dark:text-amber-300">
                        {ind.type}
                      </span>
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs">{ind.value}</td>
                    <td className="px-4 py-2.5 text-xs text-muted-foreground">{ind.source}</td>
                    <td className="px-4 py-2.5 text-xs font-medium">{Math.round(ind.confidence * 100)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
