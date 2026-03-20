'use client';

import { type ImpactAnalysis } from '@/lib/data-suite';

interface LineageImpactPanelProps {
  impact: ImpactAnalysis | null;
}

export function LineageImpactPanel({
  impact,
}: LineageImpactPanelProps) {
  if (!impact) {
    return (
      <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
        Enable impact analysis and select a node to see downstream blast radius.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="font-medium">Impact Analysis</div>
      <div className="mt-2 text-sm text-muted-foreground">{impact.summary}</div>
      <div className="mt-4 grid gap-3">
        <Metric label="Severity" value={impact.severity} />
        <Metric label="Directly Affected" value={impact.directly_affected.length.toString()} />
        <Metric label="Indirectly Affected" value={impact.indirectly_affected.length.toString()} />
        <Metric label="Affected Suites" value={impact.affected_suites.length.toString()} />
      </div>
    </div>
  );
}

function Metric({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-md border bg-muted/20 p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium capitalize">{value}</div>
    </div>
  );
}
