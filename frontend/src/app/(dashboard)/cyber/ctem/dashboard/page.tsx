'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { KpiCard } from '@/components/shared/kpi-card';
import { AreaChart } from '@/components/shared/charts/area-chart';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { ExposureScore, ExposureScorePoint } from '@/types/cyber';

import { ExposureScoreGauge } from '../_components/exposure-score-gauge';

export default function CTEMDashboardPage() {
  const scoreQuery = useQuery({
    queryKey: ['ctem-exposure-score'],
    queryFn: () => apiGet<{ data: ExposureScore }>(API_ENDPOINTS.CYBER_CTEM_EXPOSURE_SCORE),
    refetchInterval: 60000,
  });

  const historyQuery = useQuery({
    queryKey: ['ctem-exposure-history'],
    queryFn: () =>
      apiGet<{ data: ExposureScorePoint[] }>(API_ENDPOINTS.CYBER_CTEM_EXPOSURE_HISTORY, {
        days: 90,
      }),
    refetchInterval: 300000,
  });

  const score = scoreQuery.data?.data;
  const history = historyQuery.data?.data ?? [];

  const historyChart = history.map((p) => ({
    date: new Date(p.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    score: p.score,
  }));

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="CTEM Dashboard"
          description="Continuous Threat Exposure Management — track your organization's exposure score and remediation progress."
          actions={
            <Button variant="outline" size="sm" asChild>
              <Link href="/cyber/ctem">
                <ArrowRight className="mr-1.5 h-3.5 w-3.5" />
                View Assessments
              </Link>
            </Button>
          }
        />

        {/* Row 1: Exposure Score + KPIs */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
          <Card className="lg:col-span-1">
            <CardContent className="flex items-center justify-center py-6">
              {scoreQuery.isLoading ? (
                <LoadingSkeleton variant="chart" />
              ) : (
                <ExposureScoreGauge />
              )}
            </CardContent>
          </Card>

          <div className="lg:col-span-3 grid grid-cols-1 gap-4 md:grid-cols-3">
            <KpiCard
              title="Exposure Score"
              value={score?.score?.toFixed(1) ?? '—'}
              description={score?.grade ? `Grade: ${score.grade}` : undefined}
              loading={scoreQuery.isLoading}
            />
            <KpiCard
              title="Trend"
              value={score?.trend?.replace(/_/g, ' ') ?? '—'}
              description={
                score?.trend_delta !== undefined
                  ? `${score.trend_delta > 0 ? '+' : ''}${score.trend_delta.toFixed(1)} pts`
                  : undefined
              }
              iconColor={
                score?.trend === 'improving'
                  ? 'text-green-600'
                  : score?.trend === 'worsening'
                    ? 'text-red-600'
                    : undefined
              }
              loading={scoreQuery.isLoading}
            />
            <KpiCard
              title="Last Calculated"
              value={
                score?.calculated_at
                  ? new Date(score.calculated_at).toLocaleDateString()
                  : '—'
              }
              loading={scoreQuery.isLoading}
            />
          </div>
        </div>

        {/* Row 2: Exposure Score Trend */}
        <AreaChart
          title="Exposure Score Trend (90 Days)"
          data={historyChart}
          xKey="date"
          yKeys={[{ key: 'score', label: 'Exposure Score', color: '#DC2626' }]}
          height={320}
        />

        {/* Row 3: Quick Links */}
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Run New Assessment</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-3">
                Start a new CTEM assessment to discover exposures, prioritize risks, and create remediation plans.
              </p>
              <Button size="sm" asChild>
                <Link href="/cyber/ctem">Go to Assessments</Link>
              </Button>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Exposure Score Methodology</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Score = 35% Vulnerability + 25% Attack Surface + 25% Threat Activity + 15% Velocity. Graded A (≤20) through F (&gt;80).
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </PermissionRedirect>
  );
}
