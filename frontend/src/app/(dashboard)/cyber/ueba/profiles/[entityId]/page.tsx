'use client';

import { useQuery } from '@tanstack/react-query';
import { useParams } from 'next/navigation';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';

import { ActivityHeatmap } from '../../_components/activity-heatmap';
import { VolumeTimeline } from '../../_components/volume-timeline';
import { TableAccessList } from '../../_components/table-access-list';
import { SourceIPList } from '../../_components/source-ip-list';
import { BaselineComparisonCard } from '../../_components/baseline-comparison-card';
import { SignalEvidenceViewer } from '../../_components/signal-evidence-viewer';
import { RiskScoreHistory } from '../../_components/risk-score-history';
import type {
  UebaAlert,
  UebaHeatmapResponse,
  UebaProfileDetailResponse,
  UebaTimelineResponse,
} from '../../_components/types';

function badgeVariant(level: string) {
  if (level === 'critical') return 'destructive' as const;
  if (level === 'high') return 'warning' as const;
  if (level === 'mature') return 'success' as const;
  return 'outline' as const;
}

export default function UebaProfileDetailPage() {
  const params = useParams<{ entityId: string }>();
  const entityId = decodeURIComponent(params?.entityId ?? '');

  const profileQuery = useQuery({
    queryKey: ['cyber-ueba-profile', entityId],
    queryFn: () => apiGet<{ data: UebaProfileDetailResponse }>(`${API_ENDPOINTS.CYBER_UEBA_PROFILES}/${encodeURIComponent(entityId)}`),
  });
  const heatmapQuery = useQuery({
    queryKey: ['cyber-ueba-heatmap', entityId],
    queryFn: () => apiGet<{ data: UebaHeatmapResponse }>(`${API_ENDPOINTS.CYBER_UEBA_PROFILES}/${encodeURIComponent(entityId)}/heatmap?days=30`),
  });
  const timelineQuery = useQuery({
    queryKey: ['cyber-ueba-timeline', entityId],
    queryFn: () => apiGet<UebaTimelineResponse>(`${API_ENDPOINTS.CYBER_UEBA_PROFILES}/${encodeURIComponent(entityId)}/timeline?per_page=20`),
  });
  const alertsQuery = useQuery({
    queryKey: ['cyber-ueba-entity-alerts', entityId],
    queryFn: () => apiGet<{ data: UebaAlert[] }>(`${API_ENDPOINTS.CYBER_UEBA_ALERTS}?entity_id=${encodeURIComponent(entityId)}`),
  });

  const detail = profileQuery.data?.data;
  const profile = detail?.profile;

  if (profileQuery.isLoading || heatmapQuery.isLoading || timelineQuery.isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <div className="space-y-6">
          <LoadingSkeleton variant="card" />
          <LoadingSkeleton variant="card" />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (!profile || profileQuery.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState message="Failed to load the UEBA profile." onRetry={() => void profileQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const comparison = detail?.baseline_comparison ?? {};
  const accessComparison = comparison['access_times'] as Record<string, unknown> | undefined;
  const volumeComparison = comparison['data_volume'] as Record<string, unknown> | undefined;
  const patternComparison = comparison['access_patterns'] as Record<string, unknown> | undefined;
  const failureComparison = comparison['failure_rate'] as Record<string, unknown> | undefined;

  const timelinePoints = (volumeComparison?.['actual_last_7d_volume'] as Array<Record<string, unknown>> | undefined) ?? [];
  const recentTables = (patternComparison?.['actual_recent_tables'] as string[] | undefined) ?? [];
  const recentIPs = (patternComparison?.['actual_recent_source_ips'] as string[] | undefined) ?? [];
  const alerts = alertsQuery.data?.data ?? [];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title={profile.entity_name ?? profile.entity_id}
          description={`${profile.entity_type.replaceAll('_', ' ')} behavioral baseline`}
          actions={
            <div className="flex flex-wrap gap-2">
              <Badge variant={badgeVariant(profile.risk_level)}>
                Risk {profile.risk_score.toFixed(0)} · {profile.risk_level}
              </Badge>
              <Badge variant={badgeVariant(profile.profile_maturity)}>{profile.profile_maturity}</Badge>
              <Badge variant="outline">{profile.status}</Badge>
            </div>
          }
        />

        <Tabs defaultValue="activity" className="space-y-4">
          <TabsList>
            <TabsTrigger value="activity">Activity</TabsTrigger>
            <TabsTrigger value="alerts">Alerts</TabsTrigger>
            <TabsTrigger value="baseline">Baseline</TabsTrigger>
            <TabsTrigger value="risk">Risk History</TabsTrigger>
          </TabsList>

          <TabsContent value="activity" className="space-y-4">
            <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
              <Card>
                <CardHeader><CardTitle className="text-base">Access Heatmap</CardTitle></CardHeader>
                <CardContent>
                  <ActivityHeatmap matrix={heatmapQuery.data?.data.matrix ?? []} />
                </CardContent>
              </Card>
              <Card>
                <CardHeader><CardTitle className="text-base">Recent Source IPs</CardTitle></CardHeader>
                <CardContent>
                  <SourceIPList
                    expectedIPs={profile.baseline.source_ips ?? []}
                    actualIPs={recentIPs}
                  />
                </CardContent>
              </Card>
            </div>

            <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
              <Card>
                <CardHeader><CardTitle className="text-base">Volume Timeline</CardTitle></CardHeader>
                <CardContent>
                  <VolumeTimeline
                    points={timelinePoints}
                    expectedBytesMean={Number(volumeComparison?.['expected_daily_bytes_mean'] ?? 0)}
                    expectedRowsMean={Number(volumeComparison?.['expected_daily_rows_mean'] ?? 0)}
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader><CardTitle className="text-base">Recent Table Access</CardTitle></CardHeader>
                <CardContent>
                  <TableAccessList
                    expectedTables={profile.baseline.access_patterns?.tables_accessed ?? []}
                    actualTables={recentTables}
                  />
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="alerts" className="space-y-4">
            {alerts.map((alert) => (
              <Card key={alert.id}>
                <CardHeader>
                  <div className="flex items-center justify-between gap-3">
                    <CardTitle className="text-base">{alert.title}</CardTitle>
                    <Badge variant={alert.severity === 'critical' ? 'destructive' : alert.severity === 'high' ? 'warning' : 'outline'}>
                      {alert.severity}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <SignalEvidenceViewer alert={alert} />
                </CardContent>
              </Card>
            ))}
            {alerts.length === 0 && (
              <Card><CardContent className="p-8 text-center text-muted-foreground">No entity alerts found.</CardContent></Card>
            )}
          </TabsContent>

          <TabsContent value="baseline" className="space-y-4">
            <div className="grid gap-4">
              <BaselineComparisonCard
                title="Access Time Comparison"
                expected={{
                  peak_hours: accessComparison?.['expected_peak_hours'],
                  active_hours: accessComparison?.['expected_active_hours'],
                }}
                actual={accessComparison?.['actual_last_7d_heatmap']}
              />
              <BaselineComparisonCard
                title="Volume Comparison"
                expected={{
                  daily_bytes_mean: volumeComparison?.['expected_daily_bytes_mean'],
                  daily_rows_mean: volumeComparison?.['expected_daily_rows_mean'],
                }}
                actual={volumeComparison?.['actual_last_7d_volume']}
              />
              <BaselineComparisonCard
                title="Access Pattern Comparison"
                expected={{
                  tables: patternComparison?.['expected_tables'],
                  source_ips: patternComparison?.['expected_source_ips'],
                }}
                actual={{
                  recent_tables: patternComparison?.['actual_recent_tables'],
                  recent_source_ips: patternComparison?.['actual_recent_source_ips'],
                }}
              />
              <BaselineComparisonCard
                title="Failure Rate Comparison"
                expected={failureComparison?.['expected_failure_rate_percent']}
                actual={timelineQuery.data?.data.filter((event) => !event.success).length ?? 0}
              />
            </div>
          </TabsContent>

          <TabsContent value="risk" className="space-y-4">
            <Card>
              <CardHeader><CardTitle className="text-base">Risk Score History</CardTitle></CardHeader>
              <CardContent>
                <RiskScoreHistory history={detail?.risk_history ?? []} />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </PermissionRedirect>
  );
}
