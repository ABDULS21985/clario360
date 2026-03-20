'use client';

import { type LineageNode } from '@/lib/data-suite';

interface LineageDetailPanelProps {
  node: LineageNode | null;
}

export function LineageDetailPanel({
  node,
}: LineageDetailPanelProps) {
  if (!node) {
    return (
      <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
        Select a node to inspect lineage details.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="font-medium">{node.name}</div>
      <div className="mt-1 text-sm capitalize text-muted-foreground">{node.type.replace(/_/g, ' ')}</div>
      <div className="mt-4 grid gap-3">
        <Metric label="Status" value={node.status ?? '—'} />
        <Metric label="Depth" value={node.depth.toString()} />
        <Metric label="Inbound" value={node.in_degree.toString()} />
        <Metric label="Outbound" value={node.out_degree.toString()} />
        <Metric label="Critical" value={node.is_critical ? 'Yes' : 'No'} />
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
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  );
}
