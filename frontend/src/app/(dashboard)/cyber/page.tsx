'use client';

import { useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { RefreshCw, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import type { SOCDashboard } from '@/types/cyber';
import type { VulnerabilityAgingReport } from './_components/vuln-aging-chart';

import { SocKpiCards } from './_components/soc-kpi-cards';
import { AlertTimelineChart } from './_components/alert-timeline-chart';
import { SeverityDistributionChart } from './_components/severity-distribution-chart';
import { MitreHeatmapWidget } from './_components/mitre-heatmap-widget';
import { VulnAgingChart } from './_components/vuln-aging-chart';
import { RecentAlertsTable } from './_components/recent-alerts-table';
import { TopAttackedAssetsTable } from './_components/top-attacked-assets-table';
import { AnalystWorkloadChart } from './_components/analyst-workload-chart';

export default function SocDashboardPage() {
  const router = useRouter();

  const {
    data: dashboardEnvelope,
    isLoading,
    error,
    mutate,
  } = useRealtimeData<{ data: SOCDashboard }>(API_ENDPOINTS.CYBER_DASHBOARD, {
    wsTopics: [
      'cyber.alert.created',
      'cyber.alert.status_changed',
      'cyber.threat.detected',
      'cyber.vulnerability.created',
    ],
    pollInterval: 60000,
  });

  const {
    data: agingEnvelope,
    isLoading: agingLoading,
    error: agingError,
    mutate: retryAging,
  } = useRealtimeData<{ data: VulnerabilityAgingReport }>(API_ENDPOINTS.CYBER_VULNERABILITIES_AGING, {
    pollInterval: 120000,
  });

  const handleRefresh = useCallback(() => {
    void mutate();
    void retryAging();
  }, [mutate, retryAging]);

  const dashboard = dashboardEnvelope?.data;
  const agingData = agingEnvelope?.data;

  if (isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <div className="space-y-6">
          <PageHeader
            title="Security Operations Center"
            description="Real-time security monitoring and threat intelligence"
          />
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <LoadingSkeleton key={i} variant="card" />
            ))}
          </div>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <LoadingSkeleton variant="card" />
            <LoadingSkeleton variant="card" />
          </div>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <LoadingSkeleton variant="card" />
            <LoadingSkeleton variant="card" />
          </div>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <LoadingSkeleton variant="table-row" count={6} />
            <LoadingSkeleton variant="table-row" count={6} />
          </div>
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (error || !dashboard) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState
          message="Failed to load SOC dashboard. Please try again."
          onRetry={() => void mutate()}
        />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Security Operations Center"
          description="Real-time security monitoring and threat intelligence"
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={handleRefresh}>
                <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                Refresh
              </Button>
              <Button variant="ghost" size="sm" onClick={() => router.push('/settings')}>
                <Settings className="h-4 w-4" />
              </Button>
            </div>
          }
        />

        {/* ROW 1 — KPI Cards */}
        <SocKpiCards kpis={dashboard.kpis} />

        {/* ROW 2 — Timeline + Distribution */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="rounded-lg border bg-card p-4">
            <AlertTimelineChart data={dashboard.alert_timeline} />
          </div>
          <div className="rounded-lg border bg-card p-4">
            <SeverityDistributionChart data={dashboard.severity_distribution} />
          </div>
        </div>

        {/* ROW 3 — MITRE Heatmap + Vuln Aging */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="rounded-lg border bg-card p-4">
            <h3 className="mb-3 text-sm font-semibold">MITRE ATT&CK Heatmap</h3>
            <MitreHeatmapWidget data={dashboard.mitre_heatmap} />
          </div>
          <div className="rounded-lg border bg-card p-4">
            <VulnAgingChart
              data={agingData}
              loading={agingLoading}
              error={agingError?.message}
              onRetry={() => void retryAging()}
            />
          </div>
        </div>

        {/* ROW 4 — Recent Alerts + Top Attacked Assets */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="rounded-lg border bg-card p-4">
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-sm font-semibold">Recent Critical Alerts</h3>
              <button
                className="text-xs text-primary hover:underline"
                onClick={() => router.push('/cyber/alerts?severity=critical')}
              >
                View all →
              </button>
            </div>
            <RecentAlertsTable alerts={dashboard.recent_alerts} />
          </div>
          <div className="rounded-lg border bg-card p-4">
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-sm font-semibold">Top Attacked Assets</h3>
              <button
                className="text-xs text-primary hover:underline"
                onClick={() => router.push('/cyber/assets')}
              >
                View all →
              </button>
            </div>
            <TopAttackedAssetsTable assets={dashboard.top_attacked_assets} />
          </div>
        </div>

        {/* ROW 5 — Analyst Workload */}
        <div className="rounded-lg border bg-card p-4">
          <AnalystWorkloadChart data={dashboard.analyst_workload} />
        </div>

        {dashboard.partial_failures && dashboard.partial_failures.length > 0 && (
          <p className="text-xs text-muted-foreground">
            Some sections may be incomplete: {dashboard.partial_failures.join(', ')}
          </p>
        )}
      </div>
    </PermissionRedirect>
  );
}
