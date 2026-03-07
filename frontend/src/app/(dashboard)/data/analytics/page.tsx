'use client';

import { useQuery } from '@tanstack/react-query';
import { BarChart3, Boxes, FolderOpen, GitBranch, TrendingUp } from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SectionCard } from '@/components/suites/section-card';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuiteData, fetchSuitePaginated } from '@/lib/suite-api';
import { percent } from '@/lib/suite-utils';
import type { DataPipeline, Dataset, DataSource, QualityDashboard } from '@/types/suites';

export default function DataAnalyticsPage() {
  const sourcesQuery = useQuery({
    queryKey: ['data-analytics', 'sources'],
    queryFn: () => fetchSuitePaginated<DataSource>(API_ENDPOINTS.DATA_SOURCES, { page: 1, per_page: 100, order: 'desc' }),
  });
  const pipelinesQuery = useQuery({
    queryKey: ['data-analytics', 'pipelines'],
    queryFn: () => fetchSuitePaginated<DataPipeline>(API_ENDPOINTS.DATA_PIPELINES, { page: 1, per_page: 100, order: 'desc' }),
  });
  const datasetsQuery = useQuery({
    queryKey: ['data-analytics', 'datasets'],
    queryFn: () => fetchSuitePaginated<Dataset>(API_ENDPOINTS.DATA_DATASETS, { page: 1, per_page: 100, order: 'desc' }),
  });
  const qualityQuery = useQuery({
    queryKey: ['data-analytics', 'quality'],
    queryFn: () => fetchSuiteData<QualityDashboard>(API_ENDPOINTS.DATA_QUALITY),
  });

  if (sourcesQuery.isLoading && pipelinesQuery.isLoading && datasetsQuery.isLoading && qualityQuery.isLoading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Analytics" description="Operational analytics derived from the live data suite." />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (sourcesQuery.error && pipelinesQuery.error && datasetsQuery.error && qualityQuery.error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState
          message="Failed to load data analytics."
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

  const sources = sourcesQuery.data?.data ?? [];
  const pipelines = pipelinesQuery.data?.data ?? [];
  const datasets = datasetsQuery.data?.data ?? [];
  const quality = qualityQuery.data;

  const sourceTypeCounts = countBy(sources.map((source) => source.type));
  const pipelineStatusCounts = countBy(pipelines.map((pipeline) => pipeline.status));
  const datasetStatusCounts = countBy(datasets.map((dataset) => dataset.status));

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Analytics" description="Derived operational analytics across ingestion, modeling, and quality telemetry." />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Source Types" value={Object.keys(sourceTypeCounts).length} icon={FolderOpen} iconColor="text-blue-600" />
          <KpiCard title="Active Pipelines" value={pipelineStatusCounts.active ?? 0} icon={GitBranch} iconColor="text-green-600" />
          <KpiCard title="Published Models" value={(datasetStatusCounts.published ?? 0) + (datasetStatusCounts.active ?? 0)} icon={Boxes} iconColor="text-violet-600" />
          <KpiCard title="Pass Rate" value={quality ? percent(quality.pass_rate, 1) : '—'} icon={TrendingUp} iconColor="text-emerald-600" />
        </div>

        <div className="grid gap-4 xl:grid-cols-3">
          <SectionCard title="Source Mix" description="Distribution of connected source types.">
            <MetricRows rows={sourceTypeCounts} />
          </SectionCard>
          <SectionCard title="Pipeline Status" description="Current status distribution across pipelines.">
            <MetricRows rows={pipelineStatusCounts} />
          </SectionCard>
          <SectionCard title="Dataset Lifecycle" description="Observed lifecycle mix for datasets and models.">
            <MetricRows rows={datasetStatusCounts} />
          </SectionCard>
        </div>

        <SectionCard title="Quality Analytics" description="Derived quality telemetry from the live quality dashboard.">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {[
              { label: 'Results Last 7d', value: quality?.results_last_7_days ?? 0 },
              { label: 'Failures Last 7d', value: quality?.failed_last_7_days ?? 0 },
              { label: 'Critical Failures', value: quality?.critical_failures ?? 0 },
              { label: 'Enabled Rules', value: quality?.enabled_rules ?? 0 },
            ].map((item) => (
              <div key={item.label} className="rounded-lg border bg-muted/20 p-4">
                <p className="text-sm font-medium">{item.label}</p>
                <p className="mt-2 text-2xl font-semibold">{item.value}</p>
              </div>
            ))}
          </div>
        </SectionCard>
      </div>
    </PermissionRedirect>
  );
}

function countBy(values: string[]): Record<string, number> {
  return values.reduce<Record<string, number>>((acc, value) => {
    acc[value] = (acc[value] ?? 0) + 1;
    return acc;
  }, {});
}

function MetricRows({ rows }: { rows: Record<string, number> }) {
  const entries = Object.entries(rows).sort((left, right) => right[1] - left[1]);
  if (entries.length === 0) {
    return <p className="text-sm text-muted-foreground">No records available.</p>;
  }

  return (
    <div className="space-y-3">
      {entries.map(([label, value]) => (
        <div key={label}>
          <div className="mb-1 flex items-center justify-between text-sm">
            <span className="capitalize text-muted-foreground">{label.replace(/_/g, ' ')}</span>
            <span className="font-medium">{value}</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-muted">
            <div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(value * 12, 100)}%` }} />
          </div>
        </div>
      ))}
    </div>
  );
}
