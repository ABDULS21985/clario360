'use client';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { type Pipeline } from '@/lib/data-suite';

interface PipelineConfigTabProps {
  pipeline: Pipeline;
}

export function PipelineConfigTab({
  pipeline,
}: PipelineConfigTabProps) {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Property label="Source Table" value={pipeline.config.source_table ?? '—'} />
          <Property label="Source Query" value={pipeline.config.source_query ?? '—'} />
          <Property label="Target Table" value={pipeline.config.target_table ?? '—'} />
          <Property label="Load Strategy" value={pipeline.config.load_strategy ?? '—'} />
          <Property label="Batch Size" value={pipeline.config.batch_size?.toLocaleString() ?? '—'} />
          <Property label="Incremental Field" value={pipeline.config.incremental_field ?? '—'} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Transformation Flow</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(pipeline.config.transformations ?? []).length === 0 ? (
            <p className="text-sm text-muted-foreground">No transformations are configured.</p>
          ) : (
            (pipeline.config.transformations ?? []).map((transform, index) => (
              <div key={`${transform.type}-${index}`} className="rounded-lg border px-4 py-3">
                <div className="flex items-center gap-2">
                  <Badge variant="outline">{index + 1}</Badge>
                  <span className="font-medium capitalize">{transform.type.replace(/_/g, ' ')}</span>
                </div>
                <pre className="mt-2 overflow-x-auto rounded bg-muted/20 p-3 text-xs">
                  {JSON.stringify(transform.config, null, 2)}
                </pre>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function Property({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div>
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm">{value}</div>
    </div>
  );
}
