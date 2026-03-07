'use client';

import type { MITRECoverage } from '@/types/cyber';

interface MitreCoverageStatsProps {
  coverage: MITRECoverage;
}

export function MitreCoverageStats({ coverage }: MitreCoverageStatsProps) {
  const total = coverage.total_techniques;
  const active = coverage.active_techniques ?? 0;
  const passive = coverage.passive_techniques ?? (coverage.covered_techniques - active);
  const gaps = total - coverage.covered_techniques;

  const activePct = total > 0 ? (active / total) * 100 : 0;
  const passivePct = total > 0 ? (passive / total) * 100 : 0;

  const topTactic = coverage.tactics.reduce<{ name: string; covered: number } | null>(
    (best, t) => (!best || t.covered_count > best.covered ? { name: t.name, covered: t.covered_count } : best),
    null,
  );

  return (
    <div className="rounded-xl border bg-card p-5">
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm font-semibold">Overall Coverage</span>
        <span className="text-lg font-bold tabular-nums">
          {coverage.coverage_percent.toFixed(0)}%
          <span className="ml-1.5 text-xs font-normal text-muted-foreground">
            ({coverage.covered_techniques} of {total} techniques)
          </span>
        </span>
      </div>

      {/* Three-segment progress bar */}
      <div className="mb-4 h-3 w-full overflow-hidden rounded-full bg-red-100 dark:bg-red-950/30">
        <div className="flex h-full">
          <div
            className="h-full bg-green-500 transition-all"
            style={{ width: `${activePct}%` }}
            title={`Active: ${active}`}
          />
          <div
            className="h-full bg-yellow-400 transition-all"
            style={{ width: `${passivePct}%` }}
            title={`Passive: ${passive}`}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div className="rounded-lg border bg-green-50 p-3 dark:bg-green-950/20">
          <p className="text-lg font-bold tabular-nums text-green-700 dark:text-green-400">{active}</p>
          <p className="text-xs text-muted-foreground">Active Detections</p>
        </div>
        <div className="rounded-lg border bg-yellow-50 p-3 dark:bg-yellow-950/20">
          <p className="text-lg font-bold tabular-nums text-yellow-700 dark:text-yellow-400">{passive}</p>
          <p className="text-xs text-muted-foreground">Passive Rules</p>
        </div>
        <div className="rounded-lg border bg-red-50 p-3 dark:bg-red-950/20">
          <p className="text-lg font-bold tabular-nums text-red-700 dark:text-red-400">{gaps}</p>
          <p className="text-xs text-muted-foreground">Coverage Gaps</p>
        </div>
        <div className="rounded-lg border p-3">
          <p className="text-lg font-bold tabular-nums">
            {coverage.total_alerts_90d !== undefined
              ? coverage.total_alerts_90d.toLocaleString()
              : '—'}
          </p>
          <p className="text-xs text-muted-foreground">
            Alerts (90d)
            {topTactic && (
              <span className="ml-1 block truncate" title={`Top: ${topTactic.name}`}>
                Top: {topTactic.name}
              </span>
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
