'use client';

import { AlertTriangle } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { timeAgo } from '@/lib/utils';
import type { MITRETechniqueCoverage } from '@/types/cyber';

export type CellState = 'active' | 'passive' | 'gap' | 'na';

function getCellState(t: MITRETechniqueCoverage): CellState {
  if (t.rule_count > 0 && t.alert_count > 0) return 'active';
  if (t.rule_count > 0 && t.alert_count === 0) return 'passive';
  if (t.rule_count === 0) return 'gap';
  return 'na';
}

const STATE_CLASSES: Record<CellState, string> = {
  active: 'bg-green-100 border-green-400 text-green-800 dark:bg-green-950/30 dark:border-green-700 dark:text-green-300',
  passive: 'bg-yellow-50 border-yellow-300 text-yellow-800 dark:bg-yellow-950/30 dark:border-yellow-600 dark:text-yellow-300',
  gap: 'bg-red-50 border-red-200 text-red-700 dark:bg-red-950/30 dark:border-red-800 dark:text-red-400',
  na: 'bg-gray-50 border-gray-200 text-gray-400 dark:bg-gray-900/30 dark:border-gray-700 dark:text-gray-600',
};

const DOT_COLORS: Record<CellState, string> = {
  active: 'text-green-600',
  passive: 'text-yellow-600',
  gap: 'text-transparent',
  na: 'text-transparent',
};

interface MitreCellProps {
  technique: MITRETechniqueCoverage;
  selected: boolean;
  highlighted: boolean;
  onSelect: (technique: MITRETechniqueCoverage) => void;
}

export function MitreCell({ technique, selected, highlighted, onSelect }: MitreCellProps) {
  const state = getCellState(technique);
  const baseClass = STATE_CLASSES[state];
  const dotColor = DOT_COLORS[state];
  const maxDots = 3;
  const dotCount = Math.min(technique.rule_count, maxDots);
  const hasOverflow = technique.rule_count > maxDots;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={() => onSelect(technique)}
            aria-label={`${technique.technique_id} ${technique.technique_name}`}
            className={[
              'w-full rounded border p-1.5 text-left transition-all focus:outline-none focus:ring-1 focus:ring-primary',
              baseClass,
              selected ? 'ring-1 ring-primary shadow-sm' : 'hover:shadow-sm',
              highlighted ? 'ring-2 ring-blue-500' : '',
              state === 'na' ? 'cursor-default opacity-60' : 'cursor-pointer',
            ].join(' ')}
          >
            <span
              className={`block font-mono text-[10px] font-bold leading-tight ${state === 'na' ? 'line-through' : ''}`}
            >
              {technique.technique_id}
            </span>
            {state === 'gap' ? (
              <AlertTriangle className="mt-0.5 h-2.5 w-2.5 text-red-400" aria-hidden />
            ) : state !== 'na' && technique.rule_count > 0 ? (
              <span className={`text-[10px] ${dotColor}`} aria-label={`${technique.rule_count} rules`}>
                {hasOverflow ? `●${dotCount}+` : '●'.repeat(dotCount)}
              </span>
            ) : null}
          </button>
        </TooltipTrigger>
        <TooltipContent side="top" className="max-w-xs">
          <p className="font-semibold">{technique.technique_name}</p>
          <p className="text-xs text-muted-foreground">
            {technique.rule_count} detection rule{technique.rule_count !== 1 ? 's' : ''}
          </p>
          <p className="text-xs text-muted-foreground">
            {technique.alert_count} alert{technique.alert_count !== 1 ? 's' : ''} in last 90 days
          </p>
          <p className="text-xs text-muted-foreground">
            {technique.last_alert ? `Last alert: ${timeAgo(technique.last_alert)}` : 'No alerts detected'}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export { getCellState };
