'use client';

import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { type LineageGraph } from '@/lib/data-suite';

interface SourceLineageTabProps {
  sourceId: string;
  graph: LineageGraph | null;
}

export function SourceLineageTab({
  sourceId,
  graph,
}: SourceLineageTabProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Lineage Around This Source</CardTitle>
        <Button variant="outline" size="sm" asChild>
          <Link href={`/data/lineage?type=data_source&id=${sourceId}`}>
            Open full lineage
            <ArrowRight className="ml-1.5 h-3.5 w-3.5" />
          </Link>
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {!graph || graph.nodes.length === 0 ? (
          <p className="text-sm text-muted-foreground">No lineage graph is available for this source.</p>
        ) : (
          <>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
              <SummaryCard label="Nodes" value={String(graph.stats.node_count)} />
              <SummaryCard label="Edges" value={String(graph.stats.edge_count)} />
              <SummaryCard label="Depth" value={String(graph.stats.max_depth)} />
            </div>
            <div className="rounded-lg border">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b text-left">
                    <th className="px-3 py-2 font-medium">Node</th>
                    <th className="px-3 py-2 font-medium">Type</th>
                    <th className="px-3 py-2 font-medium">Status</th>
                    <th className="px-3 py-2 font-medium">Links</th>
                  </tr>
                </thead>
                <tbody>
                  {graph.nodes.map((node) => (
                    <tr key={node.id} className="border-b">
                      <td className="px-3 py-2 font-medium">{node.name}</td>
                      <td className="px-3 py-2 capitalize text-muted-foreground">{node.type.replace(/_/g, ' ')}</td>
                      <td className="px-3 py-2 text-muted-foreground">{node.status ?? '—'}</td>
                      <td className="px-3 py-2 text-muted-foreground">
                        {node.in_degree} in / {node.out_degree} out
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function SummaryCard({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border bg-muted/20 p-4">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-xl font-semibold">{value}</div>
    </div>
  );
}
