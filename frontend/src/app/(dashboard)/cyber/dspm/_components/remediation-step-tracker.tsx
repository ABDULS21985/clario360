'use client';

import {
  Circle,
  CheckCircle2,
  XCircle,
  MinusCircle,
  Loader2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { DSPMRemediationStep } from '@/types/cyber';

interface RemediationStepTrackerProps {
  steps: DSPMRemediationStep[];
  currentStep: number;
}

const STATUS_CONFIG: Record<
  DSPMRemediationStep['status'],
  { icon: typeof Circle; iconClass: string; borderClass: string }
> = {
  pending: { icon: Circle, iconClass: 'text-gray-400', borderClass: 'border-l-gray-200' },
  running: { icon: Loader2, iconClass: 'text-blue-500 animate-spin', borderClass: 'border-l-blue-400' },
  completed: { icon: CheckCircle2, iconClass: 'text-green-500', borderClass: 'border-l-green-400' },
  failed: { icon: XCircle, iconClass: 'text-red-500', borderClass: 'border-l-red-400' },
  skipped: { icon: MinusCircle, iconClass: 'text-gray-400', borderClass: 'border-l-gray-200' },
};

function formatTimestamp(ts?: string): string {
  if (!ts) return '';
  const d = new Date(ts);
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

export function RemediationStepTracker({ steps, currentStep }: RemediationStepTrackerProps) {
  const sorted = [...steps].sort((a, b) => a.order - b.order);

  return (
    <div className="space-y-0">
      {sorted.map((step, idx) => {
        const config = STATUS_CONFIG[step.status] ?? STATUS_CONFIG.pending;
        const Icon = config.icon;
        const isCurrent = step.order === currentStep;
        const isLast = idx === sorted.length - 1;

        return (
          <div key={step.step_id} className="relative flex gap-4">
            {/* Timeline connector */}
            <div className="flex flex-col items-center">
              <div
                className={cn(
                  'flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 bg-background',
                  isCurrent ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/30' : 'border-muted',
                )}
              >
                <Icon className={cn('h-4 w-4', config.iconClass)} />
              </div>
              {!isLast && (
                <div className={cn('w-0.5 flex-1 min-h-[2rem]', step.status === 'completed' ? 'bg-green-300' : step.status === 'failed' ? 'bg-red-300' : 'bg-muted')} />
              )}
            </div>

            {/* Step content */}
            <div
              className={cn(
                'flex-1 rounded-lg border pb-4 px-4 pt-3 mb-2',
                isCurrent && 'border-blue-400 bg-blue-50/50 dark:bg-blue-950/10',
                step.status === 'completed' && 'border-l-4 border-l-green-400',
                step.status === 'failed' && 'border-l-4 border-l-red-400',
              )}
            >
              <div className="flex items-center gap-2">
                <span className="text-xs font-medium text-muted-foreground">Step {step.order}</span>
                <span className={cn(
                  'inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize',
                  step.status === 'completed' && 'bg-green-100 text-green-700',
                  step.status === 'failed' && 'bg-red-100 text-red-700',
                  step.status === 'running' && 'bg-blue-100 text-blue-700',
                  step.status === 'pending' && 'bg-gray-100 text-gray-600',
                  step.status === 'skipped' && 'bg-gray-100 text-gray-600',
                )}>
                  {step.status}
                </span>
              </div>

              <p className="mt-1 text-sm font-semibold">{step.action.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}</p>
              <p className="mt-0.5 text-xs text-muted-foreground">{step.description}</p>

              {/* Timestamps */}
              {(step.started_at || step.completed_at) && (
                <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
                  {step.started_at && (
                    <span>Started: {formatTimestamp(step.started_at)}</span>
                  )}
                  {step.completed_at && (
                    <span>Completed: {formatTimestamp(step.completed_at)}</span>
                  )}
                </div>
              )}

              {/* Error message */}
              {step.status === 'failed' && step.error && (
                <p className="mt-2 rounded bg-red-50 p-2 text-xs text-red-700 dark:bg-red-950/20 dark:text-red-400">
                  {step.error}
                </p>
              )}

              {/* Result summary */}
              {step.status === 'completed' && step.result && Object.keys(step.result).length > 0 && (
                <div className="mt-2 rounded bg-muted/50 p-2">
                  <p className="mb-1 text-xs font-medium text-muted-foreground">Result</p>
                  <div className="space-y-0.5">
                    {Object.entries(step.result).map(([key, value]) => (
                      <div key={key} className="flex items-start gap-2 text-xs">
                        <span className="font-medium text-muted-foreground">{key.replace(/_/g, ' ')}:</span>
                        <span>{String(value)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
