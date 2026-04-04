'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { PieChart } from '@/components/shared/charts/pie-chart';
import type { PaginatedResponse } from '@/types/api';

interface CountResponse extends PaginatedResponse<unknown> {
  total: number;
}

const STATUSES = [
  { key: 'running', label: 'Running', color: '#3b82f6' },
  { key: 'completed', label: 'Completed', color: '#22c55e' },
  { key: 'failed', label: 'Failed', color: '#ef4444' },
  { key: 'cancelled', label: 'Cancelled', color: '#6b7280' },
  { key: 'suspended', label: 'Suspended', color: '#eab308' },
];

export function InstanceStatusChart() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['workflow-analytics-status-breakdown'],
    queryFn: async () => {
      const results = await Promise.all(
        STATUSES.map((s) =>
          apiGet<CountResponse>('/api/v1/workflows/instances', {
            per_page: 1,
            page: 1,
            status: s.key,
          }),
        ),
      );
      return STATUSES.map((s, i) => ({
        name: s.label,
        value: results[i].total ?? 0,
        color: s.color,
      }));
    },
    staleTime: 60_000,
  });

  const chartData = (data ?? []).filter((d) => d.value > 0);
  const total = chartData.reduce((sum, d) => sum + d.value, 0);

  return (
    <PieChart
      title="Instances by Status"
      data={chartData}
      loading={isLoading}
      error={isError ? 'Failed to load' : undefined}
      height={280}
      showLegend
      centerLabel="Total"
      centerValue={String(total)}
    />
  );
}
