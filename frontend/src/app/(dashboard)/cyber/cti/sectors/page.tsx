'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { BarChart3 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { PeriodSelector } from '@/components/cyber/cti/period-selector';
import { SectorThreatChart } from '@/components/cyber/cti/sector-threat-chart';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { fetchSectorThreatOverview } from '@/lib/cti-api';
import { ROUTES } from '@/lib/constants';
import type { CTIPeriod } from '@/types/cti';

export default function CTISectorsPage() {
  const router = useRouter();
  const [period, setPeriod] = useState<Extract<CTIPeriod, '24h' | '7d' | '30d'>>('7d');
  const sectorsQuery = useQuery({
    queryKey: ['cti-sector-threat-overview', period],
    queryFn: () => fetchSectorThreatOverview(period),
  });

  const sectors = sectorsQuery.data?.sectors ?? [];
  const sortedSectors = useMemo(
    () => [...sectors].sort((left, right) => right.total_count - left.total_count),
    [sectors],
  );
  const totalEvents = sectors.reduce((sum, sector) => sum + sector.total_count, 0);
  const criticalEvents = sectors.reduce((sum, sector) => sum + sector.severity_critical_count, 0);
  const topSector = sortedSectors[0];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Sector Targeting"
          description="Understand which industries are under the most pressure and pivot straight into filtered event investigation."
          actions={<PeriodSelector value={period} onChange={(nextPeriod) => setPeriod(nextPeriod as Extract<CTIPeriod, '24h' | '7d' | '30d'>)} />}
        />

        {sectorsQuery.error ? (
          <ErrorState message="Failed to load sector threat overview" onRetry={() => void sectorsQuery.refetch()} />
        ) : (
          <>
            <div className="grid gap-4 md:grid-cols-3">
              <CTIKPIStatCard label="Impacted Sectors" value={sortedSectors.length} subtitle={`${period} reporting window`} />
              <CTIKPIStatCard label="Total Sector Events" value={totalEvents} subtitle="Aggregated across all sectors" />
              <CTIKPIStatCard label="Critical Events" value={criticalEvents} subtitle="Critical severity pressure" color="#FF3B5C" />
            </div>

            <SectorThreatChart
              sectors={sectors}
              loading={sectorsQuery.isLoading}
              error={sectorsQuery.error instanceof Error ? sectorsQuery.error.message : undefined}
              onRetry={() => void sectorsQuery.refetch()}
            />

            <div className="grid gap-4 lg:grid-cols-[1.2fr,0.8fr]">
              <Card>
                <CardHeader>
                  <CardTitle>Sector Breakdown</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {sortedSectors.map((sector) => (
                    <div key={sector.id} className="grid gap-3 rounded-2xl border bg-background p-4 lg:grid-cols-[1.4fr,0.8fr,auto] lg:items-center">
                      <div>
                        <p className="font-medium text-foreground">{sector.sector_label}</p>
                        <p className="text-sm text-muted-foreground">{sector.total_count.toLocaleString()} total events in {period}</p>
                      </div>
                      <div className="text-sm text-muted-foreground">
                        {sector.severity_critical_count} critical / {sector.severity_high_count} high
                      </div>
                      <Button variant="outline" size="sm" onClick={() => router.push(`${ROUTES.CYBER_CTI_EVENTS}?target_sector=${sector.sector_id}`)}>
                        View Events
                      </Button>
                    </div>
                  ))}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Top Sector</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  {topSector ? (
                    <>
                      <p className="text-2xl font-semibold tracking-[-0.04em] text-slate-950">{topSector.sector_label}</p>
                      <p className="text-muted-foreground">{topSector.total_count.toLocaleString()} total events during the selected period.</p>
                      <Button variant="outline" onClick={() => router.push(`${ROUTES.CYBER_CTI_EVENTS}?target_sector=${topSector.sector_id}`)}>
                        Investigate Events
                      </Button>
                    </>
                  ) : (
                    <div className="rounded-2xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
                      No sector telemetry is available for the selected period.
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { BarChart3 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { PeriodSelector } from '@/components/cyber/cti/period-selector';
import { fetchSectorThreatOverview } from '@/lib/cti-api';
import type { CTIPeriod } from '@/types/cti';

export default function CTISectorTargetingPage() {
  const [period, setPeriod] = useState<Extract<CTIPeriod, '24h' | '7d' | '30d'>>('24h');

  const query = useQuery({
    queryKey: ['cti-sector-targeting', period],
    queryFn: () => fetchSectorThreatOverview(period),
    staleTime: 60_000,
  });

  const sectors = useMemo(
    () => [...(query.data?.sectors ?? [])].sort((left, right) => right.total_count - left.total_count),
    [query.data?.sectors],
  );

  if (query.isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <LoadingSkeleton variant="card" />
      </PermissionRedirect>
    );
  }

  if (query.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState
          message="Failed to load CTI sector targeting."
          onRetry={() => {
            void query.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Sector Targeting"
          description="Review which industries are most targeted across the selected CTI period."
          actions={<PeriodSelector value={period} onChange={(value) => setPeriod(value as typeof period)} />}
        />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {sectors.length > 0 ? sectors.map((sector) => (
            <Card key={sector.id}>
              <CardHeader className="p-4 pb-2">
                <CardTitle className="flex items-center justify-between gap-3 text-sm">
                  <span>{sector.sector_label}</span>
                  <span className="font-semibold tabular-nums">{sector.total_count.toLocaleString()}</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 p-4 pt-0">
                <div className="flex flex-wrap gap-2">
                  {sector.severity_critical_count > 0 && (
                    <CTISeverityBadge severity="critical" size="sm" className="capitalize" />
                  )}
                  {sector.severity_high_count > 0 && (
                    <CTISeverityBadge severity="high" size="sm" className="capitalize" />
                  )}
                  {sector.severity_medium_count > 0 && (
                    <CTISeverityBadge severity="medium" size="sm" className="capitalize" />
                  )}
                  {sector.severity_low_count > 0 && (
                    <CTISeverityBadge severity="low" size="sm" className="capitalize" />
                  )}
                </div>
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="rounded-xl border border-white/10 bg-slate-950/40 p-3">
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Critical</p>
                    <p className="mt-1 font-semibold tabular-nums">{sector.severity_critical_count}</p>
                  </div>
                  <div className="rounded-xl border border-white/10 bg-slate-950/40 p-3">
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">High</p>
                    <p className="mt-1 font-semibold tabular-nums">{sector.severity_high_count}</p>
                  </div>
                  <div className="rounded-xl border border-white/10 bg-slate-950/40 p-3">
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Medium</p>
                    <p className="mt-1 font-semibold tabular-nums">{sector.severity_medium_count}</p>
                  </div>
                  <div className="rounded-xl border border-white/10 bg-slate-950/40 p-3">
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Low</p>
                    <p className="mt-1 font-semibold tabular-nums">{sector.severity_low_count}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )) : (
            <Card className="xl:col-span-2">
              <CardContent className="p-8 text-center text-sm text-muted-foreground">
                <BarChart3 className="mx-auto mb-2 h-5 w-5" />
                No sector targeting data available for this period.
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </PermissionRedirect>
  );
}
