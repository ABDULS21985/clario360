'use client';

import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { type LineageGraph } from '@/lib/data-suite';

interface PipelineLineageTabProps {
  pipelineId: string;
  graph: LineageGraph | null;
}

export function PipelineLineageTab({
  pipelineId,
  graph,
}: PipelineLineageTabProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Lineage Position</CardTitle>
        <Button variant="outline" size="sm" asChild>
          <Link href={`/data/lineage?type=pipeline&id=${pipelineId}`}>Open full lineage</Link>
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {!graph || graph.nodes.length === 0 ? (
          <p className="text-sm text-muted-foreground">No lineage information is available for this pipeline.</p>
        ) : (
          graph.nodes.map((node) => (
            <div key={node.id} className="rounded-lg border px-4 py-3">
              <div className="font-medium">{node.name}</div>
              <div className="mt-1 text-xs capitalize text-muted-foreground">
                {node.type.replace(/_/g, ' ')} • {node.in_degree} in / {node.out_degree} out
              </div>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  );
}
