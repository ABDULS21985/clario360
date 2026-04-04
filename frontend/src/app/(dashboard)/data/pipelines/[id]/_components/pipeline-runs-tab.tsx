'use client';

import { type PipelineRun } from '@/lib/data-suite';
import { formatMaybeCompact, formatMaybeDateTime, formatMaybeDurationMs } from '@/lib/data-suite/utils';
import { Button } from '@/components/ui/button';

interface PipelineRunsTabProps {
  runs: PipelineRun[];
  onSelectRun: (run: PipelineRun) => void;
}

export function PipelineRunsTab({
  runs,
  onSelectRun,
}: PipelineRunsTabProps) {
  if (runs.length === 0) {
    return <p className="text-sm text-muted-foreground">No runs have been recorded for this pipeline yet.</p>;
  }

  return (
    <div className="rounded-lg border">
      <table className="min-w-full text-sm">
        <thead>
          <tr className="border-b text-left">
            <th className="px-3 py-2 font-medium">Status</th>
            <th className="px-3 py-2 font-medium">Phase</th>
            <th className="px-3 py-2 font-medium">Loaded</th>
            <th className="px-3 py-2 font-medium">Duration</th>
            <th className="px-3 py-2 font-medium">Completed</th>
            <th className="px-3 py-2 font-medium"></th>
          </tr>
        </thead>
        <tbody>
          {runs.map((run) => (
            <tr key={run.id} className="border-b">
              <td className="px-3 py-2 capitalize">{run.status}</td>
              <td className="px-3 py-2 capitalize text-muted-foreground">{run.current_phase ?? '—'}</td>
              <td className="px-3 py-2 text-muted-foreground">{formatMaybeCompact(run.records_loaded)}</td>
              <td className="px-3 py-2 text-muted-foreground">{formatMaybeDurationMs(run.duration_ms)}</td>
              <td className="px-3 py-2 text-muted-foreground">{formatMaybeDateTime(run.completed_at ?? run.started_at)}</td>
              <td className="px-3 py-2">
                <Button type="button" variant="outline" size="sm" onClick={() => onSelectRun(run)}>
                  Inspect
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
