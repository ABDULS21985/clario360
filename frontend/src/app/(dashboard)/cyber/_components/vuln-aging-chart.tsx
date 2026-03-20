'use client';

import { useMemo } from 'react';
import { BarChart } from '@/components/shared/charts/bar-chart';
interface VulnAgingChartProps {
  data?: VulnerabilityAgingReport;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
}

// The aging report is optionally returned from a separate endpoint.
// Accept raw AgingBucket[] too.
export interface VulnerabilityAgingReport {
  buckets: Array<{
    label: string;
    by_severity: Record<string, { count: number }>;
    total: number;
  }>;
  total_open: number;
  avg_age_days: number;
}

export function VulnAgingChart({ data, loading, error, onRetry }: VulnAgingChartProps) {
  const chartData = useMemo(() => {
    if (!data?.buckets) return [];
    return data.buckets.map((b) => ({
      bucket: b.label,
      critical: b.by_severity['critical']?.count ?? 0,
      high: b.by_severity['high']?.count ?? 0,
      medium: b.by_severity['medium']?.count ?? 0,
      low: b.by_severity['low']?.count ?? 0,
    }));
  }, [data]);

  return (
    <BarChart
      title="Vulnerability Aging"
      data={chartData}
      xKey="bucket"
      yKeys={[
        { key: 'critical', label: 'Critical', color: '#DC2626' },
        { key: 'high', label: 'High', color: '#EA580C' },
        { key: 'medium', label: 'Medium', color: '#CA8A04' },
        { key: 'low', label: 'Low', color: '#2563EB' },
      ]}
      stacked
      loading={loading}
      error={error}
      onRetry={onRetry}
      height={220}
      showLegend
      className="h-full"
    />
  );
}
