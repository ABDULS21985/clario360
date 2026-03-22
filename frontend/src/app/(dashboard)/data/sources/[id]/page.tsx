'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useSearchParams, useRouter } from 'next/navigation';
import { useQueries } from '@tanstack/react-query';
import { ArrowLeft, Power } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useAuth } from '@/hooks/use-auth';
import { dataSuiteApi, type DataModel, type Pipeline, type QualityRule } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';
import { formatMaybeBytes, formatMaybeCompact, getSourceTypeVisual } from '@/lib/data-suite/utils';
import { SourceActivityTab } from '@/app/(dashboard)/data/sources/[id]/_components/source-activity-tab';
import { SourceLineageTab } from '@/app/(dashboard)/data/sources/[id]/_components/source-lineage-tab';
import { SourceOverviewTab } from '@/app/(dashboard)/data/sources/[id]/_components/source-overview-tab';
import { SourcePipelinesTab } from '@/app/(dashboard)/data/sources/[id]/_components/source-pipelines-tab';
import { SourceQualityTab } from '@/app/(dashboard)/data/sources/[id]/_components/source-quality-tab';
import { SourceSchemaTab } from '@/app/(dashboard)/data/sources/[id]/_components/source-schema-tab';

const TABS = ['overview', 'schema', 'pipelines', 'quality', 'lineage', 'activity'] as const;
type SourceTabValue = (typeof TABS)[number];

export default function SourceDetailPage() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const { hasPermission } = useAuth();
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);
  const [togglingStatus, setTogglingStatus] = useState(false);

  const sourceId = params?.id ?? '';
  const activeTab = TABS.includes((searchParams?.get('tab') ?? 'overview') as SourceTabValue)
    ? (searchParams?.get('tab') as SourceTabValue | null) ?? 'overview'
    : 'overview';

  const [sourceQuery, schemaQuery, statsQuery, syncHistoryQuery, modelsQuery, pipelinesQuery, rulesQuery, lineageQuery] =
    useQueries({
      queries: [
        { queryKey: ['data-source', sourceId], queryFn: () => dataSuiteApi.getSource(sourceId) },
        { queryKey: ['data-source-schema', sourceId], queryFn: () => dataSuiteApi.getSourceSchema(sourceId) },
        { queryKey: ['data-source-stats', sourceId], queryFn: () => dataSuiteApi.getSourceStats(sourceId) },
        { queryKey: ['data-source-sync-history', sourceId], queryFn: () => dataSuiteApi.listSourceSyncHistory(sourceId, 20) },
        {
          queryKey: ['data-source-models', sourceId],
          queryFn: () =>
            dataSuiteApi.listModels({
              page: 1,
              per_page: 200,
              sort: 'updated_at',
              order: 'desc',
              filters: { source_id: sourceId },
            }),
        },
        {
          queryKey: ['data-source-pipelines', sourceId],
          queryFn: () =>
            dataSuiteApi.listPipelines({
              page: 1,
              per_page: 200,
              sort: 'updated_at',
              order: 'desc',
              filters: { source_id: sourceId },
            }),
        },
        {
          queryKey: ['data-source-quality-rules'],
          queryFn: () =>
            dataSuiteApi.listQualityRules({
              page: 1,
              per_page: 200,
              sort: 'updated_at',
              order: 'desc',
            }),
        },
        {
          queryKey: ['data-source-lineage', sourceId],
          queryFn: () => dataSuiteApi.getEntityLineageGraph('data_source', sourceId),
        },
      ],
    });

  const source = sourceQuery.data;
  const models = modelsQuery.data?.data ?? [];
  const pipelines = pipelinesQuery.data?.data ?? [];
  const rules = rulesQuery.data?.data ?? [];
  const relevantRules = useMemo<QualityRule[]>(
    () => {
      const modelIds = new Set(models.map((model) => model.id));
      return rules.filter((rule) => modelIds.has(rule.model_id));
    },
    [models, rules],
  );

  const isLoading = [sourceQuery, schemaQuery, statsQuery, syncHistoryQuery].some((query) => query.isLoading);
  const firstError = [sourceQuery, schemaQuery, statsQuery, syncHistoryQuery, modelsQuery, pipelinesQuery, rulesQuery, lineageQuery]
    .find((query) => query.error)?.error;

  if (isLoading || !source) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Source Detail" description="Loading source metadata, schema, and lineage." />
          <LoadingSkeleton variant="card" />
          <LoadingSkeleton variant="chart" />
        </div>
      </PermissionRedirect>
    );
  }

  if (firstError) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message={firstError instanceof Error ? firstError.message : 'Failed to load source detail.'} onRetry={() => void sourceQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const typeVisual = getSourceTypeVisual(source.type);

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title={source.name}
          description={source.description || 'Governed source detail with schema, lineage, quality, and pipeline context.'}
          actions={
            <div className="flex items-center gap-2">
              {(source.status === 'active' || source.status === 'inactive') && (
                <Button
                  variant={source.status === 'active' ? 'outline' : 'default'}
                  size="sm"
                  onClick={() => setStatusDialogOpen(true)}
                >
                  <Power className="mr-1.5 h-3.5 w-3.5" />
                  {source.status === 'active' ? 'Deactivate' : 'Activate'}
                </Button>
              )}
              <Button variant="outline" size="sm" asChild>
                <Link href="/data/sources">
                  <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
                  Back to sources
                </Link>
              </Button>
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard label="Type" value={typeVisual.label} />
          <SummaryCard label="Tables" value={formatMaybeCompact(source.table_count)} />
          <SummaryCard label="Rows" value={formatMaybeCompact(source.total_row_count)} />
          <SummaryCard label="Size" value={formatMaybeBytes(source.total_size_bytes)} />
        </div>

        <Tabs
          value={activeTab}
          onValueChange={(nextValue) => {
            const params = new URLSearchParams(searchParams?.toString() ?? '');
            params.set('tab', nextValue);
            router.replace(`/data/sources/${sourceId}?${params.toString()}`);
          }}
        >
          <TabsList className="w-full justify-start">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="schema">Schema</TabsTrigger>
            <TabsTrigger value="pipelines">Pipelines</TabsTrigger>
            <TabsTrigger value="quality">Quality</TabsTrigger>
            <TabsTrigger value="lineage">Lineage</TabsTrigger>
            <TabsTrigger value="activity">Activity</TabsTrigger>
          </TabsList>

          <TabsContent value="overview">
            <SourceOverviewTab
              source={source}
              stats={statsQuery.data ?? null}
              syncHistory={syncHistoryQuery.data ?? []}
            />
          </TabsContent>
          <TabsContent value="schema">
            <SourceSchemaTab
              sourceId={sourceId}
              schema={schemaQuery.data ?? null}
              relatedModels={models as DataModel[]}
              canViewPii={hasPermission('data:pii')}
            />
          </TabsContent>
          <TabsContent value="pipelines">
            <SourcePipelinesTab pipelines={pipelines as Pipeline[]} />
          </TabsContent>
          <TabsContent value="quality">
            <SourceQualityTab models={models as DataModel[]} rules={relevantRules} />
          </TabsContent>
          <TabsContent value="lineage">
            <SourceLineageTab sourceId={sourceId} graph={lineageQuery.data ?? null} />
          </TabsContent>
          <TabsContent value="activity">
            <SourceActivityTab source={source} syncHistory={syncHistoryQuery.data ?? []} />
          </TabsContent>
        </Tabs>

        <ConfirmDialog
          open={statusDialogOpen}
          onOpenChange={setStatusDialogOpen}
          title={source.status === 'active' ? 'Deactivate Source' : 'Activate Source'}
          description={`Are you sure you want to ${source.status === 'active' ? 'deactivate' : 'activate'} "${source.name}"?`}
          confirmLabel={source.status === 'active' ? 'Deactivate' : 'Activate'}
          variant={source.status === 'active' ? 'destructive' : 'default'}
          onConfirm={async () => {
            const newStatus = source.status === 'active' ? 'inactive' : 'active';
            setTogglingStatus(true);
            try {
              await dataSuiteApi.changeSourceStatus(source.id, newStatus);
              showSuccess(`Source ${newStatus === 'active' ? 'activated' : 'deactivated'}.`);
              setStatusDialogOpen(false);
              void sourceQuery.refetch();
            } catch (error) {
              showApiError(error);
            } finally {
              setTogglingStatus(false);
            }
          }}
          loading={togglingStatus}
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
      <div className="mt-1 text-lg font-semibold">{value}</div>
    </div>
  );
}
