'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Globe } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { GlobalThreatMap } from '@/components/cyber/cti/global-threat-map';
import { PeriodSelector } from '@/components/cyber/cti/period-selector';
import { ThreatMapPopover } from '@/components/cyber/cti/threat-map-popover';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  fetchCampaigns,
  fetchGlobalThreatMap,
  fetchThreatActors,
  fetchThreatEvents,
} from '@/lib/cti-api';
import { ROUTES } from '@/lib/constants';
import {
  countryCodeToFlag,
  formatNumber,
  formatRelativeTime,
} from '@/lib/cti-utils';
import { useCTIStore } from '@/stores/cti-store';
import type {
  CTICampaign,
  CTIGeoThreatHotspot,
  CTIPeriod,
  CTIThreatActor,
  CTIThreatEvent,
} from '@/types/cti';

const COMPARISON_PERIOD: Record<Extract<CTIPeriod, '24h' | '7d' | '30d'>, CTIPeriod> = {
  '24h': '7d',
  '7d': '30d',
  '30d': '90d',
};

const PERIOD_DAYS: Record<CTIPeriod, number> = {
  '24h': 1,
  '7d': 7,
  '30d': 30,
  '90d': 90,
};

interface CountryAggregate {
  countryCode: string;
  totalCount: number;
  criticalCount: number;
  highCount: number;
  mediumCount: number;
  lowCount: number;
  topThreatTypes: Array<[string, number]>;
  hotspots: CTIGeoThreatHotspot[];
  trendPercentage: number;
  trendDirection: 'increasing' | 'stable' | 'decreasing';
}

function countryName(code: string): string {
  try {
    return new Intl.DisplayNames(['en'], { type: 'region' }).of(code.toUpperCase()) ?? code.toUpperCase();
  } catch {
    return code.toUpperCase();
  }
}

function aggregateByCountry(
  hotspots: CTIGeoThreatHotspot[],
  baselineHotspots: CTIGeoThreatHotspot[],
  period: Extract<CTIPeriod, '24h' | '7d' | '30d'>,
): CountryAggregate[] {
  const baselineMap = new Map<string, number>();
  baselineHotspots.forEach((hotspot) => {
    const key = hotspot.country_code.toUpperCase();
    baselineMap.set(key, (baselineMap.get(key) ?? 0) + hotspot.total_count);
  });

  const aggregates = new Map<string, CountryAggregate>();
  hotspots.forEach((hotspot) => {
    const key = hotspot.country_code.toUpperCase();
    const current = aggregates.get(key) ?? {
      countryCode: key,
      totalCount: 0,
      criticalCount: 0,
      highCount: 0,
      mediumCount: 0,
      lowCount: 0,
      topThreatTypes: [],
      hotspots: [],
      trendPercentage: 0,
      trendDirection: 'stable' as const,
    };

    current.totalCount += hotspot.total_count;
    current.criticalCount += hotspot.severity_critical_count;
    current.highCount += hotspot.severity_high_count;
    current.mediumCount += hotspot.severity_medium_count;
    current.lowCount += hotspot.severity_low_count;
    current.hotspots.push(hotspot);
    aggregates.set(key, current);
  });

  return Array.from(aggregates.values())
    .map((aggregate) => {
      const threatTypeCounts = new Map<string, number>();
      aggregate.hotspots.forEach((hotspot) => {
        const label = hotspot.top_threat_type || 'Unknown';
        threatTypeCounts.set(label, (threatTypeCounts.get(label) ?? 0) + hotspot.total_count);
      });

      const currentRate = aggregate.totalCount / PERIOD_DAYS[period];
      const baselineRate = (baselineMap.get(aggregate.countryCode) ?? 0) / PERIOD_DAYS[COMPARISON_PERIOD[period]];
      const trendPercentage = baselineRate > 0
        ? Number((((currentRate - baselineRate) / baselineRate) * 100).toFixed(1))
        : aggregate.totalCount > 0
          ? 100
          : 0;

      return {
        ...aggregate,
        topThreatTypes: Array.from(threatTypeCounts.entries())
          .sort((left, right) => right[1] - left[1])
          .slice(0, 5),
        trendPercentage,
        trendDirection:
          trendPercentage > 5 ? 'increasing' : trendPercentage < -5 ? 'decreasing' : 'stable',
      };
    })
    .sort((left, right) => right.totalCount - left.totalCount);
}

function buildCountryDetail(
  events: CTIThreatEvent[],
  actors: CTIThreatActor[],
  campaigns: CTICampaign[],
) {
  const threatTypeCounts = new Map<string, number>();
  const sectorCounts = new Map<string, number>();

  events.forEach((event) => {
    const threatType = event.category_label || event.event_type;
    threatTypeCounts.set(threatType, (threatTypeCounts.get(threatType) ?? 0) + 1);

    if (event.target_sector_label) {
      sectorCounts.set(event.target_sector_label, (sectorCounts.get(event.target_sector_label) ?? 0) + 1);
    }
  });

  return {
    topThreatTypes: Array.from(threatTypeCounts.entries()).sort((left, right) => right[1] - left[1]).slice(0, 5),
    topSectors: Array.from(sectorCounts.entries()).sort((left, right) => right[1] - left[1]).slice(0, 5),
    recentEvents: events.slice(0, 6),
    activeActors: actors.slice(0, 5),
    activeCampaigns: campaigns.slice(0, 5),
  };
}

export default function CTIGeoAnalysisPage() {
  const router = useRouter();
  const { liveEvents } = useCTIStore();
  const [period, setPeriod] = useState<Extract<CTIPeriod, '24h' | '7d' | '30d'>>('24h');
  const [selectedCountryCode, setSelectedCountryCode] = useState<string | null>(null);
  const [selectedHotspot, setSelectedHotspot] = useState<CTIGeoThreatHotspot | null>(null);

  const geoQuery = useQuery({
    queryKey: ['cti-geo-analysis', period],
    queryFn: () => fetchGlobalThreatMap(period),
    staleTime: 60_000,
  });
  const baselineGeoQuery = useQuery({
    queryKey: ['cti-geo-analysis-baseline', COMPARISON_PERIOD[period]],
    queryFn: () => fetchGlobalThreatMap(COMPARISON_PERIOD[period]),
    staleTime: 60_000,
  });

  const hotspots = geoQuery.data?.hotspots ?? [];
  const countryRows = useMemo(
    () => aggregateByCountry(hotspots, baselineGeoQuery.data?.hotspots ?? [], period),
    [baselineGeoQuery.data?.hotspots, hotspots, period],
  );

  useEffect(() => {
    if (!countryRows.length) {
      setSelectedCountryCode(null);
      setSelectedHotspot(null);
      return;
    }

    if (!selectedCountryCode || !countryRows.some((row) => row.countryCode === selectedCountryCode)) {
      const nextCountry = countryRows[0];
      setSelectedCountryCode(nextCountry.countryCode);
      setSelectedHotspot(nextCountry.hotspots[0] ?? null);
    }
  }, [countryRows, selectedCountryCode]);

  const selectedCountry = useMemo(
    () => countryRows.find((country) => country.countryCode === selectedCountryCode) ?? null,
    [countryRows, selectedCountryCode],
  );

  const countryEventsQuery = useQuery({
    queryKey: ['cti-geo-country-events', selectedCountryCode, period],
    queryFn: () => fetchThreatEvents({
      page: 1,
      per_page: 100,
      origin_country: selectedCountryCode?.toLowerCase(),
      sort: 'first_seen_at',
      order: 'desc',
    }),
    enabled: Boolean(selectedCountryCode),
    staleTime: 30_000,
  });
  const countryActorsQuery = useQuery({
    queryKey: ['cti-geo-country-actors', selectedCountryCode],
    queryFn: async () => {
      const response = await fetchThreatActors({
        page: 1,
        per_page: 100,
        sort: 'risk_score',
        order: 'desc',
        is_active: true,
      });
      return response.data.filter((actor) => actor.origin_country_code?.toUpperCase() === selectedCountryCode);
    },
    enabled: Boolean(selectedCountryCode),
    staleTime: 60_000,
  });
  const countryCampaignsQuery = useQuery({
    queryKey: ['cti-geo-country-campaigns', selectedCountryCode],
    queryFn: async () => {
      const actorIds = new Set((countryActorsQuery.data ?? []).map((actor) => actor.id));
      const response = await fetchCampaigns({
        page: 1,
        per_page: 100,
        sort: 'last_seen_at',
        order: 'desc',
        status: ['active', 'monitoring'],
      });

      return response.data.filter((campaign) => (
        campaign.primary_actor_id ? actorIds.has(campaign.primary_actor_id) : false
      ));
    },
    enabled: Boolean(selectedCountryCode) && countryActorsQuery.isSuccess,
    staleTime: 60_000,
  });

  const selectedCountryDetail = useMemo(
    () => buildCountryDetail(
      countryEventsQuery.data?.data ?? [],
      countryActorsQuery.data ?? [],
      countryCampaignsQuery.data ?? [],
    ),
    [countryActorsQuery.data, countryCampaignsQuery.data, countryEventsQuery.data?.data],
  );

  const totalCountries = countryRows.length;
  const totalEvents = countryRows.reduce((sum, row) => sum + row.totalCount, 0);
  const averagePressure = totalCountries > 0 ? Math.round(totalEvents / totalCountries) : 0;

  const handleViewEvents = useCallback(
    (countryCode: string, city?: string) => {
      const params = new URLSearchParams();
      params.set('origin_country', countryCode.toLowerCase());
      if (city) {
        params.set('search', city);
      }
      router.push(`${ROUTES.CYBER_CTI_EVENTS}?${params.toString()}`);
    },
    [router],
  );

  const handleSelectCountry = useCallback((countryCode: string) => {
    setSelectedCountryCode(countryCode);
    const country = countryRows.find((row) => row.countryCode === countryCode);
    setSelectedHotspot(country?.hotspots[0] ?? null);
  }, [countryRows]);

  if (geoQuery.isLoading || baselineGeoQuery.isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <LoadingSkeleton variant="chart" />
      </PermissionRedirect>
    );
  }

  if (geoQuery.error || baselineGeoQuery.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState
          message="Failed to load geographic CTI analysis."
          onRetry={() => {
            void geoQuery.refetch();
            void baselineGeoQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Geographic Threat Analysis"
          description="Review country-level threat pressure, investigate hotspots, and pivot directly into country-scoped CTI events."
          actions={<PeriodSelector value={period} onChange={(value) => setPeriod(value as typeof period)} />}
        />

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <CTIKPIStatCard label="Countries Observed" value={totalCountries} subtitle={`${period} coverage window`} />
          <CTIKPIStatCard label="Hotspot Events" value={totalEvents} subtitle="Aggregated from geo summaries" />
          <CTIKPIStatCard
            label="Top Origin"
            value={selectedCountry ? countryName(selectedCountry.countryCode) : '—'}
            subtitle={selectedCountry ? `${formatNumber(selectedCountry.totalCount)} events` : 'No country selected'}
          />
          <CTIKPIStatCard label="Avg Pressure" value={averagePressure} subtitle="Events per active country" />
        </div>

        <div className="relative">
          <GlobalThreatMap
            hotspots={hotspots}
            period={period}
            onPeriodChange={(value) => setPeriod(value as typeof period)}
            onHotspotClick={(hotspot) => {
              setSelectedHotspot(hotspot);
              setSelectedCountryCode(hotspot.country_code.toUpperCase());
            }}
            selectedHotspot={selectedHotspot}
            liveEvents={liveEvents}
            className="min-h-[520px]"
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

        <div className="grid gap-4 xl:grid-cols-[1.15fr,0.85fr]">
          <Card>
            <CardHeader>
              <CardTitle>Country Rankings</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {countryRows.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Rank</TableHead>
                      <TableHead>Country</TableHead>
                      <TableHead className="text-right">Events</TableHead>
                      <TableHead className="text-right">Critical</TableHead>
                      <TableHead className="text-right">High</TableHead>
                      <TableHead className="text-right">Trend</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {countryRows.map((country, index) => (
                      <TableRow
                        key={country.countryCode}
                        className={selectedCountryCode === country.countryCode ? 'bg-muted/30' : 'cursor-pointer'}
                        onClick={() => handleSelectCountry(country.countryCode)}
                      >
                        <TableCell className="font-medium">{index + 1}</TableCell>
                        <TableCell>
                          <div className="flex items-center gap-3">
                            <span className="text-lg">{countryCodeToFlag(country.countryCode)}</span>
                            <div>
                              <p className="font-medium">{countryName(country.countryCode)}</p>
                              <p className="text-xs text-muted-foreground">{country.countryCode}</p>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell className="text-right font-medium tabular-nums">
                          {formatNumber(country.totalCount)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {formatNumber(country.criticalCount)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {formatNumber(country.highCount)}
                        </TableCell>
                        <TableCell className="text-right">
                          <span
                            className={
                              country.trendDirection === 'increasing'
                                ? 'text-rose-400'
                                : country.trendDirection === 'decreasing'
                                  ? 'text-emerald-400'
                                  : 'text-muted-foreground'
                            }
                          >
                            {country.trendDirection === 'increasing' ? '↑' : country.trendDirection === 'decreasing' ? '↓' : '→'}{' '}
                            {Math.abs(country.trendPercentage).toFixed(1)}%
                          </span>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className="px-4 py-10 text-center text-sm text-muted-foreground">
                  <Globe className="mx-auto mb-2 h-5 w-5" />
                  No country-level hotspot data is available for this period.
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                {selectedCountry ? `${countryName(selectedCountry.countryCode)} (${countryCodeToFlag(selectedCountry.countryCode)})` : 'Country Detail'}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {selectedCountry ? (
                <>
                  <div className="grid gap-3 md:grid-cols-3">
                    <Metric label="Total Events" value={formatNumber(selectedCountry.totalCount)} />
                    <Metric label="Critical" value={formatNumber(selectedCountry.criticalCount)} tone="#FF3B5C" />
                    <Metric label="Active Campaigns" value={formatNumber(selectedCountryDetail.activeCampaigns.length)} />
                  </div>

                  <div>
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                      Top Threat Types
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {selectedCountryDetail.topThreatTypes.length > 0 ? selectedCountryDetail.topThreatTypes.map(([label, count]) => (
                        <span key={label} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">
                          {label} ({count})
                        </span>
                      )) : (
                        <span className="text-sm text-muted-foreground">No threat-type breakdown available.</span>
                      )}
                    </div>
                  </div>

                  <div>
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                      Top Targeted Sectors
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {selectedCountryDetail.topSectors.length > 0 ? selectedCountryDetail.topSectors.map(([label, count]) => (
                        <span key={label} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">
                          {label} ({count})
                        </span>
                      )) : (
                        <span className="text-sm text-muted-foreground">No sector targeting metadata available.</span>
                      )}
                    </div>
                  </div>

                  <div className="grid gap-4 lg:grid-cols-2">
                    <div>
                      <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                        Active Actors
                      </p>
                      <div className="space-y-2">
                        {selectedCountryDetail.activeActors.length > 0 ? selectedCountryDetail.activeActors.map((actor) => (
                          <Link
                            key={actor.id}
                            href={`${ROUTES.CYBER_CTI_ACTORS}/${actor.id}`}
                            className="block rounded-2xl border px-3 py-2 hover:bg-muted/20"
                          >
                            <div className="flex items-center justify-between gap-3">
                              <div>
                                <p className="font-medium text-foreground">{actor.name}</p>
                                <p className="text-xs text-muted-foreground">{actor.sophistication_level} · {actor.primary_motivation}</p>
                              </div>
                              <CTIStatusBadge status={actor.is_active ? 'active' : 'archived'} type="campaign" />
                            </div>
                          </Link>
                        )) : (
                          <p className="text-sm text-muted-foreground">No active actors attributed to this country.</p>
                        )}
                      </div>
                    </div>

                    <div>
                      <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                        Active Campaigns
                      </p>
                      <div className="space-y-2">
                        {selectedCountryDetail.activeCampaigns.length > 0 ? selectedCountryDetail.activeCampaigns.map((campaign) => (
                          <Link
                            key={campaign.id}
                            href={`${ROUTES.CYBER_CTI_CAMPAIGNS}/${campaign.id}`}
                            className="block rounded-2xl border px-3 py-2 hover:bg-muted/20"
                          >
                            <div className="flex items-center justify-between gap-3">
                              <div>
                                <p className="font-medium text-foreground">{campaign.name}</p>
                                <p className="text-xs text-muted-foreground">{campaign.actor_name || 'Unknown actor'}</p>
                              </div>
                              <CTISeverityBadge severity={campaign.severity_code} size="sm" />
                            </div>
                          </Link>
                        )) : (
                          <p className="text-sm text-muted-foreground">No active campaigns mapped to actors from this country.</p>
                        )}
                      </div>
                    </div>
                  </div>

                  <div>
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                      Recent Events
                    </p>
                    <div className="space-y-2">
                      {selectedCountryDetail.recentEvents.length > 0 ? selectedCountryDetail.recentEvents.map((event) => (
                        <Link
                          key={event.id}
                          href={`${ROUTES.CYBER_CTI_EVENTS}/${event.id}`}
                          className="flex items-center justify-between gap-3 rounded-2xl border px-3 py-2 hover:bg-muted/20"
                        >
                          <div className="min-w-0">
                            <p className="truncate font-medium text-foreground">{event.title}</p>
                            <p className="text-xs text-muted-foreground">{event.origin_city || 'Unknown city'} · {formatRelativeTime(event.first_seen_at)}</p>
                          </div>
                          <CTISeverityBadge severity={event.severity_code} size="sm" />
                        </Link>
                      )) : (
                        <p className="text-sm text-muted-foreground">No recent events found for this country.</p>
                      )}
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-2">
                    <Link
                      href={`${ROUTES.CYBER_CTI_EVENTS}?origin_country=${encodeURIComponent(selectedCountry.countryCode.toLowerCase())}`}
                      className="inline-flex h-9 items-center rounded-md border px-4 text-sm font-medium hover:bg-muted/20"
                    >
                      View All Events from {countryName(selectedCountry.countryCode)} →
                    </Link>
                  </div>
                </>
              ) : (
                <p className="text-sm text-muted-foreground">Select a country to inspect hotspot composition and attributed CTI activity.</p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </PermissionRedirect>
  );
}

function Metric({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="rounded-2xl border px-4 py-3">
      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-semibold" style={tone ? { color: tone } : undefined}>
        {value}
      </p>
    </div>
  );
}
