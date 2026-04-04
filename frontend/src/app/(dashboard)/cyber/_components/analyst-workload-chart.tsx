'use client';

import { BarChart } from '@/components/shared/charts/bar-chart';
import type { AnalystWorkloadEntry } from '@/types/cyber';

interface AnalystWorkloadChartProps {
  data?: AnalystWorkloadEntry[];
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
}

export function AnalystWorkloadChart({ data, loading, error, onRetry }: AnalystWorkloadChartProps) {
  const chartData = (data ?? []).map((entry) => ({
    name: entry.name.split(' ')[0], // First name only to save space
    open_assigned: entry.open_assigned,
    critical_open: entry.critical_open,
  }));

  return (
    <BarChart
      title="Analyst Workload"
      data={chartData}
      xKey="name"
      yKeys={[
        { key: 'open_assigned', label: 'Open', color: '#3B82F6' },
        { key: 'critical_open', label: 'Critical', color: '#DC2626' },
      ]}
      layout="vertical"
      loading={loading}
      error={error}
      onRetry={onRetry}
      height={260}
      showLegend
      showGrid
    />
  );
}
