'use client';

import { useCallback, useMemo } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import {
  Activity,
  Fingerprint,
  Radar,
  RefreshCw,
  ShieldAlert,
  Target,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useCTIDashboard } from '@/hooks/use-cti-dashboard';
import { useCTIWebSocket } from '@/hooks/use-cti-websocket';
import { useCTIStore } from '@/stores/cti-store';
import { GlobalThreatMap } from '@/components/cyber/cti/global-threat-map';
import { ThreatMapPopover } from '@/components/cyber/cti/threat-map-popover';
import { LiveEventFeed } from '@/components/cyber/cti/live-event-feed';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { RiskScoreGauge } from '@/components/cyber/cti/risk-score-gauge';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';

function websocketTone(status: string): string {
  switch (status) {
    case 'connected':
      return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30';
    case 'connecting':
      return 'bg-amber-500/15 text-amber-300 border-amber-500/30';
    case 'error':
      return 'bg-rose-500/15 text-rose-300 border-rose-500/30';
    default:
      return 'bg-slate-500/15 text-slate-300 border-slate-500/30';
  }
}

export default function CTIDashboardPage() {
  const router = useRouter();
  const {
    period,
    setPeriod: rawSetPeriod,
    snapshot,
    hotspots,
    sectors,
    topCampaigns,
    criticalBrands,
    recentEvents,
    isLoading,
  } = useCTIDashboard();
  const setPeriod = rawSetPeriod as (period: string) => void;
  const { selectedHotspot, setSelectedHotspot, liveEvents, loadDashboard } = useCTIStore();
  const { status: wsStatus } = useCTIWebSocket();

  const mttd = snapshot?.mean_time_to_detect_hours ?? 0;
  const mttr = snapshot?.mean_time_to_respond_hours ?? 0;
  const topSector = snapshot?.top_targeted_sector_label ?? 'Unavailable';
  const recentCampaigns = useMemo(() => topCampaigns.slice(0, 5), [topCampaigns]);
  const urgentBrands = useMemo(() => criticalBrands.slice(0, 5), [criticalBrands]);

  const handleRefresh = useCallback(() => {
    void loadDashboard();
  }, [loadDashboard]);

  const handleViewEvents = useCallback(
    (countryCode: string, city: string) => {
      const params = new URLSearchParams();
      if (countryCode) {
        params.set('origin_country', countryCode.toLowerCase());
      }
      if (city) {
        params.set('search', city);
      }
      router.push(`/cyber/cti/events?${params.toString()}`);
    },
    [router],
  );

  if (isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <div className="space-y-6">
          <PageHeader
            title="Cyber Threat Intelligence"
            description="Cyber Threat Intelligence Command Center"
          />
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <LoadingSkeleton key={index} variant="card" />
            ))}
          </div>
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
            <LoadingSkeleton variant="chart" className="xl:col-span-2" />
            <LoadingSkeleton variant="card" />
          </div>
        </div>
      </PermissionRedirect>
    );
  }

  if (!snapshot) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState message="Failed to load CTI dashboard." onRetry={handleRefresh} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Cyber Threat Intelligence"
          description="Cyber Threat Intelligence Command Center"
          actions={(
            <div className="flex items-center gap-2">
              <span className={`rounded-full border px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em] ${websocketTone(wsStatus)}`}>
                WS {wsStatus}
              </span>
              <Button variant="outline" size="sm" onClick={handleRefresh}>
                <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                Refresh
              </Button>
            </div>
          )}
        />

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <CTIKPIStatCard
            label="Events 24h"
            value={snapshot.total_events_24h}
            subtitle={`${snapshot.total_events_7d.toLocaleString()} in 7d`}
            trend={{ direction: snapshot.trend_direction, percentage: snapshot.trend_percentage }}
            icon={<Activity className="h-4 w-4" />}
            color="#FF3B5C"
          />
          <CTIKPIStatCard
            label="Active Campaigns"
            value={snapshot.active_campaigns_count}
            subtitle={`${snapshot.critical_campaigns_count} critical`}
            icon={<Target className="h-4 w-4" />}
            color="#FF8C42"
            onClick={() => router.push('/cyber/cti/campaigns')}
          />
          <CTIKPIStatCard
            label="Total IOCs"
            value={snapshot.total_iocs}
            subtitle={`Top sector: ${topSector}`}
            icon={<Fingerprint className="h-4 w-4" />}
            color="#0EA5E9"
          />
          <CTIKPIStatCard
            label="Brand Abuse Alerts"
            value={snapshot.brand_abuse_total_count}
            subtitle={`${snapshot.brand_abuse_critical_count} critical`}
            icon={<ShieldAlert className="h-4 w-4" />}
            color="#F97316"
            onClick={() => router.push('/cyber/cti/brand-abuse')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <div className="relative xl:col-span-2">
            <GlobalThreatMap
              hotspots={hotspots}
              period={period}
              onPeriodChange={setPeriod}
              onHotspotClick={setSelectedHotspot}
              selectedHotspot={selectedHotspot}
              liveEvents={liveEvents}
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
              <CardTitle className="text-sm">Live Event Feed</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <LiveEventFeed events={liveEvents.length > 0 ? liveEvents : recentEvents} />
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <Card className="xl:col-span-2">
            <CardHeader className="flex flex-row items-center justify-between p-4 pb-2">
              <CardTitle className="text-sm">Active Campaigns</CardTitle>
              <Link href="/cyber/cti/campaigns" className="text-xs text-primary hover:underline">
                View all →
              </Link>
            </CardHeader>
            <CardContent className="p-4 pt-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Campaign</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Actor</TableHead>
                    <TableHead className="text-right">IOCs</TableHead>
                    <TableHead className="text-right">Events</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {recentCampaigns.length > 0 ? recentCampaigns.map((campaign) => (
                    <TableRow
                      key={campaign.id}
                      className="cursor-pointer"
                      onClick={() => router.push(`/cyber/cti/campaigns?campaign=${campaign.id}`)}
                    >
                      <TableCell>
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <p className="font-medium">{campaign.name}</p>
                            <CTISeverityBadge severity={campaign.severity_code} size="sm" />
                          </div>
                          <p className="text-xs text-muted-foreground">{campaign.campaign_code}</p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <CTIStatusBadge status={campaign.status} type="campaign" />
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {campaign.actor_name || 'Unknown actor'}
                      </TableCell>
                      <TableCell className="text-right font-medium tabular-nums">
                        {campaign.ioc_count.toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right font-medium tabular-nums">
                        {campaign.event_count.toLocaleString()}
                      </TableCell>
                    </TableRow>
                  )) : (
                    <TableRow>
                      <TableCell colSpan={5} className="py-8 text-center text-sm text-muted-foreground">
                        No campaigns available.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between p-4 pb-2">
              <CardTitle className="text-sm">Critical Brand Abuse Alerts</CardTitle>
              <Link href="/cyber/cti/brand-abuse" className="text-xs text-primary hover:underline">
                View all →
              </Link>
            </CardHeader>
            <CardContent className="space-y-3 p-4 pt-0">
              {urgentBrands.length > 0 ? urgentBrands.map((incident) => (
                <button
                  key={incident.id}
                  type="button"
                  onClick={() => router.push('/cyber/cti/brand-abuse')}
                  className="w-full rounded-xl border border-white/10 bg-slate-950/40 p-3 text-left transition hover:bg-slate-950/60"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{incident.malicious_domain}</p>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {incident.brand_name} · {incident.region_label || 'Unknown region'}
                      </p>
                    </div>
                    <CTISeverityBadge severity={incident.risk_level} size="sm" />
                  </div>
                  <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
                    <CTIStatusBadge status={incident.takedown_status} type="takedown" className="text-[10px]" />
                    <span>{incident.detection_count} detections</span>
                  </div>
                </button>
              )) : (
                <p className="py-8 text-center text-sm text-muted-foreground">
                  No active brand abuse incidents.
                </p>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <Card className="xl:col-span-2">
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm">Top Targeted Sectors</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 p-4 pt-0">
              {sectors.slice(0, 6).map((sector) => (
                <div key={sector.id} className="rounded-xl border border-white/10 bg-slate-950/35 p-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="font-medium">{sector.sector_label}</p>
                      <p className="text-xs text-muted-foreground">
                        {sector.total_count.toLocaleString()} events in {period}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      {sector.severity_critical_count > 0 && <CTISeverityBadge severity="critical" size="sm" />}
                      {sector.severity_high_count > 0 && <CTISeverityBadge severity="high" size="sm" />}
                    </div>
                  </div>
                </div>
              ))}
              {sectors.length === 0 && (
                <p className="py-8 text-center text-sm text-muted-foreground">No sector analytics available.</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm">Executive Risk Posture</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-4 pt-0">
              <div className="flex items-center justify-center">
                <RiskScoreGauge
                  score={snapshot.risk_score_overall}
                  trend={snapshot.trend_direction}
                  size={150}
                />
              </div>
              <div className="grid grid-cols-2 gap-2 text-center">
                <div className="rounded-xl border border-white/10 bg-slate-950/40 p-3">
                  <p className="text-lg font-semibold tabular-nums">{mttd.toFixed(1)}h</p>
                  <p className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">MTTD</p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-950/40 p-3">
                  <p className="text-lg font-semibold tabular-nums">{mttr.toFixed(1)}h</p>
                  <p className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">MTTR</p>
                </div>
              </div>
              <div className="rounded-xl border border-white/10 bg-slate-950/40 p-3 text-sm">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-muted-foreground">Top Origin</span>
                  <span className="font-medium uppercase">
                    {snapshot.top_threat_origin_country || '—'}
                  </span>
                </div>
                <div className="mt-2 flex items-center justify-between gap-2">
                  <span className="text-muted-foreground">Top Sector</span>
                  <span className="font-medium">{topSector}</span>
                </div>
                <div className="mt-2 flex items-center justify-between gap-2">
                  <span className="text-muted-foreground">Refresh Window</span>
                  <span className="font-medium">
                    <Radar className="mr-1 inline h-3.5 w-3.5" />
                    {period}
                  </span>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </PermissionRedirect>
  );
}
