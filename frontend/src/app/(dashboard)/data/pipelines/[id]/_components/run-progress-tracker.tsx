'use client';

import { Progress } from '@/components/ui/progress';
import { type PipelineRun } from '@/lib/data-suite';
import { formatMaybeCompact } from '@/lib/data-suite/utils';

interface RunProgressTrackerProps {
  run: PipelineRun | null;
}

export function RunProgressTracker({
  run,
}: RunProgressTrackerProps) {
  if (!run || run.status !== 'running') {
    return null;
  }

  const total = Math.max(run.records_extracted, run.records_transformed, run.records_loaded, 1);
  const progress = Math.min(100, Math.round((run.records_loaded / total) * 100));

  return (
    <div className="rounded-lg border bg-primary/5 p-4">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">Pipeline is running</span>
        <span className="capitalize text-muted-foreground">{run.current_phase ?? 'processing'}</span>
      </div>
      <Progress className="mt-3" value={progress} />
      <div className="mt-2 text-xs text-muted-foreground">
        {formatMaybeCompact(run.records_loaded)} loaded of {formatMaybeCompact(total)} observed records
      </div>
    </div>
  );
}
