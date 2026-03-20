'use client';

import { format } from 'date-fns';
import type { UebaTrendDatum } from './types';

export function AlertTrendChart({ items }: { items: UebaTrendDatum[] }) {
  const buckets = new Map<string, number>();
  items.forEach((item) => {
    buckets.set(item.bucket, (buckets.get(item.bucket) ?? 0) + item.count);
  });
  const entries = Array.from(buckets.entries()).sort(([left], [right]) => left.localeCompare(right));
  const maxValue = Math.max(...entries.map(([, value]) => value), 1);

  return (
    <div className="space-y-3">
      {entries.slice(-30).map(([bucket, count]) => (
        <div key={bucket} className="grid grid-cols-[72px_1fr_32px] items-center gap-3">
          <span className="text-xs text-muted-foreground">{format(new Date(bucket), 'MMM d')}</span>
          <div className="h-2 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-lime-400"
              style={{ width: `${(count / maxValue) * 100}%` }}
            />
          </div>
          <span className="text-right text-xs font-medium">{count}</span>
        </div>
      ))}
      {entries.length === 0 && <p className="text-sm text-muted-foreground">No trend data yet.</p>}
    </div>
  );
}
