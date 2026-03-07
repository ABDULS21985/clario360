'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, CheckCircle, ShieldAlert, Waves } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { KpiCard } from '@/components/shared/kpi-card';
import { RelativeTime } from '@/components/shared/relative-time';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SectionCard } from '@/components/suites/section-card';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuiteData } from '@/lib/suite-api';
import { percent } from '@/lib/suite-utils';
import { truncate } from '@/lib/utils';
import type { QualityDashboard } from '@/types/suites';

export default function DataQualityPage() {
  const qualityQuery = useQuery({
    queryKey: ['data-quality-dashboard'],
    queryFn: () => fetchSuiteData<QualityDashboard>(API_ENDPOINTS.DATA_QUALITY),
  });

  if (qualityQuery.isLoading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Data Quality" description="Monitor and improve data quality metrics" />
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <LoadingSkeleton key={index} variant="card" />
            ))}
          </div>
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (qualityQuery.error || !qualityQuery.data) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message="Failed to load the data quality dashboard." onRetry={() => void qualityQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const dashboard = qualityQuery.data;

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Quality"
          description="Quality scorecard backed by the live data-service quality dashboard."
          actions={
            <Button variant="outline" size="sm" asChild>
              <Link href="/data/contradictions">
                Investigate contradictions
                <ArrowRight className="ml-1 h-3.5 w-3.5" />
              </Link>
            </Button>
          }
        />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard
            title="Quality Score"
            value={`${dashboard.score.toFixed(1)}%`}
            change={dashboard.trend}
            changeLabel="last 7d"
            icon={CheckCircle}
            iconColor="text-green-600"
          />
          <KpiCard
            title="Pass Rate"
            value={percent(dashboard.pass_rate, 1)}
            icon={Waves}
            iconColor="text-blue-600"
            description={`${dashboard.results_last_7_days} evaluations in the last 7 days`}
          />
          <KpiCard
            title="Enabled Rules"
            value={dashboard.enabled_rules}
            icon={CheckCircle}
            iconColor="text-violet-600"
            description={`${dashboard.total_rules} total rules configured`}
          />
          <KpiCard
            title="Critical Failures"
            value={dashboard.critical_failures}
            icon={ShieldAlert}
            iconColor="text-red-600"
            description={`${dashboard.failed_last_7_days} failed checks in the last 7 days`}
          />
        </div>

        <div className="grid gap-4 xl:grid-cols-[1.35fr_0.85fr]">
          <SectionCard title="Recent Failures" description="Latest rule failures with affected model context.">
            <div className="space-y-3">
              {dashboard.recent_failures.length === 0 ? (
                <p className="text-sm text-muted-foreground">No recent failures were reported.</p>
              ) : (
                dashboard.recent_failures.map((failure, index) => (
                  <div
                    key={`${failure.rule_name}-${failure.model_name}-${index}`}
                    className="rounded-lg border px-4 py-3"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{failure.rule_name}</p>
                        <p className="text-xs text-muted-foreground">{failure.model_name}</p>
                      </div>
                      <SeverityIndicator severity={normalizeSeverity(failure.severity)} size="sm" />
                    </div>
                    <div className="mt-2 flex items-center justify-between gap-3 text-xs text-muted-foreground">
                      <span>{failure.records_failed.toLocaleString()} records failed</span>
                      <RelativeTime date={failure.checked_at} />
                    </div>
                    {failure.failure_samples ? (
                      <p className="mt-2 text-xs text-muted-foreground">
                        {truncate(JSON.stringify(failure.failure_samples), 160)}
                      </p>
                    ) : null}
                  </div>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard title="Control Posture" description="Coverage and failure ratios from live quality telemetry.">
            <div className="space-y-4">
              {[
                {
                  label: 'Rule Enablement',
                  value: dashboard.total_rules > 0 ? (dashboard.enabled_rules / dashboard.total_rules) * 100 : 0,
                },
                {
                  label: 'Pass Rate',
                  value: dashboard.pass_rate,
                },
                {
                  label: 'Critical Failure Ratio',
                  value: dashboard.results_last_7_days > 0 ? (dashboard.critical_failures / dashboard.results_last_7_days) * 100 : 0,
                },
              ].map((metric) => (
                <div key={metric.label}>
                  <div className="mb-1 flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">{metric.label}</span>
                    <span className="font-medium">{percent(metric.value, 1)}</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-muted">
                    <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${Math.min(metric.value, 100)}%` }} />
                  </div>
                </div>
              ))}
            </div>
          </SectionCard>
        </div>
      </div>
    </PermissionRedirect>
  );
}

function normalizeSeverity(value: string): 'critical' | 'high' | 'medium' | 'low' | 'info' {
  switch (value) {
    case 'critical':
    case 'high':
    case 'medium':
    case 'low':
      return value;
    default:
      return 'info';
  }
}
