'use client';

import { AlertTriangle } from 'lucide-react';

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { timeAgo } from '@/lib/utils';
import type { MITRETechniqueCoverage } from '@/types/cyber';

export type CellState = 'covered' | 'noisy' | 'gap' | 'idle';

const STATE_CLASSES: Record<CellState, string> = {
  covered: 'border-emerald-300 bg-emerald-50 text-emerald-900',
  noisy: 'border-amber-300 bg-amber-50 text-amber-900',
  gap: 'border-red-300 bg-red-50 text-red-900',
  idle: 'border-slate-200 bg-slate-50 text-slate-500',
};

export function MitreCell({
  technique,
  selected,
  highlighted,
  onSelect,
}: {
  technique: MITRETechniqueCoverage;
  selected: boolean;
  highlighted: boolean;
  onSelect: (technique: MITRETechniqueCoverage) => void;
}) {
  const state = technique.coverage_state;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            className={[
              'w-full rounded-2xl border p-2 text-left transition hover:shadow-sm',
              STATE_CLASSES[state],
              selected ? 'ring-2 ring-emerald-600' : '',
              highlighted ? 'ring-2 ring-sky-500' : '',
            ].join(' ')}
            onClick={() => onSelect(technique)}
          >
            <div className="flex items-start justify-between gap-2">
              <span className="font-mono text-[11px] font-semibold">{technique.technique_id}</span>
              {state === 'gap' ? <AlertTriangle className="h-3.5 w-3.5 text-red-600" /> : null}
            </div>
            <p className="mt-1 line-clamp-2 text-[11px] leading-4">{technique.technique_name}</p>
          </button>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          <p className="font-medium">{technique.technique_name}</p>
          <p className="text-xs text-muted-foreground">
            {technique.rule_count} rule{technique.rule_count === 1 ? '' : 's'} · {technique.alert_count} alert{technique.alert_count === 1 ? '' : 's'}
          </p>
          <p className="text-xs text-muted-foreground">
            {technique.active_threat_count} active threat{technique.active_threat_count === 1 ? '' : 's'}
          </p>
          <p className="text-xs text-muted-foreground">
            {technique.last_alert_at ? `Last alert ${timeAgo(technique.last_alert_at)}` : 'No recent alerts'}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
