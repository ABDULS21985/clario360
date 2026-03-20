'use client';

import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';

import type { ProvisioningStatus } from './shared';

export function ProvisioningProgress({
  status,
  fallbackStatus,
}: {
  status: ProvisioningStatus | null;
  fallbackStatus: 'pending' | 'provisioning' | 'completed' | 'failed';
}) {
  const steps = status?.steps ?? [];
  const progressPct = status?.progress_pct ?? (fallbackStatus === 'completed' ? 100 : 0);

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Progress value={progressPct} className="h-2 bg-slate-100" />
        <p className="text-right text-xs uppercase tracking-[0.2em] text-slate-500">{progressPct}% complete</p>
      </div>

      <div className="space-y-3">
        {steps.map((step) => (
          <div key={step.step_number} className="flex items-start justify-between rounded-2xl border border-slate-200 bg-white px-4 py-3">
            <div>
              <p className="font-medium text-slate-900">{step.step_name}</p>
              {step.error_message ? <p className="mt-1 text-sm text-destructive">{step.error_message}</p> : null}
            </div>
            <span
              className={cn(
                'rounded-full px-2.5 py-1 text-xs font-medium uppercase tracking-[0.15em]',
                step.status === 'completed' && 'bg-[#0f5132]/10 text-[#0f5132]',
                step.status === 'running' && 'bg-[#d97706]/10 text-[#d97706]',
                step.status === 'failed' && 'bg-red-500/10 text-red-600',
                step.status === 'pending' && 'bg-slate-100 text-slate-500',
                step.status === 'skipped' && 'bg-slate-200 text-slate-600',
              )}
            >
              {step.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
