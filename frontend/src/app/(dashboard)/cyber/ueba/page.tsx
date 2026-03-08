'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { Fingerprint, ShieldAlert } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';

import { RiskRankingChart } from './_components/risk-ranking-chart';
import { AlertTypeDistribution } from './_components/alert-type-distribution';
import { AlertTrendChart } from './_components/alert-trend-chart';
import { ProfileTable } from './_components/profile-table';
import type { UebaDashboard } from './_components/types';

function KpiCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className="border-border/70 bg-gradient-to-br from-background to-muted/30">
      <CardContent className="p-5">
        <div className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</div>
        <div className="mt-2 text-3xl font-semibold tracking-tight">{value}</div>
      </CardContent>
    </Card>
  );
}

export default function UebaDashboardPage() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['cyber-ueba-dashboard'],
    queryFn: () => apiGet<{ data: UebaDashboard }>(API_ENDPOINTS.CYBER_UEBA_DASHBOARD),
  });

  const dashboard = data?.data;

  if (isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <div className="space-y-6">
          <PageHeader title="UEBA" description="Behavioral analytics for users, services, and applications." />
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => <LoadingSkeleton key={index} variant="card" />)}
          </div>
          <LoadingSkeleton variant="card" />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (!dashboard || error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState
          message="Failed to load the UEBA dashboard."
          onRetry={() => void refetch()}
        />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="UEBA"
          description="Persistent baseline learning, anomaly correlation, and risk-ranked entity exposure."
          actions={
            <div className="flex gap-2">
              <Button asChild variant="outline" size="sm">
                <Link href="/cyber/ueba/alerts">Alerts</Link>
              </Button>
            </div>
          }
        />

        <Card className="overflow-hidden border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.16),_transparent_38%),linear-gradient(135deg,_rgba(17,24,39,0.04),_transparent)]">
          <CardContent className="flex items-center gap-4 p-6">
            <div className="rounded-full bg-sky-500/10 p-3 text-sky-600">
              <Fingerprint className="h-6 w-6" />
            </div>
            <div className="space-y-1">
              <div className="font-semibold">Precision-first behavioral detection</div>
              <div className="max-w-3xl text-sm text-muted-foreground">
                Learning profiles do not alert until they have enough evidence. Mature entities are ranked by decayed risk, not permanent historical stigma.
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <KpiCard label="Active Profiles" value={dashboard.kpis.active_profiles.toLocaleString()} />
          <KpiCard label="High Risk Entities" value={dashboard.kpis.high_risk_entities.toLocaleString()} />
          <KpiCard label="Alerts (7d)" value={dashboard.kpis.alerts_7d.toLocaleString()} />
          <KpiCard label="Avg Risk Score" value={dashboard.kpis.average_risk_score.toFixed(1)} />
        </div>

        <div className="grid gap-4 xl:grid-cols-[1.35fr_0.9fr]">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Risk Ranking</CardTitle>
            </CardHeader>
            <CardContent>
              <RiskRankingChart items={dashboard.risk_ranking} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Alert Type Distribution</CardTitle>
            </CardHeader>
            <CardContent>
              <AlertTypeDistribution items={dashboard.alert_type_distribution} />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Alert Trend</CardTitle>
          </CardHeader>
          <CardContent>
            <AlertTrendChart items={dashboard.alert_trend} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <ShieldAlert className="h-4 w-4" />
              Profile Ranking
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ProfileTable items={dashboard.profiles} />
          </CardContent>
        </Card>
      </div>
    </PermissionRedirect>
  );
}
