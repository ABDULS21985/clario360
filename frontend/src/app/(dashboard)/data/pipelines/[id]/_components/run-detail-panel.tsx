'use client';

import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { type PipelineRun, type PipelineRunLog } from '@/lib/data-suite';
import { formatMaybeBytes, formatMaybeCompact, formatMaybeDateTime, formatMaybeDurationMs } from '@/lib/data-suite/utils';
import { QualityGateResults } from '@/app/(dashboard)/data/pipelines/[id]/_components/quality-gate-results';
import { RunLogViewer } from '@/app/(dashboard)/data/pipelines/[id]/_components/run-log-viewer';

interface RunDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  run: PipelineRun | null;
  logs: PipelineRunLog[];
}

export function RunDetailPanel({
  open,
  onOpenChange,
  run,
  logs,
}: RunDetailPanelProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle>Run Detail</SheetTitle>
          <SheetDescription>
            {run ? `Run ${run.id} • ${run.status}` : 'Select a run to inspect metrics, phases, and logs.'}
          </SheetDescription>
        </SheetHeader>

        {run ? (
          <div className="mt-6 space-y-6">
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <Metric label="Status" value={run.status} />
              <Metric label="Current Phase" value={run.current_phase ?? '—'} />
              <Metric label="Started" value={formatMaybeDateTime(run.started_at)} />
              <Metric label="Completed" value={formatMaybeDateTime(run.completed_at)} />
              <Metric label="Duration" value={formatMaybeDurationMs(run.duration_ms)} />
              <Metric label="Bytes Written" value={formatMaybeBytes(run.bytes_written)} />
              <Metric label="Extracted" value={formatMaybeCompact(run.records_extracted)} />
              <Metric label="Loaded" value={formatMaybeCompact(run.records_loaded)} />
            </div>

            <div className="space-y-3">
              <h4 className="font-medium">Quality Gates</h4>
              <QualityGateResults results={run.quality_gate_results} />
            </div>

            <div className="space-y-3">
              <h4 className="font-medium">Execution Log</h4>
              <RunLogViewer logs={logs} />
            </div>
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
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
    <div className="rounded-lg border bg-muted/20 p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium capitalize">{value}</div>
    </div>
  );
}
