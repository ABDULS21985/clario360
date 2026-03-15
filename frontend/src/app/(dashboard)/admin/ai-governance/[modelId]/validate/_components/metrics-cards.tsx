'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatPercentage } from '@/lib/format';
import type { AIValidationResult } from '@/types/ai-governance';
import { ComparisonIndicator } from './comparison-indicator';

interface MetricsCardsProps {
  result: AIValidationResult;
}

export function MetricsCards({ result }: MetricsCardsProps) {
  const cards = [
    {
      title: 'Precision',
      value: result.precision,
      delta: result.deltas?.precision,
      inverse: false,
    },
    {
      title: 'Recall',
      value: result.recall,
      delta: result.deltas?.recall,
      inverse: false,
    },
    {
      title: 'F1 Score',
      value: result.f1_score,
      delta: result.deltas?.f1_score,
      inverse: false,
    },
    {
      title: 'FP Rate',
      value: result.false_positive_rate,
      delta: result.deltas?.false_positive_rate,
      inverse: true,
    },
  ];

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
      {cards.map((card) => (
        <Card key={card.title} className="border-border/70">
          <CardHeader className="pb-3">
            <CardTitle className="text-base">{card.title}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="text-4xl font-semibold tracking-[-0.06em] text-slate-950">
              {formatPercentage(card.value, 1)}
            </div>
            <ComparisonIndicator delta={card.delta} inverse={card.inverse} />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
