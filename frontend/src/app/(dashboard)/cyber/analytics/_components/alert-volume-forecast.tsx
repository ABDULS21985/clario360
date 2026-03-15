'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { AreaChart } from '@/components/shared/charts/area-chart';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';

// Mirrors backend model.ForecastPoint + model.AlertVolumeForecast
interface AlertForecastData {
  forecast?: {
    horizon_days?: number;
    // Backend field is "points", each point has { timestamp, value, bounds }
    points?: Array<{
      timestamp: string;
      value: number;
      bounds: { p10: number; p50: number; p90: number };
    }>;
    anomaly_flag?: boolean;
  };
}

export function AlertVolumeForecast() {
  const { data, isLoading } = useQuery({
    queryKey: ['cyber-analytics-alert-forecast'],
    queryFn: () => apiGet<{ data: AlertForecastData }>(API_ENDPOINTS.CYBER_ANALYTICS_ALERT_FORECAST),
    refetchInterval: 300000,
  });

  const points = data?.data?.forecast?.points ?? [];

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
    date: new Date(p.timestamp).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    predicted: p.value,
    lower: p.bounds.p10,
    upper: p.bounds.p90,
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
