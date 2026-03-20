'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { type Pipeline } from '@/lib/data-suite';

interface PipelineQualityTabProps {
  pipeline: Pipeline;
}

export function PipelineQualityTab({
  pipeline,
}: PipelineQualityTabProps) {
  const gates = pipeline.config.quality_gates ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Quality Gates</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {gates.length === 0 ? (
          <p className="text-sm text-muted-foreground">No quality gates are configured for this pipeline.</p>
        ) : (
          gates.map((gate) => (
            <div key={gate.name} className="rounded-lg border px-4 py-3">
              <div className="font-medium">{gate.name}</div>
              <div className="mt-1 text-xs text-muted-foreground">
                {gate.metric} {gate.operator ? `• ${gate.operator}` : ''}{' '}
                {gate.threshold !== undefined && gate.threshold !== null ? `• ${gate.threshold}` : ''}
              </div>
              {gate.description ? <div className="mt-2 text-sm text-muted-foreground">{gate.description}</div> : null}
            </div>
          ))
        )}
      </CardContent>
    </Card>
  );
}
