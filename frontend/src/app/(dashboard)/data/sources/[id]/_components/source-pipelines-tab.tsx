'use client';

import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { type Pipeline } from '@/lib/data-suite';
import { formatMaybeCompact, formatMaybeDateTime, humanizeCronOrFrequency } from '@/lib/data-suite/utils';

interface SourcePipelinesTabProps {
  pipelines: Pipeline[];
}

export function SourcePipelinesTab({
  pipelines,
}: SourcePipelinesTabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Pipelines Using This Source</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {pipelines.length === 0 ? (
          <p className="text-sm text-muted-foreground">No pipelines currently reference this source.</p>
        ) : (
          pipelines.map((pipeline) => (
            <div key={pipeline.id} className="rounded-lg border px-4 py-3">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <Link href={`/data/pipelines/${pipeline.id}`} className="font-medium hover:text-primary">
                    {pipeline.name}
                  </Link>
                  <div className="mt-1 flex flex-wrap gap-2 text-xs text-muted-foreground">
                    <span className="capitalize">{pipeline.type}</span>
                    <span>{humanizeCronOrFrequency(pipeline.schedule)}</span>
                  </div>
                </div>
                <Badge variant="outline" className="capitalize">
                  {pipeline.status}
                </Badge>
              </div>
              <div className="mt-2 flex flex-wrap gap-4 text-sm text-muted-foreground">
                <span>{pipeline.total_runs} total runs</span>
                <span>{formatMaybeCompact(pipeline.total_records_processed)} records processed</span>
                <span>Last run {formatMaybeDateTime(pipeline.last_run_at)}</span>
              </div>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  );
}
