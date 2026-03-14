'use client';

import { useRouter } from 'next/navigation';
import { Users, Shield, ShieldAlert, AlertTriangle, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { AccessKpiCards } from './_components/access-kpi-cards';
import type { AccessDashboard, IdentityProfile } from '@/types/cyber';

interface RiskRankingEntry {
  identity_id: string;
  identity_name: string;
  risk_score: number;
  blast_radius_score: number;
  overprivileged_count: number;
}

interface OverprivilegedResult {
  data: Array<Record<string, unknown>>;
  total: number;
}

export default function AccessIntelligencePage() {
  const router = useRouter();

  const {
    data: dashEnvelope,
    isLoading: dashLoading,
    error: dashError,
    mutate: refetchDash,
  } = useRealtimeData<{ data: AccessDashboard }>(API_ENDPOINTS.CYBER_DSPM_ACCESS_DASHBOARD, {
    pollInterval: 120000,
  });

  const {
    data: riskRankingEnvelope,
    isLoading: riskLoading,
  } = useRealtimeData<{ data: RiskRankingEntry[] }>(API_ENDPOINTS.CYBER_DSPM_ACCESS_RISK_RANKING, {
    pollInterval: 120000,
  });

  const {
    data: overprivEnvelope,
    isLoading: overprivLoading,
  } = useRealtimeData<{ data: OverprivilegedResult }>(API_ENDPOINTS.CYBER_DSPM_ACCESS_OVERPRIVILEGED, {
    pollInterval: 120000,
  });

  const dashboard = dashEnvelope?.data;
  const riskRanking = riskRankingEnvelope?.data ?? [];
  const overprivData = overprivEnvelope?.data;

  const riskChartData = riskRanking.slice(0, 10).map((entry) => ({
    name: entry.identity_name.length > 20
      ? `${entry.identity_name.slice(0, 18)}...`
      : entry.identity_name,
    risk_score: entry.risk_score,
  }));

  function getRiskBadgeVariant(score: number): 'destructive' | 'secondary' | 'outline' {
    if (score >= 75) return 'destructive';
    if (score >= 50) return 'secondary';
    return 'outline';
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Access Intelligence"
          description="Monitor identity-to-data mappings, detect overprivileged access, and enforce least-privilege governance"
          actions={
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => router.push('/cyber/dspm/access/identities')}
              >
                <Users className="mr-1.5 h-3.5 w-3.5" />
                Identities
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => router.push('/cyber/dspm/access/policies')}
              >
                <Shield className="mr-1.5 h-3.5 w-3.5" />
                Policies
              </Button>
            </div>
          }
        />

        {dashLoading ? (
          <LoadingSkeleton variant="card" />
        ) : dashError || !dashboard ? (
          <ErrorState
            message="Failed to load Access Intelligence dashboard"
            onRetry={() => void refetchDash()}
          />
        ) : (
          <>
            <AccessKpiCards dashboard={dashboard} />

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              {/* Risk ranking bar chart */}
              <div className="rounded-xl border bg-card p-5 lg:col-span-2">
                <h3 className="mb-4 text-sm font-semibold">Top 10 Riskiest Identities</h3>
                {riskLoading ? (
                  <LoadingSkeleton variant="chart" />
                ) : riskChartData.length === 0 ? (
                  <div className="rounded-lg border bg-muted/20 p-4 text-sm text-muted-foreground">
                    No risk ranking data available yet. Run an access collection to populate rankings.
                  </div>
                ) : (
                  <BarChart
                    data={riskChartData}
                    xKey="name"
                    yKeys={[{ key: 'risk_score', label: 'Risk Score', color: 'hsl(0, 84%, 60%)' }]}
                    layout="horizontal"
                    height={320}
                    showLegend={false}
                    yFormatter={(v) => `${v}`}
                  />
                )}
              </div>

              {/* Summary cards */}
              <div className="space-y-4">
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="flex items-center gap-2 text-sm">
                      <AlertTriangle className="h-4 w-4 text-orange-500" />
                      Overprivileged Findings
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-3xl font-bold tabular-nums">
                      {overprivLoading ? '...' : (overprivData?.total ?? dashboard.overprivileged_mappings)}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      Access mappings exceeding required permissions
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="flex items-center gap-2 text-sm">
                      <Clock className="h-4 w-4 text-amber-500" />
                      Stale Access
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-3xl font-bold tabular-nums">
                      {dashboard.stale_permissions}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      Permissions unused for 90+ days
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="flex items-center gap-2 text-sm">
                      <ShieldAlert className="h-4 w-4 text-red-500" />
                      Risk Distribution
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {Object.entries(dashboard.risk_distribution).map(([level, count]) => (
                        <div key={level} className="flex items-center justify-between text-sm">
                          <span className="capitalize text-muted-foreground">{level}</span>
                          <Badge variant={level === 'critical' || level === 'high' ? 'destructive' : 'secondary'}>
                            {count}
                          </Badge>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>

            {/* Top risky identities table */}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-semibold">Top Risky Identities</CardTitle>
                <p className="text-xs text-muted-foreground">
                  Identities with the highest composite risk scores based on access patterns and blast radius
                </p>
              </CardHeader>
              <CardContent>
                {dashboard.top_risky_identities.length === 0 ? (
                  <div className="rounded-lg border bg-muted/20 p-4 text-sm text-muted-foreground">
                    No risky identities detected. Run an access collection to analyze identity risk profiles.
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b text-left text-xs text-muted-foreground">
                          <th className="pb-3 pr-4 font-medium">Name</th>
                          <th className="pb-3 pr-4 font-medium">Type</th>
                          <th className="pb-3 pr-4 font-medium text-right">Risk Score</th>
                          <th className="pb-3 pr-4 font-medium text-right">Blast Radius</th>
                          <th className="pb-3 font-medium text-right">Overprivileged</th>
                        </tr>
                      </thead>
                      <tbody>
                        {dashboard.top_risky_identities.map((identity: IdentityProfile) => (
                          <tr
                            key={identity.id}
                            className="cursor-pointer border-b last:border-0 hover:bg-muted/50 transition-colors"
                            onClick={() => router.push(`/cyber/dspm/access/identities/${identity.id}`)}
                          >
                            <td className="py-3 pr-4">
                              <div>
                                <p className="font-medium">{identity.identity_name}</p>
                                <p className="text-xs text-muted-foreground">{identity.identity_email}</p>
                              </div>
                            </td>
                            <td className="py-3 pr-4">
                              <Badge variant="outline" className="capitalize">
                                {identity.identity_type.replace(/_/g, ' ')}
                              </Badge>
                            </td>
                            <td className="py-3 pr-4 text-right">
                              <Badge variant={getRiskBadgeVariant(identity.access_risk_score)}>
                                {Math.round(identity.access_risk_score)}
                              </Badge>
                            </td>
                            <td className="py-3 pr-4 text-right tabular-nums">
                              {Math.round(identity.blast_radius_score)}
                            </td>
                            <td className="py-3 text-right tabular-nums">
                              {identity.overprivileged_count}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
