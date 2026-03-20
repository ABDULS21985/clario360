'use client';

import { Building2, CheckCircle2, ImagePlus, LayoutGrid, Sparkles, Users } from 'lucide-react';

import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';

export function StepIndicator({ currentStep }: { currentStep: number }) {
  const steps = [
    { number: 1, label: 'Organization', icon: Building2 },
    { number: 2, label: 'Branding', icon: ImagePlus },
    { number: 3, label: 'Team', icon: Users },
    { number: 4, label: 'Suites', icon: LayoutGrid },
    { number: 5, label: 'Ready', icon: Sparkles },
  ] as const;

  return (
    <div className="mb-8">
      <div className="mb-4 flex items-center justify-between gap-2">
        {steps.map((step) => {
          const Icon = step.icon;
          const isActive = currentStep === step.number;
          const isComplete = currentStep > step.number;
          return (
            <div key={step.number} className="flex flex-1 flex-col items-center gap-2">
              <div
                className={cn(
                  'flex h-11 w-11 items-center justify-center rounded-full border text-sm transition-all',
                  isComplete && 'border-[#0f5132] bg-[#0f5132] text-white',
                  isActive && 'border-[#0f5132] bg-white text-[#0f5132] shadow-sm',
                  !isComplete && !isActive && 'border-slate-200 bg-white text-slate-400',
                )}
              >
                {isComplete ? <CheckCircle2 className="h-4 w-4" /> : <Icon className="h-4 w-4" />}
              </div>
              <span className={cn('text-[11px] uppercase tracking-[0.2em]', isActive ? 'text-[#0f5132]' : 'text-slate-400')}>
                {step.label}
              </span>
            </div>
          );
        })}
      </div>
      <Progress value={((currentStep - 1) / 4) * 100} className="h-2 bg-slate-100" />
    </div>
  );
}
