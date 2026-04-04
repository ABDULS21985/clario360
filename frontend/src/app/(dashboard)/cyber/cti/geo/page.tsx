'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Globe } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { GlobalThreatMap } from '@/components/cyber/cti/global-threat-map';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { ThreatMapPopover } from '@/components/cyber/cti/threat-map-popover';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { fetchGlobalThreatMap } from '@/lib/cti-api';
import { ROUTES } from '@/lib/constants';
import type { CTIGeoThreatHotspot, CTIPeriod } from '@/types/cti';

export default function CTIGeoPage() {
  const router = useRouter();
  const [period, setPeriod] = useState<Extract<CTIPeriod, '24h' | '7d' | '30d'>>('7d');
  const [selectedHotspot, setSelectedHotspot] = useState<CTIGeoThreatHotspot | null>(null);
  const mapQuery = useQuery({
    queryKey: ['cti-global-threat-map-page', period],
    queryFn: () => fetchGlobalThreatMap(period),
  });

  const hotspots = mapQuery.data?.hotspots ?? [];
  const sortedHotspots = useMemo(
    () => [...hotspots].sort((left, right) => right.total_count - left.total_count),
    [hotspots],
  );

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Geographic Analysis"
          description="Pivot across origin hotspots and trace where the current CTI signal is concentrated around the world."
        />

        {mapQuery.error ? (
          <ErrorState message="Failed to load geographic threat map" onRetry={() => void mapQuery.refetch()} />
        ) : (
          <>
            <div className="grid gap-4 md:grid-cols-3">
              <CTIKPIStatCard label="Hotspots" value={hotspots.length} subtitle={`${period} active window`} />
              <CTIKPIStatCard label="Mapped Events" value={mapQuery.data?.total_events ?? 0} subtitle="Geo-attributed CTI events" />
              <CTIKPIStatCard label="Top Origin" value={sortedHotspots[0]?.country_code?.toUpperCase() ?? '—'} subtitle={sortedHotspots[0]?.city ?? 'No hotspot selected'} />
            </div>

            <div className="grid gap-4 xl:grid-cols-[1.35fr,0.65fr]">
              <Card>
                <CardHeader>
                  <CardTitle>Threat Hotspots</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <GlobalThreatMap
                    hotspots={hotspots}
                    period={period}
                    onPeriodChange={(nextPeriod) => setPeriod(nextPeriod as Extract<CTIPeriod, '24h' | '7d' | '30d'>)}
                    onHotspotClick={(hotspot) => setSelectedHotspot(hotspot)}
                    selectedHotspot={selectedHotspot}
                  />
                </CardContent>
              </Card>

              <div className="space-y-4">
                {selectedHotspot ? (
                  <ThreatMapPopover
                    hotspot={selectedHotspot}
                    onClose={() => setSelectedHotspot(null)}
                    onViewEvents={(countryCode) => router.push(`${ROUTES.CYBER_CTI_EVENTS}?origin_country=${countryCode}`)}
                  />
                ) : null}

                <Card>
                  <CardHeader>
                    <CardTitle>Top Locations</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {sortedHotspots.slice(0, 8).map((hotspot) => (
                      <div key={hotspot.id} className="grid gap-3 rounded-2xl border bg-background p-4 lg:grid-cols-[1.1fr,0.8fr,auto] lg:items-center">
                        <div>
                          <p className="font-medium text-foreground">{hotspot.city}, {hotspot.country_code.toUpperCase()}</p>
                          <p className="text-sm text-muted-foreground">{hotspot.total_count.toLocaleString()} mapped events</p>
                        </div>
                        <span className="text-sm text-muted-foreground">{hotspot.top_threat_type || 'Mixed activity'}</span>
                        <Button variant="outline" size="sm" onClick={() => router.push(`${ROUTES.CYBER_CTI_EVENTS}?origin_country=${hotspot.country_code}`)}>
                          View Events
                        </Button>
                      </div>
                    ))}

                    {sortedHotspots.length === 0 ? (
                      <div className="rounded-2xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
                        No geographic hotspot data is available for the selected period.
                      </div>
                    ) : null}
                  </CardContent>
                </Card>
              </div>
            </div>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}'use client';

import { useCallback, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Globe } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { GlobalThreatMap } from '@/components/cyber/cti/global-threat-map';
import { ThreatMapPopover } from '@/components/cyber/cti/threat-map-popover';
import { PeriodSelector } from '@/components/cyber/cti/period-selector';
import { fetchGlobalThreatMap } from '@/lib/cti-api';
import type { CTIGeoThreatHotspot, CTIPeriod } from '@/types/cti';

export default function CTIGeoAnalysisPage() {
  const router = useRouter();
  const [period, setPeriod] = useState<Extract<CTIPeriod, '24h' | '7d' | '30d'>>('24h');
  const [selectedHotspot, setSelectedHotspot] = useState<CTIGeoThreatHotspot | null>(null);

  const query = useQuery({
    queryKey: ['cti-geo-analysis', period],
    queryFn: () => fetchGlobalThreatMap(period),
    staleTime: 60_000,
  });

  const hotspots = query.data?.hotspots ?? [];
  const topHotspots = useMemo(() => [...hotspots].sort((left, right) => right.total_count - left.total_count).slice(0, 8), [hotspots]);

  const handleViewEvents = useCallback(
    (countryCode: string, city: string) => {
      const params = new URLSearchParams();
      params.set('origin_country', countryCode.toLowerCase());
      if (city) {
        params.set('search', city);
      }
      router.push(`/cyber/cti/events?${params.toString()}`);
    },
    [router],
  );

  if (query.isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <LoadingSkeleton variant="chart" />
      </PermissionRedirect>
    );
  }

  if (query.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState
          message="Failed to load geographic CTI analysis."
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
          title="Geographic Analysis"
          description="Inspect CTI hotspot concentration and cross-region threat activity."
          actions={<PeriodSelector value={period} onChange={(value) => setPeriod(value as typeof period)} />}
        />

        <div className="relative">
          <GlobalThreatMap
            hotspots={hotspots}
            period={period}
            onPeriodChange={(value) => setPeriod(value as typeof period)}
            onHotspotClick={setSelectedHotspot}
            selectedHotspot={selectedHotspot}
          />
          {selectedHotspot && (
            <div className="absolute right-4 top-16 z-10">
              <ThreatMapPopover
                hotspot={selectedHotspot}
                onClose={() => setSelectedHotspot(null)}
                onViewEvents={handleViewEvents}
              />
            </div>
          )}
        </div>

        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-sm">Top Hotspots</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-1 gap-3 p-4 pt-0 xl:grid-cols-2">
            {topHotspots.length > 0 ? topHotspots.map((hotspot) => (
              <button
                key={hotspot.id}
                type="button"
                className="rounded-xl border border-white/10 bg-slate-950/40 p-3 text-left transition hover:bg-slate-950/60"
                onClick={() => setSelectedHotspot(hotspot)}
              >
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="font-medium">{hotspot.city}</p>
                    <p className="text-xs text-muted-foreground">{hotspot.country_code.toUpperCase()}</p>
                  </div>
                  <span className="font-semibold tabular-nums">{hotspot.total_count.toLocaleString()}</span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  {hotspot.top_threat_type || 'Unknown threat type'}
                </p>
              </button>
            )) : (
              <div className="rounded-xl border border-dashed border-white/10 p-8 text-center text-sm text-muted-foreground xl:col-span-2">
                <Globe className="mx-auto mb-2 h-5 w-5" />
                No hotspot data available for this period.
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </PermissionRedirect>
  );
}
