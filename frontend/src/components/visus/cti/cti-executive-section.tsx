'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { AlertTriangle, ArrowRight, RefreshCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { CTIBrandAbuseWidget } from './cti-brand-abuse-widget';
import { CTICampaignsWidget } from './cti-campaigns-widget';
import { CTIKPIRowWidget } from './cti-kpi-row-widget';
import { CTIRiskSummaryWidget } from './cti-risk-summary-widget';
import { CTISectorChartMiniWidget } from './cti-sector-chart-mini-widget';
import { CTIThreatMapMiniWidget } from './cti-threat-map-mini-widget';
import { useVisusCTIWidgets } from '@/hooks/use-visus-cti-widgets';
import type { CTIExecutiveSnapshot } from '@/types/cti';

interface CTIExecutiveSectionProps {
  className?: string;
}

function WidgetErrorCard({
  title,
  description,
  onRetry,
}: {
  title: string;
  description: string;
  onRetry: () => void;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <AlertTriangle className="h-4 w-4 text-amber-500" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{description}</p>
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCcw className="mr-2 h-4 w-4" />
          Retry
        </Button>
      </CardContent>
    </Card>
  );
}

function buildRiskSnapshot(
  overviewSnapshot: CTIExecutiveSnapshot | null | undefined,
  riskScore: {
    risk_score: number;
    trend_direction: string;
    trend_percentage: number;
    total_events_24h: number;
    mttd_hours: number;
    mttr_hours: number;
    computed_at: string;
  } | null | undefined,
): CTIExecutiveSnapshot | null {
  if (overviewSnapshot) {
    return overviewSnapshot;
  }

  if (!riskScore) {
    return null;
  }

  return {
    tenant_id: '',
    total_events_24h: riskScore.total_events_24h,
    total_events_7d: 0,
    total_events_30d: 0,
    active_campaigns_count: 0,
    critical_campaigns_count: 0,
    total_iocs: 0,
    brand_abuse_critical_count: 0,
    brand_abuse_total_count: 0,
    top_targeted_sector_id: null,
    top_targeted_sector_label: null,
    top_threat_origin_country: null,
    mean_time_to_detect_hours: riskScore.mttd_hours,
    mean_time_to_respond_hours: riskScore.mttr_hours,
    risk_score_overall: riskScore.risk_score,
    trend_direction: riskScore.trend_direction as CTIExecutiveSnapshot['trend_direction'],
    trend_percentage: riskScore.trend_percentage,
    computed_at: riskScore.computed_at,
  };
}

export function CTIExecutiveSection({ className }: CTIExecutiveSectionProps) {
  const router = useRouter();
  const { overview, threatMap, sectors, campaigns, brandAbuse, riskScore, refetch, queries } = useVisusCTIWidgets();

  const riskSnapshot = buildRiskSnapshot(overview?.snapshot, riskScore);
  const campaignRows = campaigns?.data ?? overview?.top_campaigns ?? [];
  const brandRows = brandAbuse?.data ?? overview?.critical_brands ?? [];
  const sectorRows = sectors?.sectors ?? overview?.top_sectors ?? [];
  const hotspotRows = threatMap?.hotspots ?? [];

  return (
    <section className={className}>
      <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="space-y-1">
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-sky-600">Cyber Threat Intelligence</p>
          <h2 className="text-2xl font-semibold tracking-tight">Executive CTI Snapshot</h2>
          <p className="text-sm text-muted-foreground">
            Live threat posture, active campaigns, geographic hotspots, and sector targeting from the cyber suite.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void refetch()}>
            <RefreshCcw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" asChild>
            <Link href="/cyber/cti">
              View Full CTI
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </div>
      </div>

      <div className="space-y-4">
        <CTIKPIRowWidget snapshot={overview?.snapshot ?? null} isLoading={queries.overview.isLoading} />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[0.95fr_1.45fr]">
          {(queries.riskScore.error || (queries.overview.error && !overview?.snapshot)) && !riskSnapshot ? (
            <WidgetErrorCard
              title="CTI Risk Summary"
              description="The executive risk snapshot could not be loaded."
              onRetry={() => {
                void queries.overview.refetch();
                void queries.riskScore.refetch();
              }}
            />
          ) : (
            <CTIRiskSummaryWidget
              snapshot={riskSnapshot}
              isLoading={queries.overview.isLoading && queries.riskScore.isLoading}
              onViewDetails={() => router.push('/cyber/cti')}
            />
          )}

          {queries.threatMap.error && hotspotRows.length === 0 ? (
            <WidgetErrorCard
              title="Threat Map"
              description="The CTI threat map is temporarily unavailable."
              onRetry={() => {
                void queries.threatMap.refetch();
              }}
            />
          ) : (
            <CTIThreatMapMiniWidget
              hotspots={hotspotRows}
              isLoading={queries.threatMap.isLoading}
              onExpand={() => router.push('/cyber/cti/geo')}
            />
          )}
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          {queries.campaigns.error && campaignRows.length === 0 ? (
            <WidgetErrorCard
              title="Active Campaigns"
              description="Campaign data could not be loaded from the CTI bridge."
              onRetry={() => {
                void queries.campaigns.refetch();
              }}
            />
          ) : (
            <CTICampaignsWidget
              campaigns={campaignRows}
              isLoading={queries.campaigns.isLoading}
              onViewAll={() => router.push('/cyber/cti/campaigns')}
              onCampaignClick={(id) => router.push(`/cyber/cti/campaigns/${id}`)}
            />
          )}

          {queries.brandAbuse.error && brandRows.length === 0 ? (
            <WidgetErrorCard
              title="Critical Brand Abuse"
              description="Brand abuse signals are temporarily unavailable."
              onRetry={() => {
                void queries.brandAbuse.refetch();
              }}
            />
          ) : (
            <CTIBrandAbuseWidget
              incidents={brandRows}
              isLoading={queries.brandAbuse.isLoading}
              onViewAll={() => router.push('/cyber/cti/brand-abuse')}
              onIncidentClick={(id) => router.push(`/cyber/cti/brand-abuse/${id}`)}
            />
          )}

          {queries.sectors.error && sectorRows.length === 0 ? (
            <WidgetErrorCard
              title="Sector Targeting"
              description="Sector aggregation data is temporarily unavailable."
              onRetry={() => {
                void queries.sectors.refetch();
              }}
            />
          ) : (
            <CTISectorChartMiniWidget
              sectors={sectorRows}
              isLoading={queries.sectors.isLoading}
              onExpand={() => router.push('/cyber/cti/sectors')}
            />
          )}
        </div>
      </div>
    </section>
  );
}
