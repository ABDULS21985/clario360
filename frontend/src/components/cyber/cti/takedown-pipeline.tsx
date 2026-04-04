'use client';

import { cn } from '@/lib/utils';
import { CTI_TAKEDOWN_STATUS_LABELS, type CTITakedownStatus } from '@/types/cti';

const STEPS: CTITakedownStatus[] = [
  'detected',
  'reported',
  'takedown_requested',
  'taken_down',
  'monitoring',
  'false_positive',
];

interface TakedownPipelineProps {
  status: CTITakedownStatus;
  requestedAt?: string | null;
  takenDownAt?: string | null;
}

export function TakedownPipeline({
  status,
  requestedAt,
  takenDownAt,
}: TakedownPipelineProps) {
  const activeIndex = STEPS.indexOf(status);

  return (
    <div className="space-y-3 rounded-[24px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
      <div>
        <h3 className="text-sm font-semibold text-slate-950">Takedown Workflow</h3>
        <p className="text-sm text-muted-foreground">
          Track where the incident sits in the takedown pipeline and whether escalation is still required.
        </p>
      </div>
      <div className="grid gap-3 md:grid-cols-6">
        {STEPS.map((step, index) => {
          const isReached = index <= activeIndex;
          const isCurrent = step === status;

          return (
            <div key={step} className="relative rounded-2xl border bg-background px-3 py-4">
              {index < STEPS.length - 1 && (
                <div className="absolute right-[-0.75rem] top-1/2 hidden h-px w-6 -translate-y-1/2 bg-border md:block" />
              )}
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    'inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold',
                    isReached ? 'bg-emerald-500 text-white' : 'bg-muted text-muted-foreground',
                  )}
                >
                  {index + 1}
                </span>
                <span className={cn('text-xs font-medium', isCurrent && 'text-emerald-700')}>
                  {CTI_TAKEDOWN_STATUS_LABELS[step]}
                </span>
              </div>
              {(step === 'takedown_requested' && requestedAt) || (step === 'taken_down' && takenDownAt) ? (
                <p className="mt-2 text-[11px] text-muted-foreground">
                  {step === 'takedown_requested' ? requestedAt : takenDownAt}
                </p>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}