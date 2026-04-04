'use client';

import { Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import {
  CTI_TAKEDOWN_STATUS_LABELS,
  type CTITakedownStatus,
} from '@/types/cti';
import { CTI_TAKEDOWN_WORKFLOW } from '@/lib/cti-utils';

interface TakedownPipelineProps {
  currentStatus?: CTITakedownStatus;
  status?: CTITakedownStatus;
  requestedAt?: string | null;
  takenDownAt?: string | null;
  onAdvanceStatus?: (newStatus: CTITakedownStatus) => void;
  isLoading?: boolean;
}

export function TakedownPipeline({
  currentStatus,
  status,
  requestedAt,
  takenDownAt,
  onAdvanceStatus,
  isLoading = false,
}: TakedownPipelineProps) {
  const resolvedStatus = currentStatus ?? status ?? 'detected';
  const activeIndex = CTI_TAKEDOWN_WORKFLOW.indexOf(resolvedStatus);

  return (
    <div className="space-y-3 rounded-[24px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
      <div>
        <h3 className="text-sm font-semibold text-slate-950">Takedown Workflow</h3>
        <p className="text-sm text-muted-foreground">
          Advance the takedown lifecycle as evidence is confirmed and external remediation progresses.
        </p>
      </div>
      <div className="grid gap-3 md:grid-cols-4">
        {CTI_TAKEDOWN_WORKFLOW.map((step, index) => {
          const isDone = index < activeIndex;
          const isCurrent = index === activeIndex;
          const isNext = index === activeIndex + 1;
          const isClickable = Boolean(onAdvanceStatus) && isNext && !isLoading;

          return (
            <button
              key={step}
              type="button"
              disabled={!isClickable}
              onClick={() => isClickable && onAdvanceStatus?.(step)}
              className={cn(
                'relative rounded-2xl border px-3 py-4 text-left transition',
                isCurrent && 'border-emerald-500/40 bg-emerald-500/10',
                isDone && 'border-emerald-500/30 bg-emerald-500/5',
                isClickable && 'hover:border-amber-500/40 hover:bg-amber-500/10',
                !isCurrent && !isDone && !isClickable && 'bg-background',
                !isClickable && 'cursor-default',
              )}
            >
              {index < CTI_TAKEDOWN_WORKFLOW.length - 1 && (
                <div className="absolute right-[-0.75rem] top-1/2 hidden h-px w-6 -translate-y-1/2 bg-border md:block" />
              )}
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    'inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold',
                    isDone || isCurrent ? 'bg-emerald-500 text-white' : 'bg-muted text-muted-foreground',
                  )}
                >
                  {isDone ? <Check className="h-3.5 w-3.5" /> : index + 1}
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
            </button>
          );
        })}
      </div>
      {onAdvanceStatus && activeIndex < CTI_TAKEDOWN_WORKFLOW.length - 1 && (
        <div className="flex justify-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={isLoading}
            onClick={() => onAdvanceStatus(CTI_TAKEDOWN_WORKFLOW[activeIndex + 1])}
          >
            Advance to {CTI_TAKEDOWN_STATUS_LABELS[CTI_TAKEDOWN_WORKFLOW[activeIndex + 1]]}
          </Button>
        </div>
      )}
    </div>
  );
}
