'use client';

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { AlertTriangle } from 'lucide-react';
import type { DetectionRule } from '@/types/cyber';

interface RulePerformanceCardProps {
  rule: DetectionRule;
}

export function RulePerformanceCard({ rule }: RulePerformanceCardProps) {
  const fpPct = rule.false_positive_rate * 100;
  const fpColor =
    fpPct > 40 ? 'text-red-600 dark:text-red-400' : fpPct > 20 ? 'text-yellow-600 dark:text-yellow-400' : 'text-green-600 dark:text-green-400';

  const tp = rule.tp_count;
  const fp = rule.fp_count;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-1.5 text-xs">
            <span className="tabular-nums text-muted-foreground">{rule.trigger_count.toLocaleString()} triggers</span>
            <span className="text-muted-foreground">·</span>
            <span className={`tabular-nums font-medium ${fpColor}`}>{fpPct.toFixed(1)}% FP</span>
            {fpPct > 50 && (
              <span className="flex items-center gap-0.5 rounded-full bg-red-100 px-1.5 py-0.5 text-[10px] font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400">
                <AlertTriangle className="h-2.5 w-2.5" />
                High FP
              </span>
            )}
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <p className="text-xs">
            True Positives: {tp !== undefined ? tp : '—'}
            {' · '}
            False Positives: {fp !== undefined ? fp : '—'}
          </p>
          {fpPct > 50 && (
            <p className="text-xs text-red-400">Auto-disable risk at current FP rate</p>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
