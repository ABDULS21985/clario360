'use client';

import { CheckCircle, XCircle, AlertTriangle } from 'lucide-react';
import type { DryRunResult } from '@/types/cyber';

interface DryRunResultsPanelProps {
  result: DryRunResult;
}

export function DryRunResultsPanel({ result }: DryRunResultsPanelProps) {
  return (
    <div className="space-y-4">
      {/* Overall status */}
      <div className={`flex items-center gap-3 rounded-xl border p-4 ${result.success ? 'border-green-200 bg-green-50 dark:bg-green-950/20' : 'border-red-200 bg-red-50 dark:bg-red-950/20'}`}>
        {result.success
          ? <CheckCircle className="h-6 w-6 text-green-600 shrink-0" />
          : <XCircle className="h-6 w-6 text-red-600 shrink-0" />
        }
        <div>
          <p className={`font-semibold ${result.success ? 'text-green-800 dark:text-green-300' : 'text-red-800 dark:text-red-300'}`}>
            Dry Run {result.success ? 'Succeeded' : 'Failed'}
          </p>
          <p className="text-xs text-muted-foreground">
            {result.simulated_changes.length} changes simulated in {(result.duration_ms / 1000).toFixed(1)}s
          </p>
        </div>
      </div>

      {/* Blockers */}
      {result.blockers.length > 0 && (
        <div className="rounded-lg border border-red-200 bg-red-50/50 p-3 dark:border-red-900 dark:bg-red-950/20">
          <p className="mb-2 text-xs font-semibold text-red-700">Blockers</p>
          {result.blockers.map((b, i) => (
            <div key={i} className="flex items-start gap-2 text-xs text-red-700">
              <XCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              {b}
            </div>
          ))}
        </div>
      )}

      {/* Warnings */}
      {result.warnings.length > 0 && (
        <div className="rounded-lg border border-amber-200 bg-amber-50/50 p-3 dark:border-amber-900 dark:bg-amber-950/20">
          <p className="mb-2 text-xs font-semibold text-amber-700">Warnings</p>
          {result.warnings.map((w, i) => (
            <div key={i} className="flex items-start gap-2 text-xs text-amber-700">
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              {w}
            </div>
          ))}
        </div>
      )}

      {/* Simulated changes */}
      {result.simulated_changes.length > 0 && (
        <div>
          <p className="mb-2 text-xs font-semibold">Simulated Changes</p>
          <div className="space-y-1.5">
            {result.simulated_changes.map((c, i) => (
              <div key={i} className="rounded-lg border p-2.5 text-xs">
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium">{c.asset_name}</span>
                  <span className="rounded bg-muted px-1.5 py-0.5 capitalize">{c.change_type}</span>
                </div>
                <p className="mt-0.5 text-muted-foreground">{c.description}</p>
                {(c.before_value || c.after_value) && (
                  <div className="mt-1 flex items-center gap-2 font-mono">
                    {c.before_value && <span className="rounded bg-red-100 px-1 text-red-700 line-through dark:bg-red-950/30 dark:text-red-400">{c.before_value}</span>}
                    {c.after_value && <span className="rounded bg-green-100 px-1 text-green-700 dark:bg-green-950/30 dark:text-green-400">{c.after_value}</span>}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Impact estimate */}
      <div className="rounded-lg border p-3">
        <p className="mb-2 text-xs font-semibold">Estimated Impact</p>
        <div className="grid grid-cols-2 gap-2 text-xs">
          <div>
            <p className="text-muted-foreground">Downtime</p>
            <p className="font-medium">{result.estimated_impact.downtime}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Services Affected</p>
            <p className="font-medium">{result.estimated_impact.services_affected}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Risk Level</p>
            <p className="font-medium capitalize">{result.estimated_impact.risk_level}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Recommended Window</p>
            <p className="font-medium">{result.estimated_impact.recommend_window}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
