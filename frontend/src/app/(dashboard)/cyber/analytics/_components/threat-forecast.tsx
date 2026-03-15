'use client';

import { useQuery } from '@tanstack/react-query';
import { ArrowUp } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { ThreatForecastItem } from '@/types/cyber';

// The /threat-forecast endpoint calls PredictTechniqueTrends with a short
// 7-day horizon. We filter to increasing-only so this section is a distinct
// "imminent threat watchlist", as opposed to the 30-day TechniqueTrends table
// below which shows the full technique landscape.
interface ForecastResponse {
  items?: ThreatForecastItem[];
}

export function ThreatForecast() {
  const { data, isLoading } = useQuery({
    queryKey: ['cyber-analytics-threat-forecast', 7],
    queryFn: () =>
      apiGet<{ data: ForecastResponse }>(API_ENDPOINTS.CYBER_ANALYTICS_THREAT_FORECAST, {
        horizon_days: 7,
      }),
    refetchInterval: 300000,
  });

  // Only surface techniques predicted to grow — this distinguishes this section
  // from the broader 30-day TechniqueTrends table.
  const items = (data?.data?.items ?? []).filter((i) => i.trend === 'increasing');

  if (isLoading) {
    return <LoadingSkeleton variant="card" />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Emerging Threats — 7-Day Forecast</CardTitle>
        <CardDescription className="text-xs">
          Attack techniques predicted to increase in activity over the next 7 days, ranked by growth rate.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No techniques are forecasted to increase in the next 7 days.
          </p>
        ) : (
          <div className="space-y-2 overflow-x-auto">
            <div className="grid min-w-[520px] grid-cols-5 border-b px-2 pb-1 text-xs font-medium text-muted-foreground">
              <span className="col-span-2">Technique</span>
              <span>Growth</span>
              <span className="text-right">Predicted (p50)</span>
              <span className="text-right">Range (p10–p90)</span>
            </div>
            {items.slice(0, 10).map((item) => (
              <div
                key={item.technique_id}
                className="grid min-w-[520px] grid-cols-5 items-center rounded px-2 py-1.5 text-sm hover:bg-muted/50"
              >
                <span className="col-span-2 flex items-center gap-1.5 truncate" title={item.technique_name}>
                  <ArrowUp className="h-3 w-3 shrink-0 text-red-500" />
                  <Badge variant="outline" className="text-xs">{item.technique_id}</Badge>
                  <span className="truncate">{item.technique_name}</span>
                </span>
                <span className="text-xs tabular-nums text-red-600">
                  +{(item.growth_rate * 100).toFixed(1)}%
                </span>
                <span className="text-right text-xs tabular-nums font-medium">
                  {item.forecast.p50.toFixed(0)}
                </span>
                <span className="text-right text-xs tabular-nums text-muted-foreground">
                  {item.forecast.p10.toFixed(0)}–{item.forecast.p90.toFixed(0)}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
