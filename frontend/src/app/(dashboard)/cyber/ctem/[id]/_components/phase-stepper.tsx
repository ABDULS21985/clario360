'use client';

import { cn } from '@/lib/utils';
import { CheckCircle, Circle, Loader2, XCircle } from 'lucide-react';
import type { CTEMPhaseInfo, CTEMPhase } from '@/types/cyber';

const PHASES: { key: CTEMPhase; label: string; description: string }[] = [
  { key: 'scoping', label: 'Scoping', description: 'Define assessment boundaries' },
  { key: 'discovery', label: 'Discovery', description: 'Enumerate assets and vulnerabilities' },
  { key: 'prioritization', label: 'Prioritization', description: 'Rank findings by business risk' },
  { key: 'validation', label: 'Validation', description: 'Verify exploitability and impact' },
  { key: 'mobilization', label: 'Mobilization', description: 'Generate remediation guidance' },
];

interface PhaseStepperProps {
  phases: CTEMPhaseInfo[];
  currentPhase?: CTEMPhase;
}

export function PhaseStepper({ phases, currentPhase }: PhaseStepperProps) {
  const phaseMap = Object.fromEntries(phases.map((p) => [p.phase, p]));

  return (
    <div className="overflow-x-auto">
      <div className="flex min-w-max items-start gap-0">
        {PHASES.map((def, idx) => {
          const info = phaseMap[def.key];
          const status = info?.status ?? 'pending';
          const isCurrent = def.key === currentPhase;
          const isLast = idx === PHASES.length - 1;

          return (
            <div key={def.key} className="flex items-center">
              {/* Step */}
              <div className="flex flex-col items-center">
                <div className={cn(
                  'flex h-9 w-9 items-center justify-center rounded-full border-2 transition-all',
                  status === 'completed' && 'border-green-500 bg-green-50 dark:bg-green-950/30',
                  status === 'running' && 'border-blue-500 bg-blue-50 dark:bg-blue-950/30',
                  status === 'failed' && 'border-red-500 bg-red-50 dark:bg-red-950/30',
                  status === 'pending' && 'border-muted-foreground/30 bg-muted/50',
                )}>
                  {status === 'completed' && <CheckCircle className="h-5 w-5 text-green-600" />}
                  {status === 'running' && <Loader2 className="h-5 w-5 text-blue-600 animate-spin" />}
                  {status === 'failed' && <XCircle className="h-5 w-5 text-red-600" />}
                  {status === 'pending' && <Circle className="h-5 w-5 text-muted-foreground/40" />}
                </div>
                <div className="mt-2 text-center">
                  <p className={cn(
                    'text-xs font-semibold',
                    isCurrent && 'text-blue-600',
                    status === 'completed' && 'text-green-600',
                    status === 'failed' && 'text-red-600',
                    status === 'pending' && 'text-muted-foreground',
                  )}>
                    {def.label}
                  </p>
                  <p className="text-xs text-muted-foreground max-w-[80px] leading-tight">{def.description}</p>
                  {info?.progress_percent != null && status === 'running' && (
                    <div className="mt-1 h-1 w-16 rounded-full bg-muted overflow-hidden">
                      <div
                        className="h-full rounded-full bg-blue-500 transition-all"
                        style={{ width: `${info.progress_percent}%` }}
                      />
                    </div>
                  )}
                </div>
              </div>

              {/* Connector */}
              {!isLast && (
                <div className={cn(
                  'mx-2 h-0.5 w-12 flex-shrink-0 self-start mt-4',
                  status === 'completed' ? 'bg-green-400' : 'bg-muted-foreground/20',
                )} />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
