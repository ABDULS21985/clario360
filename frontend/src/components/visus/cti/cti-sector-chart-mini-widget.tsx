'use client';

import { ArrowRight, Building2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { CTI_SEVERITY_COLORS, type CTISectorThreatSummary } from '@/types/cti';
import { formatNumber } from '@/lib/cti-utils';

interface CTISectorChartMiniWidgetProps {
  sectors: CTISectorThreatSummary[];
  isLoading: boolean;
  onExpand: () => void;
}

const SEGMENTS = [
  { key: 'severity_critical_count', color: CTI_SEVERITY_COLORS.critical },
  { key: 'severity_high_count', color: CTI_SEVERITY_COLORS.high },
  { key: 'severity_medium_count', color: CTI_SEVERITY_COLORS.medium },
  { key: 'severity_low_count', color: CTI_SEVERITY_COLORS.low },
] as const;

export function CTISectorChartMiniWidget({
  sectors,
  isLoading,
  onExpand,
}: CTISectorChartMiniWidgetProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-48" />
        </CardHeader>
        <CardContent className="space-y-3">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-11 w-full rounded-lg" />
          ))}
        </CardContent>
      </Card>
    );
  }

  const topSectors = [...sectors].sort((left, right) => right.total_count - left.total_count).slice(0, 5);
  const maxTotal = topSectors[0]?.total_count ?? 1;

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle className="text-base">Sector Targeting</CardTitle>
          <CardDescription>Industries absorbing the highest CTI event volume.</CardDescription>
        </div>
        <Building2 className="h-5 w-5 text-muted-foreground" />
      </CardHeader>
      <CardContent className="space-y-3">
        {topSectors.length === 0 ? (
          <div className="rounded-lg border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
            No sector aggregation data is available.
          </div>
        ) : (
          topSectors.map((sector) => (
            <div key={sector.id} className="space-y-2 rounded-lg border px-3 py-3">
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm font-medium">{sector.sector_label}</p>
                <span className="text-xs font-medium tabular-nums text-muted-foreground">
                  {formatNumber(sector.total_count)}
                </span>
              </div>
              <div className="h-3 overflow-hidden rounded-full bg-muted/50" style={{ width: `${Math.max((sector.total_count / maxTotal) * 100, 16)}%` }}>
                <div className="flex h-full">
                  {SEGMENTS.map((segment) => {
                    const count = sector[segment.key];
                    const width = sector.total_count > 0 ? `${(count / sector.total_count) * 100}%` : '0%';

                    return <div key={segment.key} className="h-full" style={{ width, backgroundColor: segment.color }} />;
                  })}
                </div>
              </div>
            </div>
          ))
        )}

        <Button variant="ghost" size="sm" className="px-0" onClick={onExpand}>
          View Details
          <ArrowRight className="ml-1 h-4 w-4" />
        </Button>
      </CardContent>
    </Card>
  );
}
