'use client';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { AccessRecommendation } from '@/types/cyber';

interface RecommendationCardsProps {
  recommendations: AccessRecommendation[];
}

function typeBadgeVariant(type: string) {
  switch (type) {
    case 'revoke':
      return 'destructive' as const;
    case 'downgrade':
      return 'warning' as const;
    case 'time_bound':
      return 'default' as const;
    case 'review':
      return 'secondary' as const;
    default:
      return 'outline' as const;
  }
}

function formatLabel(value: string): string {
  return value
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export function RecommendationCards({ recommendations }: RecommendationCardsProps) {
  if (recommendations.length === 0) {
    return (
      <div className="rounded-lg border bg-muted/20 p-6 text-center text-sm text-muted-foreground">
        No recommendations available for this identity.
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      {recommendations.map((rec) => (
        <Card key={`${rec.permission_id}-${rec.type}`}>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between gap-2">
              <CardTitle className="text-sm">{rec.asset_name}</CardTitle>
              <Badge variant={typeBadgeVariant(rec.type)}>{formatLabel(rec.type)}</Badge>
            </div>
            <p className="text-xs text-muted-foreground">
              {formatLabel(rec.permission_type)}
            </p>
          </CardHeader>
          <CardContent className="space-y-2">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Reason</p>
              <p className="text-sm">{rec.reason}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-muted-foreground">Impact</p>
              <p className="text-sm">{rec.impact}</p>
            </div>
            <div className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2">
              <span className="text-xs text-muted-foreground">Risk Reduction</span>
              <span className="text-sm font-semibold text-green-600">
                {Math.round(rec.risk_reduction_estimate * 100)}%
              </span>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
