'use client';

import { Card, CardContent } from '@/components/ui/card';
import { type DarkDataStatsSummary } from '@/lib/data-suite';
import { formatMaybeBytes } from '@/lib/data-suite/utils';

interface DarkDataKpiCardsProps {
  stats: DarkDataStatsSummary;
}

export function DarkDataKpiCards({
  stats,
}: DarkDataKpiCardsProps) {
  const items = [
    { label: 'Total Assets', value: stats.total_assets.toLocaleString() },
    { label: 'High Risk', value: stats.high_risk_assets.toLocaleString() },
    { label: 'With PII', value: stats.pii_assets.toLocaleString() },
    { label: 'Total Size', value: formatMaybeBytes(stats.total_size_bytes) },
  ];

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
      {items.map((item) => (
        <Card key={item.label}>
          <CardContent className="py-5">
            <div className="text-xs uppercase tracking-wide text-muted-foreground">{item.label}</div>
            <div className="mt-1 text-2xl font-semibold">{item.value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
