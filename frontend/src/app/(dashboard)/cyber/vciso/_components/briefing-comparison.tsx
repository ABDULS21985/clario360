'use client';
import { TrendingDown, TrendingUp, Minus } from 'lucide-react';
import type { VCISOBriefing } from '@/types/cyber';

export function BriefingComparison({
  current,
  previousScore,
}: {
  current: VCISOBriefing;
  previousScore?: number;
}) {
  if (previousScore === undefined) {
    return (
      <div className="rounded-lg border bg-muted/40 p-4 text-center">
        <p className="text-sm text-muted-foreground">
          No previous briefing available for comparison.
        </p>
      </div>
    );
  }

  const currentScore = current.risk_posture.overall_score;
  const delta = currentScore - previousScore;
  const improved = delta < 0;
  const declined = delta > 0;
  const unchanged = delta === 0;

  // A lower risk score is better.
  const changeLabel = improved ? 'Improved' : declined ? 'Declined' : 'No Change';
  const changeColorClass = improved
    ? 'text-green-600'
    : declined
    ? 'text-red-600'
    : 'text-muted-foreground';
  const deltaSign = delta > 0 ? '+' : '';

  const TrendIcon = improved ? TrendingDown : declined ? TrendingUp : Minus;
  const trendIconClass = improved
    ? 'text-green-500'
    : declined
    ? 'text-red-500'
    : 'text-muted-foreground';

  return (
    <div className="rounded-lg border bg-white p-5 space-y-4">
      <p className="text-sm font-semibold text-foreground">Period-over-Period Comparison</p>

      {/* Delta summary */}
      <div className="flex items-center gap-3">
        <TrendIcon className={`h-6 w-6 flex-shrink-0 ${trendIconClass}`} />
        <div>
          <p className={`text-2xl font-bold ${changeColorClass}`}>
            {deltaSign}{delta.toFixed(1)} pts
          </p>
          <p className={`text-sm font-medium ${changeColorClass}`}>{changeLabel}</p>
        </div>
      </div>

      {/* Before / After */}
      <div className="grid grid-cols-1 divide-x rounded-lg border overflow-hidden sm:grid-cols-2">
        <div className="px-4 py-3 space-y-0.5">
          <p className="text-xs text-muted-foreground uppercase tracking-wide font-medium">
            Previous
          </p>
          <p className="text-2xl font-bold text-foreground tabular-nums">
            {previousScore.toFixed(0)}
          </p>
          {current.previous_briefing_id && (
            <p className="text-xs text-muted-foreground">
              Grade:{' '}
              <span className="font-medium">
                {/* Approximate grade from previous score */}
                {previousScore <= 20
                  ? 'A'
                  : previousScore <= 40
                  ? 'B'
                  : previousScore <= 60
                  ? 'C'
                  : previousScore <= 80
                  ? 'D'
                  : 'F'}
              </span>
            </p>
          )}
        </div>
        <div className="px-4 py-3 space-y-0.5">
          <p className="text-xs text-muted-foreground uppercase tracking-wide font-medium">
            Current
          </p>
          <p className="text-2xl font-bold text-foreground tabular-nums">
            {currentScore.toFixed(0)}
          </p>
          <p className="text-xs text-muted-foreground">
            Grade:{' '}
            <span className="font-semibold">{current.risk_posture.grade}</span>
          </p>
        </div>
      </div>

      {/* Unchanged note */}
      {unchanged && (
        <p className="text-xs text-muted-foreground italic">
          Risk score has not changed since the last briefing.
        </p>
      )}
    </div>
  );
}
