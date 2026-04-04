'use client';

import Link from 'next/link';
import { ArrowRight, AlertTriangle, CheckCircle2, Database, FileQuestion, GitBranch } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { AreaChart } from '@/components/shared/charts/area-chart';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { GaugeChart } from '@/components/shared/charts/gauge-chart';
import { LineChart } from '@/components/shared/charts/line-chart';
import { RelativeTime } from '@/components/shared/relative-time';
import { SectionCard } from '@/components/suites/section-card';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { type DataEnvelope, type DataSuiteDashboard } from '@/lib/data-suite';
import { API_ENDPOINTS } from '@/lib/constants';
import {
  buildPipelineTrendSeries,
  buildSourceStatusChartRows,
  formatMaybeDurationMs,
  qualitySeverityVisuals,
} from '@/lib/data-suite/utils';
import { formatCompactNumber, formatDate, formatPercentage } from '@/lib/format';
import { cn } from '@/lib/utils';

const KPI_LINKS = {
  total_sources: '/data/sources',
  active_pipelines: '/data/pipelines',
  quality_score: '/data/quality',
  open_contradictions: '/data/contradictions',
  dark_data_assets: '/data/dark-data',
} as const;

export default function DataPage() {
  const { data: envelope, isLoading, error, mutate, isValidating } = useRealtimeData<DataEnvelope<DataSuiteDashboard>>(
    API_ENDPOINTS.DATA_DASHBOARD,
    {
      wsTopics: [
        'pipeline.run.completed',
        'pipeline.run.failed',
        'quality.check_failed',
        'contradiction.detected',
      ],
      pollInterval: 60_000,
    },
  );

  const dashboard = envelope?.data;

  if (isLoading || !dashboard) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Data Suite" description="Unified operational view across sources, models, pipelines, quality, lineage, and governed analytics." />
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
            {Array.from({ length: 5 }).map((_, index) => (
              <LoadingSkeleton key={index} variant="card" />
            ))}
          </div>
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <LoadingSkeleton variant="chart" />
            <LoadingSkeleton variant="chart" />
          </div>
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <LoadingSkeleton variant="chart" />
            <LoadingSkeleton variant="chart" />
          </div>
          <LoadingSkeleton variant="chart" />
        </div>
      </PermissionRedirect>
    );
  }

  if (error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message={error.message} onRetry={() => void mutate()} />
      </PermissionRedirect>
    );
  }

  const pipelineSeries = buildPipelineTrendSeries(dashboard);
  const sourceStatusRows = buildSourceStatusChartRows(dashboard);

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Suite"
          description="Operational command center for sources, pipelines, quality posture, contradictions, dark data, lineage, and governed analytics."
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" asChild>
                <Link href="/data/sources">Manage sources</Link>
              </Button>
              <Button size="sm" asChild>
                <Link href="/data/pipelines">Open pipelines</Link>
              </Button>
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
          <DashboardKpiCard
            href={KPI_LINKS.total_sources}
            title="Total Sources"
            value={dashboard.kpis.total_sources.toLocaleString()}
            subtitle={`${dashboard.kpis.sources_delta >= 0 ? '+' : ''}${dashboard.kpis.sources_delta} since last period`}
            icon={Database}
          />
          <DashboardKpiCard
            href={KPI_LINKS.active_pipelines}
            title="Active Pipelines"
            value={dashboard.kpis.active_pipelines.toLocaleString()}
            subtitle={`${dashboard.kpis.failed_pipelines_24h} failed in last 24h`}
            icon={GitBranch}
          />
          <DashboardKpiCard
            href={KPI_LINKS.quality_score}
            title="Quality Score"
            value={`${dashboard.kpis.quality_score.toFixed(1)}`}
            subtitle={`Grade ${dashboard.kpis.quality_grade}`}
            icon={CheckCircle2}
            rightSlot={
              <GaugeChart
                value={dashboard.kpis.quality_score}
                size={64}
                showValue={false}
                thresholds={{ good: 90, warning: 70 }}
              />
            }
          />
          <DashboardKpiCard
            href={KPI_LINKS.open_contradictions}
            title="Open Contradictions"
            value={dashboard.kpis.open_contradictions.toLocaleString()}
            subtitle={`${dashboard.kpis.contradictions_delta >= 0 ? '+' : ''}${dashboard.kpis.contradictions_delta} trend`}
            icon={AlertTriangle}
            tint={dashboard.kpis.open_contradictions > 0 ? 'danger' : 'default'}
          />
          <DashboardKpiCard
            href={KPI_LINKS.dark_data_assets}
            title="Dark Data Assets"
            value={dashboard.kpis.dark_data_assets.toLocaleString()}
            subtitle={`${dashboard.dark_data_stats.total_assets ?? 0} total assets tracked`}
            icon={FileQuestion}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <SectionCard
            title="Pipeline Success Rate"
            description="Last 30 days of pipeline outcomes."
            actions={
              <span className="text-xs text-muted-foreground">
                Success rate {formatPercentage(dashboard.pipeline_success_rate_30d / 100, 1)}
              </span>
            }
          >
            <AreaChart
              data={pipelineSeries}
              xKey="day"
              height={320}
              stacked
              yKeys={[
                { key: 'success', label: 'Success', color: '#16a34a' },
                { key: 'failed', label: 'Failed', color: '#dc2626' },
                { key: 'cancelled', label: 'Cancelled', color: '#94a3b8' },
              ]}
              xFormatter={(value) => formatDate(`${value}`, 'MMM d')}
            />
          </SectionCard>

          <SectionCard title="Quality Score Trend" description="30-day rolling quality score from the live quality service.">
            <LineChart
              data={dashboard.quality_trend_30d.map((point) => ({
                day: point.day,
                value: point.value,
              }))}
              xKey="day"
              height={320}
              yKeys={[{ key: 'value', label: 'Quality score', color: '#2563eb' }]}
              xFormatter={(value) => formatDate(`${value}`, 'MMM d')}
            />
          </SectionCard>
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <SectionCard
            title="Recent Pipeline Runs"
            description="Last 10 executions."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/data/pipelines">
                  View all
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead className="text-left text-xs uppercase tracking-wide text-muted-foreground">
                  <tr>
                    <th className="pb-2 font-medium">Pipeline</th>
                    <th className="pb-2 font-medium">Status</th>
                    <th className="pb-2 font-medium">Duration</th>
                    <th className="pb-2 font-medium">Completed</th>
                  </tr>
                </thead>
                <tbody>
                  {dashboard.recent_runs.map((run) => (
                    <tr key={run.id} className="border-t">
                      <td className="py-3">
                        <Link href={`/data/pipelines/${run.pipeline_id}`} className="font-medium hover:text-primary">
                          {run.pipeline_name}
                        </Link>
                      </td>
                      <td className="py-3">
                        <span className="inline-flex items-center gap-2 text-xs font-medium">
                          <span className={cn('h-2.5 w-2.5 rounded-full', run.status === 'completed' ? 'bg-emerald-500' : run.status === 'failed' ? 'bg-rose-500' : 'bg-sky-500')} />
                          {run.status}
                        </span>
                      </td>
                      <td className="py-3 text-muted-foreground">{formatMaybeDurationMs(run.duration_ms)}</td>
                      <td className="py-3 text-muted-foreground">
                        {run.completed_at ? <RelativeTime date={run.completed_at} /> : <RelativeTime date={run.started_at} />}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </SectionCard>

          <SectionCard
            title="Quality Issues"
            description="Current failed or warning rules with impacted records."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/data/quality">
                  Open quality
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead className="text-left text-xs uppercase tracking-wide text-muted-foreground">
                  <tr>
                    <th className="pb-2 font-medium">Model</th>
                    <th className="pb-2 font-medium">Rule</th>
                    <th className="pb-2 font-medium">Severity</th>
                    <th className="pb-2 font-medium">Failures</th>
                  </tr>
                </thead>
                <tbody>
                  {dashboard.top_quality_failures.map((item) => {
                    const severity = qualitySeverityVisuals[item.severity] ?? qualitySeverityVisuals.low;
                    return (
                      <tr key={item.rule_id} className="border-t">
                        <td className="py-3">
                          <Link href={`/data/quality?model=${item.model_id}`} className="font-medium hover:text-primary">
                            {item.model_name}
                          </Link>
                        </td>
                        <td className="py-3 text-muted-foreground">{item.rule_name}</td>
                        <td className="py-3">
                          <span className={cn('inline-flex rounded-full border px-2 py-0.5 text-xs font-medium', severity.className)}>
                            {severity.label}
                          </span>
                        </td>
                        <td className="py-3 text-muted-foreground">{formatCompactNumber(item.records_failed)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </SectionCard>
        </div>

        <SectionCard
          title="Sources by Status"
          description="Source-type coverage overlaid with current status mix from the dashboard."
          actions={
            <span className="text-xs text-muted-foreground">{isValidating ? 'Refreshing…' : 'Live every 60s'}</span>
          }
        >
          <BarChart
            data={sourceStatusRows}
            xKey="type"
            layout="horizontal"
            stacked
            height={360}
            yKeys={[
              { key: 'active', label: 'Active', color: '#16a34a' },
              { key: 'inactive', label: 'Inactive', color: '#94a3b8' },
              { key: 'error', label: 'Error', color: '#dc2626' },
              { key: 'syncing', label: 'Syncing', color: '#2563eb' },
            ]}
          />
        </SectionCard>
      </div>
    </PermissionRedirect>
  );
}

interface DashboardKpiCardProps {
  href: string;
  title: string;
  value: string;
  subtitle: string;
  icon: typeof Database;
  tint?: 'default' | 'danger';
  rightSlot?: React.ReactNode;
}

function DashboardKpiCard({
  href,
  title,
  value,
  subtitle,
  icon: Icon,
  tint = 'default',
  rightSlot,
}: DashboardKpiCardProps) {
  return (
    <Link href={href} className="block">
      <Card className={cn('h-full transition-colors hover:border-primary/40 hover:bg-muted/30', tint === 'danger' && 'border-rose-200 bg-rose-50/40')}>
        <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
            <CardDescription className="text-xs">{subtitle}</CardDescription>
          </div>
          {rightSlot ?? <Icon className={cn('h-5 w-5', tint === 'danger' ? 'text-rose-600' : 'text-primary')} />}
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-semibold tracking-tight">{value}</div>
        </CardContent>
      </Card>
    </Link>
  );
}
