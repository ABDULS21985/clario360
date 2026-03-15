'use client';

import { Badge } from '@/components/ui/badge';
import { type ModelQualityScore } from '@/lib/data-suite/types';
import { getClassificationBadge } from '@/lib/data-suite/utils';

interface QualityModelCardsProps {
  items: ModelQualityScore[];
}

export function QualityModelCards({
  items,
}: QualityModelCardsProps) {
  if (items.length === 0) {
    return <p className="text-sm text-muted-foreground">No model quality scores are available.</p>;
  }

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      {items.map((item) => {
        const classification = getClassificationBadge(item.classification);
        return (
          <div key={item.model_id} className="rounded-lg border bg-card p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="font-medium">{item.model_name}</div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {item.total_rules} rules • {item.failed_rules} failed
                </div>
              </div>
              <Badge variant="outline" className={classification.className}>
                {classification.label}
              </Badge>
            </div>
            <div className="mt-4">
              <div className="mb-1 flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Quality score</span>
                <span className="font-medium">{item.score.toFixed(1)}</span>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(item.score, 100)}%` }} />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
