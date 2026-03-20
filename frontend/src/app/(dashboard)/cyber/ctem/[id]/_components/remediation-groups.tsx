'use client';

import { useState } from 'react';
import { ChevronRight, CheckCircle2, Zap } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import type { CTEMFinding, CyberSeverity } from '@/types/cyber';

const SEVERITY_ORDER: CyberSeverity[] = ['critical', 'high', 'medium', 'low'];

const severityHeaderColors: Record<CyberSeverity, string> = {
  critical: 'text-red-700 dark:text-red-400 border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20',
  high: 'text-orange-700 dark:text-orange-400 border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-950/20',
  medium: 'text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-950/20',
  low: 'text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-950/20',
  info: 'text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900/20',
};

function FindingCard({ finding }: { finding: CTEMFinding }) {
  const [expanded, setExpanded] = useState(false);
  const hasSteps = (finding.remediation_steps?.length ?? 0) > 0;

  return (
    <div className="rounded-lg border bg-card">
      <button
        type="button"
        className="flex w-full items-start justify-between gap-3 px-4 py-3 text-left"
        onClick={() => hasSteps && setExpanded((v) => !v)}
        aria-expanded={expanded}
      >
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium leading-tight">{finding.title}</span>
            {finding.exploit_available && (
              <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-900/40 dark:text-red-400">
                <Zap className="h-3 w-3" aria-hidden />
                Exploit Available
              </span>
            )}
          </div>
          {finding.asset_name && (
            <span className="text-xs text-muted-foreground">{finding.asset_name}</span>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-3">
          <Badge variant="outline" className="text-xs tabular-nums">
            Priority {finding.priority_score}
          </Badge>
          {hasSteps && (
            <ChevronRight
              className={`h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-150 ${expanded ? 'rotate-90' : ''}`}
              aria-hidden
            />
          )}
        </div>
      </button>

      {expanded && hasSteps && (
        <div className="border-t px-4 pb-4 pt-3">
          <p className="mb-2 text-xs font-semibold text-muted-foreground">Remediation Steps</p>
          <ol className="space-y-2">
            {finding.remediation_steps!.map((step, idx) => (
              <li key={idx} className="flex items-start gap-2.5 text-sm">
                <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
                <span>{step}</span>
              </li>
            ))}
          </ol>
        </div>
      )}
    </div>
  );
}

export function RemediationGroups({ findings }: { findings: CTEMFinding[] }) {
  const withSteps = findings.filter((f) => (f.remediation_steps?.length ?? 0) > 0);

  const grouped = SEVERITY_ORDER.reduce<Record<CyberSeverity, CTEMFinding[]>>(
    (acc, sev) => {
      acc[sev] = withSteps.filter((f) => f.severity === sev);
      return acc;
    },
    { critical: [], high: [], medium: [], low: [], info: [] },
  );

  const hasAny = withSteps.length > 0;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <h3 className="text-base font-semibold">Remediation Actions</h3>
        <Badge variant="secondary" className="tabular-nums">
          {withSteps.length}
        </Badge>
      </div>

      {!hasAny && (
        <p className="py-6 text-center text-sm text-muted-foreground">
          No remediation actions available
        </p>
      )}

      {SEVERITY_ORDER.map((sev) => {
        const group = grouped[sev];
        if (group.length === 0) return null;
        return (
          <div key={sev} className="space-y-2">
            <div className={`flex items-center gap-2 rounded-lg border px-3 py-2 ${severityHeaderColors[sev]}`}>
              <SeverityIndicator severity={sev} showLabel size="sm" />
              <span className="ml-auto text-xs font-medium tabular-nums">{group.length} finding{group.length !== 1 ? 's' : ''}</span>
            </div>
            <div className="space-y-2 pl-1">
              {group.map((finding) => (
                <FindingCard key={finding.id} finding={finding} />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}
