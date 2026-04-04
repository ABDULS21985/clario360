'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { PeriodSelector } from '@/components/cyber/cti/period-selector';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { SectorThreatChart } from '@/components/cyber/cti/sector-threat-chart';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { fetchCampaigns, fetchSectorThreatOverview, fetchSectors, fetchThreatEvents } from '@/lib/cti-api';
import { ROUTES } from '@/lib/constants';
import { formatRelativeTime } from '@/lib/cti-utils';
import { CTI_SEVERITY_COLORS, type CTIPeriod, type CTISectorThreatSummary } from '@/types/cti';

type SectorLens = CTISectorThreatSummary & { snapshot_count: number };

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

function sectorTimestamp(sector: CTISectorThreatSummary): number {
  const parsed = Date.parse(sector.computed_at ?? sector.period_end ?? sector.period_start);
  return Number.isNaN(parsed) ? 0 : parsed;
}

function normalizeSectorSummaries(sectors: CTISectorThreatSummary[]): SectorLens[] {
  const latestBySector = new Map<string, SectorLens>();

  sectors.forEach((sector) => {
    const existing = latestBySector.get(sector.sector_id);
    if (!existing) {
      latestBySector.set(sector.sector_id, { ...sector, snapshot_count: 1 });
      return;
    }

    const nextSnapshotCount = existing.snapshot_count + 1;
    if (sectorTimestamp(sector) >= sectorTimestamp(existing)) {
      latestBySector.set(sector.sector_id, { ...sector, snapshot_count: nextSnapshotCount });
      return;
    }

    latestBySector.set(sector.sector_id, { ...existing, snapshot_count: nextSnapshotCount });
  });

  return Array.from(latestBySector.values()).sort((left, right) => right.total_count - left.total_count);
}

function percentOfTotal(value: number, total: number): number {
  if (total <= 0) {
    return 0;
  }
  return Number(((value / total) * 100).toFixed(1));
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

  const sectors = useMemo(() => sectorsQuery.data?.sectors ?? [], [sectorsQuery.data?.sectors]);
  const sectorMeta = useMemo(() => sectorMetaQuery.data ?? [], [sectorMetaQuery.data]);
  const normalizedSectors = useMemo(
    () => normalizeSectorSummaries(sectors),
    [sectors],
  );
  const activeSector = useMemo(
    () => normalizedSectors.find((sector) => sector.sector_id === selectedSectorId) ?? normalizedSectors[0] ?? null,
    [selectedSectorId, normalizedSectors],
  );
  const activeSectorMeta = useMemo(
    () => sectorMeta.find((sector) => sector.id === activeSector?.sector_id) ?? null,
    [activeSector?.sector_id, sectorMeta],
  );
  const activeSectorCode = activeSector?.sector_code ?? activeSectorMeta?.code ?? undefined;
  const selectorSectors = useMemo(() => {
    if (!activeSector) {
      return normalizedSectors.slice(0, 12);
    }

    const featured = normalizedSectors.slice(0, 12).filter((sector) => sector.sector_id !== activeSector.sector_id);
    return [activeSector, ...featured].slice(0, 12);
  }, [activeSector, normalizedSectors]);

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

  const totalEvents = normalizedSectors.reduce((sum, sector) => sum + sector.total_count, 0);
  const criticalEvents = normalizedSectors.reduce((sum, sector) => sum + sector.severity_critical_count, 0);
  const selectedEvents = useMemo(() => deepDiveEventsQuery.data?.data ?? [], [deepDiveEventsQuery.data?.data]);
  const selectedCampaigns = useMemo(() => sectorCampaignsQuery.data ?? [], [sectorCampaignsQuery.data]);
  const normalizedSnapshotCount = sectors.length;
  const selectedShare = percentOfTotal(activeSector?.total_count ?? 0, totalEvents);

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
          <CTIKPIStatCard
            label="Impacted Sectors"
            value={normalizedSectors.length}
            subtitle={`${period} reporting window`}
            className="border-emerald-200/70 bg-white/85 shadow-sm"
          />
          <CTIKPIStatCard label="Total Sector Events" value={totalEvents} subtitle="Aggregated across all sectors" />
          <CTIKPIStatCard
            label="Critical Events"
            value={criticalEvents}
            subtitle="Critical severity pressure"
            color="#FF3B5C"
            className="border-rose-200/80 bg-rose-50/50 shadow-sm"
          />
        </div>

        <Card className="overflow-hidden border-emerald-200/60 bg-[radial-gradient(circle_at_top_right,rgba(45,212,191,0.18),transparent_30%),linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.96))] shadow-sm">
          <CardContent className="grid gap-6 p-6 lg:grid-cols-[1.2fr,0.8fr]">
            <div className="space-y-4">
              <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700">
                Sector Intelligence Lens
              </Badge>
              <div className="space-y-2">
                <h2 className="text-2xl font-semibold tracking-tight text-foreground">
                  Focus the industries carrying the highest threat pressure.
                </h2>
                <p className="max-w-2xl text-sm text-muted-foreground">
                  The selector below now normalizes repeated aggregation snapshots into a clean set of sector lenses so you can pivot quickly without the page turning into noise.
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Badge variant="secondary" className="bg-slate-950/[0.03] text-slate-700">
                  {normalizedSectors.length} unique sectors
                </Badge>
                <Badge variant="secondary" className="bg-slate-950/[0.03] text-slate-700">
                  {normalizedSnapshotCount.toLocaleString()} summary points analyzed
                </Badge>
                {activeSector && (
                  <Badge variant="secondary" className="bg-emerald-100 text-emerald-800">
                    {activeSector.sector_label} holds {selectedShare}% of pressure
                  </Badge>
                )}
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
              <SignalMetric
                label="Primary focus"
                value={activeSector?.sector_label ?? 'None'}
                subtitle={activeSector ? `${activeSector.total_count.toLocaleString()} events` : 'Select a sector'}
              />
              <SignalMetric
                label="Snapshot freshness"
                value={activeSector?.computed_at ? formatRelativeTime(activeSector.computed_at) : 'Live'}
                subtitle="Latest sector aggregation"
              />
              <SignalMetric
                label="Critical mix"
                value={`${percentOfTotal(criticalEvents, totalEvents)}%`}
                subtitle="Of all displayed sector events"
                accent="#FF3B5C"
              />
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/70 bg-white/85 shadow-sm">
          <CardHeader className="pb-3">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <CardTitle className="text-base">Sector Navigator</CardTitle>
                <p className="mt-1 text-sm text-muted-foreground">
                  A tighter view of the highest-pressure sectors. Scroll horizontally to pivot the deep dive.
                </p>
              </div>
              {activeSector && (
                <div className="rounded-2xl border border-emerald-200 bg-emerald-50 px-3 py-2 text-right">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-emerald-700">Selected</p>
                  <p className="text-sm font-medium text-foreground">{activeSector.sector_label}</p>
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            <ScrollArea className="w-full whitespace-nowrap">
              <div className="flex gap-3 pb-3">
                {selectorSectors.map((sector) => {
                  const isActive = activeSector?.sector_id === sector.sector_id;
                  return (
                    <Button
                      key={sector.sector_id}
                      variant={isActive ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setSelectedSectorId(sector.sector_id)}
                      className={[
                        'h-auto min-w-[13rem] justify-between rounded-2xl px-4 py-3 text-left',
                        isActive
                          ? 'border-emerald-500 bg-emerald-700 text-white shadow-[0_22px_48px_-24px_rgba(5,150,105,0.8)]'
                          : 'border-border/80 bg-white/90 hover:border-emerald-200 hover:bg-emerald-50/60',
                      ].join(' ')}
                    >
                      <span className="flex flex-col items-start gap-1">
                        <span className="truncate text-sm font-semibold normal-case">{sector.sector_label}</span>
                        <span className={`text-[11px] ${isActive ? 'text-emerald-50/90' : 'text-muted-foreground'}`}>
                          {sector.total_count.toLocaleString()} events
                        </span>
                      </span>
                      <span className={`rounded-full px-2 py-1 text-[11px] font-semibold ${isActive ? 'bg-white/15 text-white' : 'bg-slate-950/[0.04] text-slate-700'}`}>
                        {sector.snapshot_count}x
                      </span>
                    </Button>
                  );
                })}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>

        <SectorThreatChart
          sectors={normalizedSectors}
          loading={sectorsQuery.isLoading}
          error={undefined}
          onRetry={() => void sectorsQuery.refetch()}
          onSectorClick={setSelectedSectorId}
          selectedSectorId={activeSector?.sector_id}
        />

        {activeSector && (
          <div className="grid gap-4 lg:grid-cols-[1.3fr,0.9fr]">
            <Card className="overflow-hidden border-border/70 bg-white/85 shadow-sm">
              <CardHeader className="border-b border-border/60 bg-slate-950/[0.02]">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="space-y-2">
                    <Badge variant="outline" className="border-slate-200 bg-white text-slate-700">
                      Deep Dive
                    </Badge>
                    <CardTitle className="text-2xl tracking-tight">{activeSector.sector_label}</CardTitle>
                    <p className="text-sm text-muted-foreground">
                      Latest aggregation {formatRelativeTime(activeSector.computed_at ?? activeSector.period_end)}
                    </p>
                  </div>
                  <div className="rounded-3xl border border-border/70 bg-white/80 px-4 py-3 text-right shadow-sm">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Sector share</p>
                    <p className="mt-1 text-2xl font-semibold text-foreground">{selectedShare}%</p>
                    <p className="text-xs text-muted-foreground">of displayed sector activity</p>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-6 p-6 text-sm">
                <div className="grid gap-3 md:grid-cols-4">
                  <MetricPill label="Total" value={activeSector.total_count.toLocaleString()} subtitle="Events attributed to this sector" />
                  <MetricPill label="Critical" value={activeSector.severity_critical_count.toLocaleString()} color={CTI_SEVERITY_COLORS.critical} subtitle="Highest-severity pressure" />
                  <MetricPill label="High" value={activeSector.severity_high_count.toLocaleString()} color={CTI_SEVERITY_COLORS.high} subtitle="Active campaign overlap" />
                  <MetricPill label="Medium / Low" value={`${activeSector.severity_medium_count}/${activeSector.severity_low_count}`} subtitle="Broader background activity" />
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="rounded-[26px] border border-border/70 bg-white/70 p-4 shadow-sm">
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Top Threat Types</p>
                    {threatTypeCounts.length > 0 ? (
                      <div className="flex flex-wrap gap-2">
                        {threatTypeCounts.slice(0, 5).map(([label, count]) => (
                          <Badge key={label} variant="outline" className="border-slate-200 bg-white text-slate-700 normal-case">
                            {label} · {count}
                          </Badge>
                        ))}
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">No recent threat taxonomy available for this sector.</p>
                    )}
                  </div>
                  <div className="rounded-[26px] border border-border/70 bg-white/70 p-4 shadow-sm">
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Top Origins</p>
                    {topOrigins.length > 0 ? (
                      <div className="flex flex-wrap gap-2">
                        {topOrigins.map(([country, count]) => (
                          <Badge key={country} variant="outline" className="border-slate-200 bg-white text-slate-700 normal-case">
                            {country} ({count})
                          </Badge>
                        ))}
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">No geographic origin signal is available yet.</p>
                    )}
                  </div>
                </div>

                <div className="rounded-[28px] border border-border/70 bg-slate-950/[0.02] p-4 shadow-sm">
                  <div className="mb-3 flex items-center justify-between gap-3">
                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Severity Trend</p>
                    <span className="text-xs text-muted-foreground">Recent event cadence for this sector</span>
                  </div>
                  <TrendSparkline points={trendPoints.length > 0 ? trendPoints : [0, 0, 0]} />
                </div>

                <div className="space-y-3">
                  <div className="flex items-center justify-between gap-3">
                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Recent Events</p>
                    <span className="text-xs text-muted-foreground">{selectedEvents.length.toLocaleString()} matched events</span>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    {selectedEvents.slice(0, 6).map((event) => (
                      <div key={event.id} className="rounded-[24px] border border-border/70 bg-white/80 p-4 shadow-sm">
                        <div className="flex items-center justify-between gap-3">
                          <Link href={`${ROUTES.CYBER_CTI_EVENTS}/${event.id}`} className="font-medium text-foreground hover:underline">
                            {event.title}
                          </Link>
                          <span className="text-xs text-muted-foreground">{formatRelativeTime(event.first_seen_at)}</span>
                        </div>
                        <p className="mt-1 text-xs text-muted-foreground">{event.origin_city || 'Unknown origin'} · {event.origin_country_code?.toUpperCase() || '—'}</p>
                      </div>
                    ))}
                    {selectedEvents.length === 0 && (
                      <div className="rounded-[24px] border border-dashed border-border/70 bg-white/70 p-4 text-sm text-muted-foreground md:col-span-2">
                        No recent events are mapped to this sector for the selected period.
                      </div>
                    )}
                  </div>
                </div>

                {activeSectorCode && (
                  <Button variant="outline" className="w-full justify-between rounded-2xl sm:w-auto" asChild>
                    <Link href={`${ROUTES.CYBER_CTI_EVENTS}?target_sector=${encodeURIComponent(activeSectorCode)}`}>
                      View All Events for {activeSector.sector_label} →
                    </Link>
                  </Button>
                )}
              </CardContent>
            </Card>

            <div className="space-y-4">
              <Card className="border-border/70 bg-white/85 shadow-sm">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between gap-3">
                    <CardTitle className="text-base">Campaign Pressure</CardTitle>
                    <Badge variant="outline" className="border-orange-200 bg-orange-50 text-orange-700">
                      {selectedCampaigns.length} linked
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {selectedCampaigns.length > 0 ? selectedCampaigns.slice(0, 5).map((campaign) => (
                    <Link key={campaign.id} href={`${ROUTES.CYBER_CTI_CAMPAIGNS}/${campaign.id}`} className="block rounded-[24px] border border-border/70 bg-white/80 p-4 shadow-sm transition hover:-translate-y-0.5 hover:border-orange-200 hover:bg-orange-50/40">
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-1">
                          <p className="font-medium text-foreground">{campaign.name}</p>
                          <p className="text-xs text-muted-foreground">{campaign.actor_name || 'Unknown actor'}</p>
                        </div>
                        <CTISeverityBadge severity={campaign.severity_code} size="sm" />
                      </div>
                      <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
                        <span>{campaign.ioc_count.toLocaleString()} IOCs</span>
                        <span>{campaign.event_count.toLocaleString()} events</span>
                      </div>
                    </Link>
                  )) : (
                    <p className="rounded-[24px] border border-dashed border-border/70 bg-white/70 px-4 py-5 text-sm text-muted-foreground">
                      No campaigns explicitly target this sector.
                    </p>
                  )}
                </CardContent>
              </Card>

              <Card className="border-border/70 bg-white/85 shadow-sm">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between gap-3">
                    <CardTitle className="text-base">Attribution Lane</CardTitle>
                    <Badge variant="outline" className="border-sky-200 bg-sky-50 text-sky-700">
                      {Array.from(new Set(selectedCampaigns.map((campaign) => campaign.actor_name).filter(Boolean))).length} actors
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {Array.from(new Set(selectedCampaigns.map((campaign) => campaign.actor_name).filter(Boolean))).slice(0, 5).map((actorName) => (
                    <p key={actorName} className="rounded-[20px] border border-border/70 bg-white/80 px-4 py-3 text-sm font-medium text-foreground shadow-sm">
                      {actorName}
                    </p>
                  ))}
                  {selectedCampaigns.every((campaign) => !campaign.actor_name) && (
                    <p className="rounded-[24px] border border-dashed border-border/70 bg-white/70 px-4 py-5 text-sm text-muted-foreground">
                      No attributed actors for this sector yet.
                    </p>
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

function MetricPill({
  label,
  value,
  color,
  subtitle,
}: {
  label: string;
  value: string;
  color?: string;
  subtitle?: string;
}) {
  return (
    <div className="rounded-[24px] border border-border/70 bg-white/80 px-4 py-4 shadow-sm">
      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold tracking-tight" style={color ? { color } : undefined}>{value}</p>
      {subtitle && <p className="mt-1 text-xs text-muted-foreground">{subtitle}</p>}
    </div>
  );
}

function SignalMetric({
  label,
  value,
  subtitle,
  accent,
}: {
  label: string;
  value: string;
  subtitle: string;
  accent?: string;
}) {
  return (
    <div className="rounded-[24px] border border-white/70 bg-white/80 px-4 py-4 shadow-sm backdrop-blur">
      <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold tracking-tight" style={accent ? { color: accent } : undefined}>
        {value}
      </p>
      <p className="mt-1 text-xs text-muted-foreground">{subtitle}</p>
    </div>
  );
}
