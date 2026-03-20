'use client';

import { useMemo } from 'react';
import { PieChart } from '@/components/shared/charts/pie-chart';
import type { SeverityDistribution } from '@/types/cyber';

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#DC2626',
  high: '#EA580C',
  medium: '#CA8A04',
  low: '#2563EB',
  info: '#6B7280',
};

interface SeverityDistributionChartProps {
  data?: SeverityDistribution;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
}

export function SeverityDistributionChart({ data, loading, error, onRetry }: SeverityDistributionChartProps) {
  const { chartData, total } = useMemo(() => {
    if (!data?.counts) return { chartData: [], total: 0 };
    const items = Object.entries(data.counts).map(([name, value]) => ({
      name: name.charAt(0).toUpperCase() + name.slice(1),
      value,
      color: SEVERITY_COLORS[name] ?? '#9CA3AF',
    }));
    return { chartData: items, total: data.total };
  }, [data]);

  return (
    <PieChart
      title="Severity Distribution"
      data={chartData}
      centerLabel="total"
      centerValue={String(total)}
      loading={loading}
      error={error}
      onRetry={onRetry}
      height={220}
      showLegend
      innerRadius={55}
      outerRadius={90}
    />
  );
}
