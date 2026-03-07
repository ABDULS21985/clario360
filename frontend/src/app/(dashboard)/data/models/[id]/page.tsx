'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useQueries } from '@tanstack/react-query';
import { ArrowLeft } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ModelQualityRules } from '@/app/(dashboard)/data/models/_components/model-quality-rules';
import { ModelSchemaViewer } from '@/app/(dashboard)/data/models/_components/model-schema-viewer';
import { ModelVersionHistory } from '@/app/(dashboard)/data/models/_components/model-version-history';
import { dataSuiteApi } from '@/lib/data-suite';
import { formatMaybeDateTime, getClassificationBadge } from '@/lib/data-suite/utils';

export default function DataModelDetailPage() {
  const params = useParams<{ id: string }>();
  const modelId = params.id;

  const [modelQuery, versionsQuery, lineageQuery, rulesQuery] = useQueries({
    queries: [
      { queryKey: ['data-model', modelId], queryFn: () => dataSuiteApi.getModel(modelId) },
      { queryKey: ['data-model-versions', modelId], queryFn: () => dataSuiteApi.getModelVersions(modelId) },
      { queryKey: ['data-model-lineage', modelId], queryFn: () => dataSuiteApi.getModelLineage(modelId) },
      {
        queryKey: ['data-model-rules', modelId],
        queryFn: () =>
          dataSuiteApi.listQualityRules({
            page: 1,
            per_page: 200,
            sort: 'updated_at',
            order: 'desc',
            filters: { model_id: modelId },
          }),
      },
    ],
  });

  const model = modelQuery.data;
  const error = [modelQuery, versionsQuery, lineageQuery, rulesQuery].find((query) => query.error)?.error;

  if (modelQuery.isLoading || !model) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Model Detail" description="Loading model schema, rules, lineage, and versions." />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message={error instanceof Error ? error.message : 'Failed to load model detail.'} onRetry={() => void modelQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const classification = getClassificationBadge(model.data_classification);

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title={model.display_name || model.name}
          description={model.description || 'Governed model definition with schema, quality, lineage, and version history.'}
          actions={
            <Button variant="outline" size="sm" asChild>
              <Link href="/data/models">
                <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
                Back to models
              </Link>
            </Button>
          }
        />

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard label="Status" value={model.status} />
          <SummaryCard label="Fields" value={model.field_count.toLocaleString()} />
          <SummaryCard label="PII Columns" value={model.pii_columns.length.toLocaleString()} />
          <SummaryCard label="Updated" value={formatMaybeDateTime(model.updated_at)} />
        </div>

        <Card>
          <CardContent className="flex flex-wrap items-center gap-3 py-4">
            <span className="text-sm text-muted-foreground">Classification</span>
            <span className={`inline-flex rounded-full border px-2 py-1 text-xs ${classification.className}`}>
              {classification.label}
            </span>
            {model.source_id ? (
              <Button variant="ghost" size="sm" asChild>
                <Link href={`/data/sources/${model.source_id}`}>Open source</Link>
              </Button>
            ) : null}
          </CardContent>
        </Card>

        <Tabs defaultValue="schema">
          <TabsList>
            <TabsTrigger value="schema">Schema</TabsTrigger>
            <TabsTrigger value="quality">Quality Rules</TabsTrigger>
            <TabsTrigger value="lineage">Lineage</TabsTrigger>
            <TabsTrigger value="versions">Versions</TabsTrigger>
          </TabsList>

          <TabsContent value="schema">
            <ModelSchemaViewer model={model} />
          </TabsContent>
          <TabsContent value="quality">
            <ModelQualityRules rules={rulesQuery.data?.data ?? []} />
          </TabsContent>
          <TabsContent value="lineage">
            <Card>
              <CardContent className="space-y-4 py-4">
                <div className="text-sm">
                  Upstream source: {lineageQuery.data?.source?.name ?? '—'}
                </div>
                <div className="text-sm">
                  Source table: {lineageQuery.data?.source_table?.name ?? '—'}
                </div>
                <div className="text-sm">
                  Consumers: {lineageQuery.data?.consumers?.length ?? 0}
                </div>
                <Button variant="outline" size="sm" asChild>
                  <Link href={`/data/lineage?type=data_model&id=${model.id}`}>Open full lineage</Link>
                </Button>
              </CardContent>
            </Card>
          </TabsContent>
          <TabsContent value="versions">
            <ModelVersionHistory versions={versionsQuery.data ?? []} currentModelId={model.id} />
          </TabsContent>
        </Tabs>
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
