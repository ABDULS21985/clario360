'use client';

import Link from 'next/link';
import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useQueries, useQuery } from '@tanstack/react-query';
import { ArrowLeft, GitBranch } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { RootCauseAnalysisPanel } from '@/components/cyber/root-cause-analysis-panel';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { RunDetailPanel } from '@/app/(dashboard)/data/pipelines/[id]/_components/run-detail-panel';
import { RunProgressTracker } from '@/app/(dashboard)/data/pipelines/[id]/_components/run-progress-tracker';
import { PipelineConfigTab } from '@/app/(dashboard)/data/pipelines/[id]/_components/pipeline-config-tab';
import { PipelineLineageTab } from '@/app/(dashboard)/data/pipelines/[id]/_components/pipeline-lineage-tab';
import { PipelineQualityTab } from '@/app/(dashboard)/data/pipelines/[id]/_components/pipeline-quality-tab';
import { PipelineRunsTab } from '@/app/(dashboard)/data/pipelines/[id]/_components/pipeline-runs-tab';
import { dataSuiteApi, type PipelineRun } from '@/lib/data-suite';
import { formatMaybeCompact, formatMaybeDateTime, formatMaybeDurationMs } from '@/lib/data-suite/utils';
import type { RootCauseAnalysis } from '@/types/cyber';

export default function DataPipelineDetailPage() {
  const params = useParams<{ id: string }>();
  const pipelineId = params?.id ?? '';
  const [selectedRun, setSelectedRun] = useState<PipelineRun | null>(null);
  const [activeTab, setActiveTab] = useState('runs');

  const [pipelineQuery, runsQuery, lineageQuery] = useQueries({
    queries: [
      { queryKey: ['data-pipeline', pipelineId], queryFn: () => dataSuiteApi.getPipeline(pipelineId) },
      {
        queryKey: ['data-pipeline-runs', pipelineId],
        queryFn: () =>
          dataSuiteApi.listPipelineRuns(pipelineId, {
            page: 1,
            per_page: 50,
            sort: 'started_at',
            order: 'desc',
          }),
      },
      {
        queryKey: ['data-pipeline-lineage', pipelineId],
        queryFn: () => dataSuiteApi.getEntityLineageGraph('pipeline', pipelineId),
      },
    ],
  });

  const logsQuery = useQuery({
    queryKey: ['data-pipeline-run-logs', pipelineId, selectedRun?.id],
    queryFn: () => dataSuiteApi.getPipelineRunLogs(pipelineId, selectedRun!.id),
    enabled: Boolean(selectedRun?.id),
  });

  const pipeline = pipelineQuery.data;
  const runs = runsQuery.data?.data ?? [];
  const latestRun = runs[0] ?? null;
  const failedRun = selectedRun?.status === 'failed'
    ? selectedRun
    : runs.find((run) => run.status === 'failed') ?? null;
  const rootCauseQuery = useQuery({
    queryKey: ['data-pipeline-root-cause', failedRun?.id],
    queryFn: () => apiGet<{ data: RootCauseAnalysis }>(`/api/v1/rca/pipeline_failure/${failedRun!.id}`),
    enabled: activeTab === 'root-cause' && Boolean(failedRun?.id),
    staleTime: 120000,
  });
  const error = [pipelineQuery, runsQuery, lineageQuery].find((query) => query.error)?.error;

  if (pipelineQuery.isLoading || !pipeline) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Pipeline Detail" description="Loading pipeline runs, configuration, and lineage." />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message={error instanceof Error ? error.message : 'Failed to load pipeline detail.'} onRetry={() => void pipelineQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title={pipeline.name}
          description={pipeline.description || 'Pipeline execution, configuration, quality, and lineage detail.'}
          actions={
            <div className="flex items-center gap-2">
              {failedRun ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setSelectedRun(failedRun);
                    setActiveTab('root-cause');
                  }}
                >
                  <GitBranch className="mr-1.5 h-3.5 w-3.5" />
                  Analyze Root Cause
                </Button>
              ) : null}
              <Button variant="outline" size="sm" asChild>
                <Link href="/data/pipelines">
                  <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
                  Back to pipelines
                </Link>
              </Button>
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard label="Status" value={pipeline.status} />
          <SummaryCard label="Runs" value={pipeline.total_runs.toLocaleString()} />
          <SummaryCard label="Processed" value={formatMaybeCompact(pipeline.total_records_processed)} />
          <SummaryCard label="Avg Duration" value={formatMaybeDurationMs(pipeline.avg_duration_ms)} />
        </div>

        <RunProgressTracker run={latestRun} />

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="runs">Runs</TabsTrigger>
            <TabsTrigger value="config">Config</TabsTrigger>
            <TabsTrigger value="quality">Quality</TabsTrigger>
            <TabsTrigger value="lineage">Lineage</TabsTrigger>
            <TabsTrigger value="root-cause">Root Cause</TabsTrigger>
          </TabsList>

          <TabsContent value="runs" className="space-y-4">
            <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
              Last run {formatMaybeDateTime(pipeline.last_run_at)} • status {pipeline.last_run_status ?? 'never run'}
            </div>
            <PipelineRunsTab runs={runs} onSelectRun={setSelectedRun} />
          </TabsContent>
          <TabsContent value="config">
            <PipelineConfigTab pipeline={pipeline} />
          </TabsContent>
          <TabsContent value="quality">
            <PipelineQualityTab pipeline={pipeline} />
          </TabsContent>
          <TabsContent value="lineage">
            <PipelineLineageTab pipelineId={pipelineId} graph={lineageQuery.data ?? null} />
          </TabsContent>
          <TabsContent value="root-cause">
            <RootCauseAnalysisPanel
              analysis={rootCauseQuery.data?.data}
              isLoading={rootCauseQuery.isLoading || rootCauseQuery.isFetching}
              error={rootCauseQuery.error instanceof Error ? rootCauseQuery.error.message : null}
              onAnalyze={failedRun ? () => void rootCauseQuery.refetch() : undefined}
              analyzeLabel="Refresh Analysis"
              emptyTitle="Analyze pipeline failure"
              emptyDescription="Trace the failure upstream through lineage, schema changes, and recent run history to isolate the real source of the outage."
              disabledReason={failedRun ? undefined : 'Select a failed run to analyze root cause.'}
            />
          </TabsContent>
        </Tabs>

        <RunDetailPanel
          open={Boolean(selectedRun)}
          onOpenChange={(open) => {
            if (!open) {
              setSelectedRun(null);
            }
          }}
          run={selectedRun}
          logs={logsQuery.data ?? []}
        />
      </div>
    </PermissionRedirect>
  );
}

function SummaryCard({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-lg font-semibold capitalize">{value}</div>
    </div>
  );
}
