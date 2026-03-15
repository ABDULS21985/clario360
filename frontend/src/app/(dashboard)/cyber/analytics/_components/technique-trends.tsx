'use client';

import { useQuery } from '@tanstack/react-query';
import { ArrowUp, ArrowDown, Minus } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { ThreatForecastItem } from '@/types/cyber';

interface TrendsResponse {
  items?: ThreatForecastItem[];
}

export function TechniqueTrends() {
  const { data, isLoading } = useQuery({
    queryKey: ['cyber-analytics-technique-trends'],
    queryFn: () =>
      apiGet<{ data: TrendsResponse }>(API_ENDPOINTS.CYBER_ANALYTICS_TECHNIQUE_TRENDS, {
        horizon_days: 30,
      }),
    refetchInterval: 300000,
  });

  const items = data?.data?.items ?? [];

  if (isLoading) {
    return <LoadingSkeleton variant="card" />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Attack Technique Trends (30 Days)</CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No technique trend data available yet.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-xs text-muted-foreground">
                  <th className="text-left py-2 pr-4">Technique</th>
                  <th className="text-left py-2 pr-4">ID</th>
                  <th className="text-left py-2 pr-4">Trend</th>
                  <th className="text-right py-2 pr-4">Growth</th>
                  <th className="text-right py-2">Predicted (p50)</th>
                  <th className="text-right py-2">Range (p10–p90)</th>
                </tr>
              </thead>
              <tbody>
                {items.slice(0, 20).map((item) => (
                  <tr key={item.technique_id} className="border-b last:border-0 hover:bg-muted/50">
                    <td className="py-2 pr-4 max-w-[120px] sm:max-w-[200px] truncate">{item.technique_name}</td>
                    <td className="py-2 pr-4">
                      <Badge variant="outline" className="text-xs">{item.technique_id}</Badge>
                    </td>
                    <td className="py-2 pr-4">
                      <span className="flex items-center gap-1">
                        {item.trend === 'increasing' && <ArrowUp className="h-3.5 w-3.5 text-red-500" />}
                        {item.trend === 'decreasing' && <ArrowDown className="h-3.5 w-3.5 text-green-500" />}
                        {item.trend === 'stable' && <Minus className="h-3.5 w-3.5 text-muted-foreground" />}
                        <span className="text-xs capitalize">{item.trend}</span>
                      </span>
                    </td>
                    <td className={`py-2 pr-4 text-right tabular-nums text-xs ${item.growth_rate > 0 ? 'text-red-600' : item.growth_rate < 0 ? 'text-green-600' : ''}`}>
                      {item.growth_rate > 0 ? '+' : ''}{(item.growth_rate * 100).toFixed(1)}%
                    </td>
                    <td className="py-2 pr-2 text-right tabular-nums text-xs font-medium">
                      {item.forecast.p50.toFixed(0)}
                    </td>
                    <td className="py-2 text-right tabular-nums text-xs text-muted-foreground">
                      {item.forecast.p10.toFixed(0)}–{item.forecast.p90.toFixed(0)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
