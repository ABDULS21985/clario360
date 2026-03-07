'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, Boxes, CheckCircle2, FolderOpen, GitBranch } from 'lucide-react';
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
import { fetchSuiteData, fetchSuitePaginated } from '@/lib/suite-api';
import { objectKeyCount, percent } from '@/lib/suite-utils';
import { formatDateTime, truncate } from '@/lib/utils';
import type { DataPipeline, Dataset, DataSource, QualityDashboard } from '@/types/suites';

export default function DataPage() {
  const sourcesQuery = useQuery({
    queryKey: ['data-overview', 'sources'],
    queryFn: () =>
      fetchSuitePaginated<DataSource>(API_ENDPOINTS.DATA_SOURCES, {
        page: 1,
        per_page: 5,
        sort: 'updated_at',
        order: 'desc',
      }),
  });

  const pipelinesQuery = useQuery({
    queryKey: ['data-overview', 'pipelines'],
    queryFn: () =>
      fetchSuitePaginated<DataPipeline>(API_ENDPOINTS.DATA_PIPELINES, {
        page: 1,
        per_page: 6,
        sort: 'updated_at',
        order: 'desc',
      }),
  });

  const datasetsQuery = useQuery({
    queryKey: ['data-overview', 'datasets'],
    queryFn: () =>
      fetchSuitePaginated<Dataset>(API_ENDPOINTS.DATA_DATASETS, {
        page: 1,
        per_page: 5,
        sort: 'updated_at',
        order: 'desc',
      }),
  });

  const qualityQuery = useQuery({
    queryKey: ['data-overview', 'quality'],
    queryFn: () => fetchSuiteData<QualityDashboard>(API_ENDPOINTS.DATA_QUALITY),
  });

  const loading =
    sourcesQuery.isLoading &&
    pipelinesQuery.isLoading &&
    datasetsQuery.isLoading &&
    qualityQuery.isLoading;

  const fatalError =
    sourcesQuery.error &&
    pipelinesQuery.error &&
    datasetsQuery.error &&
    qualityQuery.error;

  if (loading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Data Intelligence" description="Data pipelines, quality monitoring, and dataset management" />
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <LoadingSkeleton key={index} variant="card" />
            ))}
          </div>
          <div className="grid gap-4 xl:grid-cols-3">
            <LoadingSkeleton variant="card" />
            <LoadingSkeleton variant="card" />
            <LoadingSkeleton variant="card" />
          </div>
        </div>
      </PermissionRedirect>
    );
  }

  if (fatalError) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState
          message="Failed to load data intelligence overview."
          onRetry={() => {
            void sourcesQuery.refetch();
            void pipelinesQuery.refetch();
            void datasetsQuery.refetch();
            void qualityQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  const quality = qualityQuery.data;
  const pipelines = pipelinesQuery.data?.data ?? [];
  const datasets = datasetsQuery.data?.data ?? [];
  const recentFailures = quality?.recent_failures ?? [];

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Intelligence"
          description="Operational view across connected sources, pipelines, datasets, and quality posture."
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" asChild>
                <Link href="/data/sources">Sources</Link>
              </Button>
              <Button size="sm" asChild>
                <Link href="/data/pipelines">Pipelines</Link>
              </Button>
            </div>
          }
        />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard
            title="Connected Sources"
            value={sourcesQuery.data?.meta.total ?? 0}
            icon={FolderOpen}
            iconColor="text-blue-600"
            description="Live source connections discovered in the tenant"
          />
          <KpiCard
            title="Datasets / Models"
            value={datasetsQuery.data?.meta.total ?? 0}
            icon={Boxes}
            iconColor="text-violet-600"
            description="Registered analytical models and datasets"
          />
          <KpiCard
            title="Pipeline Failures (7d)"
            value={quality?.failed_last_7_days ?? 0}
            icon={GitBranch}
            iconColor="text-orange-600"
            description="Quality failures observed in recent runs"
          />
          <KpiCard
            title="Quality Score"
            value={quality ? `${quality.score.toFixed(1)}%` : '—'}
            change={quality?.trend}
            changeLabel="last 7d"
            icon={CheckCircle2}
            iconColor="text-green-600"
            description={quality ? `${quality.critical_failures} critical exceptions open` : 'Quality telemetry unavailable'}
          />
        </div>

        <div className="grid gap-4 xl:grid-cols-3">
          <SectionCard
            title="Pipeline Operations"
            description="Most recently updated ingestion and transformation pipelines."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/data/pipelines">
                  View all
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {pipelines.length === 0 ? (
                <p className="text-sm text-muted-foreground">No pipelines are available for this tenant.</p>
              ) : (
                pipelines.map((pipeline) => (
                  <div key={pipeline.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{pipeline.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {pipeline.source_name ?? 'Unknown source'} → {pipeline.target_name ?? 'Unknown target'}
                        </p>
                      </div>
                      <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">
                        {pipeline.status.replace(/_/g, ' ')}
                      </span>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
                      <span>Type: {pipeline.type}</span>
                      <span>Last run: {pipeline.last_run_at ? formatDateTime(pipeline.last_run_at) : 'Never'}</span>
                      <span>
                        Processed: {(pipeline.last_run_records_processed ?? 0).toLocaleString()}
                      </span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard
            title="Quality Exceptions"
            description="Recent high-signal rule failures from the live quality dashboard."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/data/quality">
                  Open quality view
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {recentFailures.length === 0 ? (
                <p className="text-sm text-muted-foreground">No recent quality failures were reported.</p>
              ) : (
                recentFailures.slice(0, 5).map((failure, index) => (
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
                    <div className="mt-2 flex items-center justify-between text-xs text-muted-foreground">
                      <span>{failure.records_failed.toLocaleString()} records failed</span>
                      <RelativeTime date={failure.checked_at} />
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard
            title="Model Coverage"
            description="Recent datasets and their metadata completeness profile."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/data/models">
                  Browse models
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {datasets.length === 0 ? (
                <p className="text-sm text-muted-foreground">No datasets have been registered yet.</p>
              ) : (
                datasets.map((dataset) => (
                  <div key={dataset.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{dataset.name}</p>
                        <p className="text-xs text-muted-foreground">
                          v{dataset.version} • {dataset.source_name ?? 'Unmapped source'}
                        </p>
                      </div>
                      <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">
                        {dataset.status}
                      </span>
                    </div>
                    <div className="mt-2 grid grid-cols-2 gap-2 text-xs text-muted-foreground">
                      <span>{objectKeyCount(dataset.schema_definition)} schema fields</span>
                      <span>{objectKeyCount(dataset.lineage)} lineage links</span>
                    </div>
                    {dataset.description ? (
                      <p className="mt-2 text-xs text-muted-foreground">{truncate(dataset.description, 120)}</p>
                    ) : null}
                  </div>
                ))
              )}
            </div>
          </SectionCard>
        </div>

        <SectionCard
          title="Operating Baseline"
          description="Cross-cutting telemetry derived directly from the suite services."
        >
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <div className="rounded-lg border bg-muted/20 p-4">
              <p className="text-sm font-medium">Enabled Rules</p>
              <p className="mt-2 text-2xl font-semibold">{quality?.enabled_rules ?? 0}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {quality ? `${quality.total_rules} total quality rules configured` : 'No rules available'}
              </p>
            </div>
            <div className="rounded-lg border bg-muted/20 p-4">
              <p className="text-sm font-medium">Pass Rate</p>
              <p className="mt-2 text-2xl font-semibold">{quality ? percent(quality.pass_rate, 1) : '—'}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {quality ? `${quality.results_last_7_days} checks in the last 7 days` : 'Pass rate unavailable'}
              </p>
            </div>
            <div className="rounded-lg border bg-muted/20 p-4">
              <p className="text-sm font-medium">Source Metadata Coverage</p>
              <p className="mt-2 text-2xl font-semibold">
                {sourcesQuery.data?.data.filter((source) => objectKeyCount(source.schema_metadata) > 0).length ?? 0}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">Sources on this page with non-empty schema metadata</p>
            </div>
            <div className="rounded-lg border bg-muted/20 p-4">
              <p className="text-sm font-medium">Pipeline Freshness</p>
              <p className="mt-2 text-2xl font-semibold">
                {pipelines.filter((pipeline) => pipeline.last_run_at).length}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">Pipelines that have executed at least once</p>
            </div>
          </div>
        </SectionCard>
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
