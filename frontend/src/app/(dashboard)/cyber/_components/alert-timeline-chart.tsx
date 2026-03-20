'use client';

import { useMemo } from 'react';
import { format, parseISO } from 'date-fns';
import { LineChart } from '@/components/shared/charts/line-chart';
import type { AlertTimelineData } from '@/types/cyber';

interface AlertTimelineChartProps {
  data?: AlertTimelineData;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
}

// The backend returns granular timeline with a single "count" per bucket.
// We display the timeline as a single line by default.
export function AlertTimelineChart({ data, loading, error, onRetry }: AlertTimelineChartProps) {
  const chartData = useMemo(() => {
    if (!data?.points) return [];
    return data.points.map((pt) => ({
      hour: format(parseISO(pt.bucket), 'HH:mm'),
      count: pt.count,
    }));
  }, [data]);

  return (
    <LineChart
      title="Alert Volume (24h)"
      data={chartData}
      xKey="hour"
      yKeys={[{ key: 'count', label: 'Alerts', color: '#3B82F6' }]}
      loading={loading}
      error={error}
      onRetry={onRetry}
      height={220}
      showGrid
      showLegend={false}
      className="h-full"
    />
  );
}
