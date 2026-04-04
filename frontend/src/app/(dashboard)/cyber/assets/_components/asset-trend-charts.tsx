'use client';

import { useMemo } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import type { AssetStats } from '@/types/cyber';

interface TrendDataPoint {
  label: string;
  value: number;
  color: string;
}

export function AssetTrendCharts() {
  const { data: envelope, isLoading } = useRealtimeData<{ data: AssetStats }>(
    API_ENDPOINTS.CYBER_ASSETS_STATS,
    { pollInterval: 120_000 },
  );
  const stats = envelope?.data;

  const typeData = useMemo<TrendDataPoint[]>(() => {
    if (!stats?.by_type) return [];
    return Object.entries(stats.by_type)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 8)
      .map(([type, count]) => ({
        label: type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
        value: count,
        color: TYPE_COLORS[type] ?? '#64748B',
      }));
  }, [stats]);

  const critData = useMemo<TrendDataPoint[]>(() => {
    if (!stats?.by_criticality) return [];
    return ['critical', 'high', 'medium', 'low']
      .filter((k) => (stats.by_criticality[k] ?? 0) > 0)
      .map((k) => ({
        label: k.charAt(0).toUpperCase() + k.slice(1),
        value: stats.by_criticality[k] ?? 0,
        color: CRIT_COLORS[k] ?? '#64748B',
      }));
  }, [stats]);

  const total = stats?.total ?? 0;

  if (isLoading || !stats) return null;

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-sm">Assets by Type</CardTitle>
        </CardHeader>
        <CardContent className="p-4 pt-0">
          <div className="space-y-2">
            {typeData.map((d) => (
              <div key={d.label} className="flex items-center gap-3">
                <span className="w-28 truncate text-xs text-muted-foreground">{d.label}</span>
                <div className="flex-1">
                  <div className="h-4 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full transition-all duration-500"
                      style={{
                        width: `${total > 0 ? Math.max((d.value / total) * 100, 2) : 0}%`,
                        backgroundColor: d.color,
                      }}
                    />
                  </div>
                </div>
                <span className="w-10 text-right text-xs font-medium tabular-nums">{d.value}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-sm">Assets by Criticality</CardTitle>
        </CardHeader>
        <CardContent className="p-4 pt-0">
          <div className="flex items-end gap-4 justify-center h-32">
            {critData.map((d) => {
              const maxVal = Math.max(...critData.map((c) => c.value), 1);
              const heightPct = Math.max((d.value / maxVal) * 100, 8);
              return (
                <div key={d.label} className="flex flex-col items-center gap-1">
                  <span className="text-xs font-medium tabular-nums">{d.value}</span>
                  <div
                    className="w-12 rounded-t-md transition-all duration-500"
                    style={{ height: `${heightPct}%`, backgroundColor: d.color }}
                  />
                  <span className="text-[10px] text-muted-foreground">{d.label}</span>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

const TYPE_COLORS: Record<string, string> = {
  server: '#3B82F6',
  endpoint: '#8B5CF6',
  cloud_resource: '#06B6D4',
  network_device: '#F59E0B',
  iot_device: '#10B981',
  application: '#EC4899',
  database: '#EF4444',
  container: '#6366F1',
};

const CRIT_COLORS: Record<string, string> = {
  critical: '#EF4444',
  high: '#F97316',
  medium: '#EAB308',
  low: '#3B82F6',
};
