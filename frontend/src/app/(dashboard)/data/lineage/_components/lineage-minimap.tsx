'use client';

import { type LineageGraph } from '@/lib/data-suite';

interface LineageMinimapProps {
  graph: LineageGraph;
}

export function LineageMinimap({
  graph,
}: LineageMinimapProps) {
  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">Overview</div>
      <div className="mt-2 text-sm text-muted-foreground">
        {graph.stats.node_count} nodes • {graph.stats.edge_count} edges
      </div>
      <div className="mt-3 grid grid-cols-2 gap-2 text-xs text-muted-foreground">
        <div>Sources: {graph.stats.source_count}</div>
        <div>Consumers: {graph.stats.consumer_count}</div>
      </div>
    </div>
  );
}
