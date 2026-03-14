'use client';

import { useQuery } from '@tanstack/react-query';
import { ArrowUp, ArrowDown, Minus } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { ThreatForecastItem } from '@/types/cyber';

interface ForecastResponse {
  items?: ThreatForecastItem[];
}

export function ThreatForecast() {
  const { data, isLoading } = useQuery({
    queryKey: ['cyber-analytics-threat-forecast'],
    queryFn: () => apiGet<{ data: ForecastResponse }>(API_ENDPOINTS.CYBER_ANALYTICS_THREAT_FORECAST),
    refetchInterval: 300000,
  });

  const items = data?.data?.items ?? [];

  if (isLoading) {
    return <LoadingSkeleton variant="card" />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Next 30-Day Threat Forecast</CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Insufficient data to generate threat forecast. More historical data is needed.
          </p>
        ) : (
          <div className="space-y-2">
            <div className="grid grid-cols-5 text-xs font-medium text-muted-foreground px-2 pb-1 border-b">
              <span className="col-span-2">Technique</span>
              <span>Trend</span>
              <span>Growth</span>
              <span>Confidence</span>
            </div>
            {items.slice(0, 15).map((item) => (
              <div
                key={item.technique_id}
                className="grid grid-cols-5 items-center text-sm px-2 py-1.5 rounded hover:bg-muted/50"
              >
                <span className="col-span-2 truncate" title={item.technique_name}>
                  <Badge variant="outline" className="text-xs mr-1.5">{item.technique_id}</Badge>
                  {item.technique_name}
                </span>
                <span className="flex items-center gap-1">
                  {item.trend === 'increasing' && (
                    <ArrowUp className="h-3.5 w-3.5 text-red-500" />
                  )}
                  {item.trend === 'decreasing' && (
                    <ArrowDown className="h-3.5 w-3.5 text-green-500" />
                  )}
                  {item.trend === 'stable' && (
                    <Minus className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                  <span className="text-xs capitalize">{item.trend}</span>
                </span>
                <span className={`text-xs tabular-nums ${item.growth_rate > 0 ? 'text-red-600' : item.growth_rate < 0 ? 'text-green-600' : 'text-muted-foreground'}`}>
                  {item.growth_rate > 0 ? '+' : ''}{(item.growth_rate * 100).toFixed(1)}%
                </span>
                <span className="text-xs tabular-nums text-muted-foreground">
                  {item.forecast.lower.toFixed(0)}–{item.forecast.upper.toFixed(0)}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
