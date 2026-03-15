'use client';

import { LineChart } from '@/components/shared/charts/line-chart';
import { type QualityTrendPoint } from '@/lib/data-suite';

interface QualityTrendChartProps {
  trend: QualityTrendPoint[];
}

function formatTrendDay(value: string | number): string {
  if (!value) return '';
  try {
    return new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  } catch {
    return String(value);
  }
}

export function QualityTrendChart({
  trend,
}: QualityTrendChartProps) {
  return (
    <LineChart
      data={trend.map((point) => ({ day: point.day, score: point.score }))}
      xKey="day"
      yKeys={[{ key: 'score', label: 'Quality score', color: '#2563eb' }]}
      xFormatter={formatTrendDay}
      height={320}
    />
  );
}
