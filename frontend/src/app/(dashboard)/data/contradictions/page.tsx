'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, AlertOctagon, Zap } from 'lucide-react';
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
import { truncate } from '@/lib/utils';
import type { QualityDashboard, QualityFailure } from '@/types/suites';

export default function DataContradictionsPage() {
  const qualityQuery = useQuery({
    queryKey: ['data-contradictions'],
    queryFn: () => fetchSuiteData<QualityDashboard>(API_ENDPOINTS.DATA_QUALITY),
  });

  if (qualityQuery.isLoading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Contradictions" description="Derived contradiction view from live data quality exceptions." />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (qualityQuery.error || !qualityQuery.data) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message="Failed to load contradiction candidates." onRetry={() => void qualityQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const candidates = qualityQuery.data.recent_failures
    .filter((failure) => failure.records_failed > 0)
    .sort((left, right) => severityRank(right.severity) - severityRank(left.severity) || right.records_failed - left.records_failed);

  const highPriority = candidates.filter((failure) => severityRank(failure.severity) >= severityRank('high'));
  const affectedModels = new Set(candidates.map((failure) => failure.model_name)).size;
  const totalFailedRecords = candidates.reduce((sum, failure) => sum + failure.records_failed, 0);

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Contradictions"
          description="Derived operational view of conflicting or failing records from the live quality exception feed."
          actions={
            <Button variant="outline" size="sm" asChild>
              <Link href="/data/quality">
                Open quality dashboard
                <ArrowRight className="ml-1 h-3.5 w-3.5" />
              </Link>
            </Button>
          }
        />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Candidates" value={candidates.length} icon={Zap} iconColor="text-amber-600" />
          <KpiCard title="High Priority" value={highPriority.length} icon={AlertOctagon} iconColor="text-red-600" />
          <KpiCard title="Affected Models" value={affectedModels} icon={Zap} iconColor="text-blue-600" />
          <KpiCard title="Failed Records" value={totalFailedRecords.toLocaleString()} icon={AlertOctagon} iconColor="text-violet-600" />
        </div>

        <SectionCard
          title="Ranked Contradiction Candidates"
          description="This page is derived from the current quality failure stream. Use it to triage the most disruptive exceptions first."
        >
          <div className="space-y-3">
            {candidates.length === 0 ? (
              <p className="text-sm text-muted-foreground">No contradiction candidates are currently present.</p>
            ) : (
              candidates.map((failure, index) => (
                <ContradictionRow
                  key={`${failure.rule_name}-${failure.model_name}-${index}`}
                  failure={failure}
                />
              ))
            )}
          </div>
        </SectionCard>
      </div>
    </PermissionRedirect>
  );
}

function ContradictionRow({ failure }: { failure: QualityFailure }) {
  return (
    <div className="rounded-lg border px-4 py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-medium">{failure.rule_name}</p>
          <p className="text-xs text-muted-foreground">{failure.model_name}</p>
        </div>
        <SeverityIndicator severity={normalizeSeverity(failure.severity)} size="sm" />
      </div>
      <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
        <span>{failure.records_failed.toLocaleString()} conflicting records</span>
        <RelativeTime date={failure.checked_at} />
      </div>
      {failure.failure_samples ? (
        <p className="mt-2 text-xs text-muted-foreground">{truncate(JSON.stringify(failure.failure_samples), 180)}</p>
      ) : null}
    </div>
  );
}

function severityRank(value: string): number {
  switch (value) {
    case 'critical':
      return 5;
    case 'high':
      return 4;
    case 'medium':
      return 3;
    case 'low':
      return 2;
    default:
      return 1;
  }
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
