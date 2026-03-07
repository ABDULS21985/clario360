'use client';

import { GaugeChart } from '@/components/shared/charts/gauge-chart';
import { type QualityScore } from '@/lib/data-suite';

interface QualityScoreGaugeProps {
  score: QualityScore;
}

export function QualityScoreGauge({
  score,
}: QualityScoreGaugeProps) {
  return (
    <div className="flex flex-col items-center justify-center gap-4 rounded-lg border bg-card p-6">
      <GaugeChart value={score.overall_score} size={180} />
      <div className="text-center">
        <div className="text-sm text-muted-foreground">Overall grade</div>
        <div className="text-3xl font-semibold">{score.grade}</div>
        <div className="mt-1 text-sm text-muted-foreground">
          {score.passed_rules} passed • {score.failed_rules} failed • {score.warning_rules} warnings
        </div>
      </div>
    </div>
  );
}
