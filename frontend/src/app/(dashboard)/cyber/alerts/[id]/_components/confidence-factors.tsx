'use client';

import { cn } from '@/lib/utils';
import type { ConfidenceFactor } from '@/types/cyber';

interface ConfidenceFactorsProps {
  factors: ConfidenceFactor[];
}

export function ConfidenceFactors({ factors }: ConfidenceFactorsProps) {
  if (!factors || factors.length === 0) return null;

  return (
    <div className="space-y-2">
      {factors.map((factor, i) => {
        const isPositive = factor.impact > 0;
        return (
          <div key={i} className="flex items-center gap-3 rounded-md border p-2.5">
            <div className={cn(
              'flex h-7 w-10 shrink-0 items-center justify-center rounded-md text-xs font-bold tabular-nums',
              isPositive
                ? 'bg-red-100 text-red-700 dark:bg-red-950/30 dark:text-red-400'
                : 'bg-green-100 text-green-700 dark:bg-green-950/30 dark:text-green-400',
            )}>
              {isPositive ? '+' : ''}{factor.impact}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-xs font-medium">{factor.factor.replace(/_/g, ' ')}</p>
              <p className="text-xs text-muted-foreground">{factor.description}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
