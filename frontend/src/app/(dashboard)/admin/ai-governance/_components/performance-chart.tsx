'use client';

import { LineChart } from '@/components/shared/charts/line-chart';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { AIPerformancePoint } from '@/types/ai-governance';

interface PerformanceChartProps {
  points: AIPerformancePoint[];
}

export function PerformanceChart({ points }: PerformanceChartProps) {
  const chartData = points
    .slice()
    .reverse()
    .map((point) => ({
      period: new Date(point.period_start).toLocaleDateString(),
      volume: point.volume,
      avg_latency_ms: point.avg_latency_ms ?? 0,
      accuracy: point.accuracy ? Math.round(point.accuracy * 100) : 0,
    }));

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <Card className="border-border/70">
        <CardHeader>
          <CardTitle>Volume and Latency</CardTitle>
        </CardHeader>
        <CardContent>
          <LineChart
            data={chartData}
            xKey="period"
            yKeys={[
              { key: 'volume', label: 'Volume', color: '#1d4ed8' },
              { key: 'avg_latency_ms', label: 'Avg latency (ms)', color: '#dc2626', dashed: true },
            ]}
            height={320}
          />
        </CardContent>
      </Card>

      <Card className="border-border/70">
        <CardHeader>
          <CardTitle>Accuracy</CardTitle>
        </CardHeader>
        <CardContent>
          <LineChart
            data={chartData}
            xKey="period"
            yKeys={[{ key: 'accuracy', label: 'Accuracy %', color: '#0f766e' }]}
            height={320}
          />
        </CardContent>
      </Card>
    </div>
  );
}
