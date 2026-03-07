'use client';

import { LineChart } from '@/components/shared/charts/line-chart';
import { type QualityTrendPoint } from '@/lib/data-suite';

interface QualityTrendChartProps {
  trend: QualityTrendPoint[];
}

export function QualityTrendChart({
  trend,
}: QualityTrendChartProps) {
  return (
    <LineChart
      data={trend.map((point) => ({ day: point.day, score: point.score }))}
      xKey="day"
      yKeys={[{ key: 'score', label: 'Quality score', color: '#2563eb' }]}
      height={320}
    />
  );
}
