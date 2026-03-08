'use client';

import Link from 'next/link';
import { ShieldAlert } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import type { UebaRiskRankingItem } from './types';

export function RiskRankingChart({ items }: { items: UebaRiskRankingItem[] }) {
  const maxScore = Math.max(...items.map((item) => item.risk_score), 1);

  return (
    <div className="space-y-3">
      {items.slice(0, 20).map((item) => (
        <Link
          key={item.entity_id}
          href={`/cyber/ueba/profiles/${encodeURIComponent(item.entity_id)}`}
          className="block rounded-lg border border-border/70 bg-background/60 p-3 transition hover:border-primary/40 hover:bg-background"
        >
          <div className="mb-2 flex items-center justify-between gap-3">
            <div className="min-w-0">
              <div className="truncate font-medium">{item.entity_name}</div>
              <div className="text-xs text-muted-foreground">
                {item.entity_type.replaceAll('_', ' ')} · {item.alert_count_7d} alerts in 7d
              </div>
            </div>
            <Badge variant={item.risk_score >= 75 ? 'destructive' : item.risk_score >= 50 ? 'warning' : 'outline'}>
              {item.risk_score.toFixed(0)}
            </Badge>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-gradient-to-r from-amber-500 via-orange-500 to-red-600"
              style={{ width: `${(item.risk_score / maxScore) * 100}%` }}
            />
          </div>
        </Link>
      ))}
      {items.length === 0 && (
        <div className="flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
          <ShieldAlert className="h-4 w-4" />
          No ranked entities yet.
        </div>
      )}
    </div>
  );
}
