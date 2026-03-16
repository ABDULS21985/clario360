'use client';

import { Badge } from '@/components/ui/badge';
import { type QualityGateResult } from '@/lib/data-suite';

interface QualityGateResultsProps {
  results: QualityGateResult[];
}

export function QualityGateResults({
  results,
}: QualityGateResultsProps) {
  if (!results || results.length === 0) {
    return <p className="text-sm text-muted-foreground">No quality gates were evaluated for this run.</p>;
  }

  return (
    <div className="space-y-3">
      {results.map((result) => (
        <div key={`${result.name}-${result.evaluated_at}`} className="rounded-lg border px-4 py-3">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="font-medium">{result.name}</div>
              <div className="mt-1 text-xs text-muted-foreground">
                {result.metric} • value {result.metric_value}
              </div>
            </div>
            <Badge variant="outline">{result.status}</Badge>
          </div>
          {result.message ? <div className="mt-2 text-sm text-muted-foreground">{result.message}</div> : null}
        </div>
      ))}
    </div>
  );
}
