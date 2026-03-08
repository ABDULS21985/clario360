'use client';

import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';

import { SUITES } from './shared';

type SuiteDefinition = (typeof SUITES)[number];

export function SuiteSelectorCard({
  suite,
  active,
  onToggle,
}: {
  suite: SuiteDefinition;
  active: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={cn(
        'overflow-hidden rounded-3xl border text-left transition-all',
        active ? 'border-[#0f5132] shadow-md' : 'border-slate-200 hover:border-slate-300',
      )}
    >
      <div className={cn('h-2 w-full bg-gradient-to-r', suite.accent)} />
      <div className="space-y-4 bg-white p-5">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-base font-semibold text-slate-900">{suite.title}</p>
            <p className="mt-1 text-sm text-slate-500">{suite.description}</p>
          </div>
          <Checkbox checked={active} />
        </div>
      </div>
    </button>
  );
}
