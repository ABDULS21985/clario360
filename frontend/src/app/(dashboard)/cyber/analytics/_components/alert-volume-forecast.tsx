'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { AreaChart } from '@/components/shared/charts/area-chart';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';

interface AlertForecastData {
  forecast?: {
    daily_points?: Array<{
      date: string;
      predicted: number;
      lower: number;
      upper: number;
      actual?: number;
    }>;
  };
}

export function AlertVolumeForecast() {
  const { data, isLoading } = useQuery({
    queryKey: ['cyber-analytics-alert-forecast'],
    queryFn: () => apiGet<{ data: AlertForecastData }>(API_ENDPOINTS.CYBER_ANALYTICS_ALERT_FORECAST),
    refetchInterval: 300000,
  });

  const points = data?.data?.forecast?.daily_points ?? [];

  if (isLoading) {
    return <LoadingSkeleton variant="card" />;
  }

  if (points.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Alert Volume Forecast</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Insufficient data to generate alert volume forecast.
          </p>
        </CardContent>
      </Card>
    );
  }

  const chartData = points.map((p) => ({
    date: new Date(p.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    predicted: p.predicted,
    lower: p.lower,
    upper: p.upper,
    actual: p.actual ?? null,
  }));

  return (
    <AreaChart
      title="Alert Volume Forecast (30 Days)"
      data={chartData}
      xKey="date"
      yKeys={[
        { key: 'predicted', label: 'Predicted', color: '#3B82F6' },
        { key: 'upper', label: 'Upper Bound', color: '#93C5FD' },
        { key: 'lower', label: 'Lower Bound', color: '#BFDBFE' },
      ]}
      stacked={false}
      height={320}
    />
  );
}
