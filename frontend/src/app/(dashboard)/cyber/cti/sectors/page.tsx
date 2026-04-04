'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { PeriodSelector } from '@/components/cyber/cti/period-selector';
import { SectorThreatChart } from '@/components/cyber/cti/sector-threat-chart';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { fetchCampaigns, fetchSectorThreatOverview, fetchSectors, fetchThreatEvents } from '@/lib/cti-api';
import { ROUTES } from '@/lib/constants';
import { formatRelativeTime } from '@/lib/cti-utils';
import { CTI_SEVERITY_COLORS, type CTIPeriod } from '@/types/cti';

function TrendSparkline({ points }: { points: number[] }) {
  const width = 320;
  const height = 96;
  const max = Math.max(...points, 1);
  const path = points
    .map((point, index) => {
      const x = (index / Math.max(points.length - 1, 1)) * width;
      const y = height - (point / max) * (height - 12) - 6;
      return `${index === 0 ? 'M' : 'L'} ${x} ${y}`;
    })
    .join(' ');

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="h-24 w-full">
      <path d={path} fill="none" stroke="#0EA5E9" strokeWidth="3" strokeLinecap="round" />
    </svg>
  );
}

export default function CTISectorsPage() {
  const [period, setPeriod] = useState<Extract<CTIPeriod, '24h' | '7d' | '30d'>>('7d');
  const [selectedSectorId, setSelectedSectorId] = useState<string | null>(null);

  const sectorsQuery = useQuery({
    queryKey: ['cti-sector-threat-overview', period],
    queryFn: () => fetchSectorThreatOverview(period),
  });
  const sectorMetaQuery = useQuery({
    queryKey: ['cti-sector-meta'],
    queryFn: fetchSectors,
  });

  const sectors = sectorsQuery.data?.sectors ?? [];
  const sectorMeta = sectorMetaQuery.data ?? [];
  const sortedSectors = useMemo(
    () => [...sectors].sort((left, right) => right.total_count - left.total_count),
    [sectors],
  );
  const activeSector = useMemo(
    () => sortedSectors.find((sector) => sector.sector_id === selectedSectorId) ?? sortedSectors[0] ?? null,
    [selectedSectorId, sortedSectors],
  );
  const activeSectorMeta = useMemo(
    () => sectorMeta.find((sector) => sector.id === activeSector?.sector_id) ?? null,
    [activeSector?.sector_id, sectorMeta],
  );
  const activeSectorCode = activeSector?.sector_code ?? activeSectorMeta?.code ?? undefined;

  const deepDiveEventsQuery = useQuery({
    queryKey: ['cti-sector-events', activeSectorCode, period],
    queryFn: () => fetchThreatEvents({
      page: 1,
      per_page: 100,
      target_sector: activeSectorCode,
      sort: 'first_seen_at',
      order: 'desc',
    }),
    enabled: Boolean(activeSectorCode),
  });
  const sectorCampaignsQuery = useQuery({
    queryKey: ['cti-sector-campaigns', activeSector?.sector_id],
    queryFn: async () => {
      const response = await fetchCampaigns({ page: 1, per_page: 100, sort: 'last_seen_at', order: 'desc' });
      return response.data.filter((campaign) => campaign.target_sectors.includes(activeSector!.sector_id));
    },
    enabled: Boolean(activeSector?.sector_id),
  });

  const totalEvents = sectors.reduce((sum, sector) => sum + sector.total_count, 0);
  const criticalEvents = sectors.reduce((sum, sector) => sum + sector.severity_critical_count, 0);
  const selectedEvents = deepDiveEventsQuery.data?.data ?? [];
  const selectedCampaigns = sectorCampaignsQuery.data ?? [];

  const threatTypeCounts = useMemo(() => {
    const counts = new Map<string, number>();
    selectedEvents.forEach((event) => {
      const key = event.category_label || event.event_type;
      counts.set(key, (counts.get(key) ?? 0) + 1);
    });
    return Array.from(counts.entries()).sort((a, b) => b[1] - a[1]);
  }, [selectedEvents]);

  const topOrigins = useMemo(() => {
    const counts = new Map<string, number>();
    selectedEvents.forEach((event) => {
      const key = event.origin_country_code?.toUpperCase() || 'Unknown';
      counts.set(key, (counts.get(key) ?? 0) + 1);
    });
    return Array.from(counts.entries()).sort((a, b) => b[1] - a[1]).slice(0, 5);
  }, [selectedEvents]);

  const trendPoints = useMemo(() => {
    const buckets = new Map<string, number>();
    selectedEvents.forEach((event) => {
      const day = event.first_seen_at.slice(0, 10);
      buckets.set(day, (buckets.get(day) ?? 0) + 1);
    });
    return Array.from(buckets.entries())
      .sort((a, b) => a[0].localeCompare(b[0]))
      .slice(-8)
      .map(([, count]) => count);
  }, [selectedEvents]);

  if (sectorsQuery.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState message="Failed to load sector threat overview" onRetry={() => void sectorsQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Sector & Geographic Targeting"
          description="Understand how threats are distributed across industries and pivot into deeper CTI investigations."
          actions={<PeriodSelector value={period} onChange={(nextPeriod) => setPeriod(nextPeriod as Extract<CTIPeriod, '24h' | '7d' | '30d'>)} />}
        />

        <div className="grid gap-4 md:grid-cols-3">
          <CTIKPIStatCard label="Impacted Sectors" value={sortedSectors.length} subtitle={`${period} reporting window`} />
          <CTIKPIStatCard label="Total Sector Events" value={totalEvents} subtitle="Aggregated across all sectors" />
          <CTIKPIStatCard label="Critical Events" value={criticalEvents} subtitle="Critical severity pressure" color="#FF3B5C" />
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Sector Filter Bar</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {sortedSectors.map((sector) => (
              <Button
                key={sector.id}
                variant={activeSector?.sector_id === sector.sector_id ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedSectorId(sector.sector_id)}
              >
                {sector.sector_label}
              </Button>
            ))}
          </CardContent>
        </Card>

        <SectorThreatChart
          sectors={sortedSectors}
          loading={sectorsQuery.isLoading}
          error={undefined}
          onRetry={() => void sectorsQuery.refetch()}
          onSectorClick={setSelectedSectorId}
          selectedSectorId={activeSector?.sector_id}
        />

        {activeSector && (
          <div className="grid gap-4 lg:grid-cols-[1.3fr,0.9fr]">
            <Card>
              <CardHeader>
                <CardTitle>{activeSector.sector_label} Deep Dive</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4 text-sm">
                <div className="grid gap-3 md:grid-cols-4">
                  <MetricPill label="Total" value={activeSector.total_count.toLocaleString()} />
                  <MetricPill label="Critical" value={activeSector.severity_critical_count.toLocaleString()} color={CTI_SEVERITY_COLORS.critical} />
                  <MetricPill label="High" value={activeSector.severity_high_count.toLocaleString()} color={CTI_SEVERITY_COLORS.high} />
                  <MetricPill label="Medium / Low" value={`${activeSector.severity_medium_count}/${activeSector.severity_low_count}`} />
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <div>
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Top Threat Types</p>
                    <div className="flex flex-wrap gap-2">
                      {threatTypeCounts.slice(0, 5).map(([label]) => (
                        <span key={label} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">{label}</span>
                      ))}
                    </div>
                  </div>
                  <div>
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Top Origins</p>
                    <div className="flex flex-wrap gap-2">
                      {topOrigins.map(([country, count]) => (
                        <span key={country} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">{country} ({count})</span>
                      ))}
                    </div>
                  </div>
                </div>

                <div>
                  <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Severity Trend</p>
                  <TrendSparkline points={trendPoints.length > 0 ? trendPoints : [0, 0, 0]} />
                </div>

                <div>
                  <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Recent Events</p>
                  <div className="space-y-3">
                    {selectedEvents.slice(0, 5).map((event) => (
                      <div key={event.id} className="rounded-2xl border p-3">
                        <div className="flex items-center justify-between gap-3">
                          <Link href={`${ROUTES.CYBER_CTI_EVENTS}/${event.id}`} className="font-medium text-foreground hover:underline">
                            {event.title}
                          </Link>
                          <span className="text-xs text-muted-foreground">{formatRelativeTime(event.first_seen_at)}</span>
                        </div>
                        <p className="text-xs text-muted-foreground">{event.origin_city || 'Unknown origin'} · {event.origin_country_code?.toUpperCase() || '—'}</p>
                      </div>
                    ))}
                  </div>
                </div>

                {activeSectorCode && (
                  <Button variant="outline" asChild>
                    <Link href={`${ROUTES.CYBER_CTI_EVENTS}?target_sector=${encodeURIComponent(activeSectorCode)}`}>
                      View All Events for {activeSector.sector_label} →
                    </Link>
                  </Button>
                )}
              </CardContent>
            </Card>

            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>Top Campaigns</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {selectedCampaigns.length > 0 ? selectedCampaigns.slice(0, 5).map((campaign) => (
                    <Link key={campaign.id} href={`${ROUTES.CYBER_CTI_CAMPAIGNS}/${campaign.id}`} className="block rounded-2xl border p-3 hover:bg-muted/20">
                      <p className="font-medium text-foreground">{campaign.name}</p>
                      <p className="text-xs text-muted-foreground">{campaign.actor_name || 'Unknown actor'}</p>
                    </Link>
                  )) : (
                    <p className="text-sm text-muted-foreground">No campaigns explicitly target this sector.</p>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Top Threat Actors</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {Array.from(new Set(selectedCampaigns.map((campaign) => campaign.actor_name).filter(Boolean))).slice(0, 5).map((actorName) => (
                    <p key={actorName} className="rounded-2xl border px-3 py-2 text-sm text-foreground">{actorName}</p>
                  ))}
                  {selectedCampaigns.every((campaign) => !campaign.actor_name) && (
                    <p className="text-sm text-muted-foreground">No attributed actors for this sector yet.</p>
                  )}
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </div>
    </PermissionRedirect>
  );
}

function MetricPill({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="rounded-2xl border px-4 py-3">
      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-semibold" style={color ? { color } : undefined}>{value}</p>
    </div>
  );
}
