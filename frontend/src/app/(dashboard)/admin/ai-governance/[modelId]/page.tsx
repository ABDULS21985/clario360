'use client';

import { useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeftRight } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { RelativeTime } from '@/components/shared/relative-time';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { AIModelVersion, AIPredictionStats, AIRegisteredModel } from '@/types/ai-governance';
import { VersionTimeline } from '../_components/version-timeline';
import { PredictionLogTable } from '../_components/prediction-log-table';
import { ShadowComparisonChart } from '../_components/shadow-comparison-chart';
import { DriftChart } from '../_components/drift-chart';
import { PerformanceChart } from '../_components/performance-chart';
import { PromoteDialog } from '../_components/promote-dialog';
import { RollbackDialog } from '../_components/rollback-dialog';

export default function AIModelDetailPage() {
  const params = useParams<{ modelId: string }>();
  const modelId = params.modelId;

  const [busyVersionId, setBusyVersionId] = useState<string | null>(null);
  const [promoteTarget, setPromoteTarget] = useState<AIModelVersion | null>(null);
  const [rollbackOpen, setRollbackOpen] = useState(false);

  const modelQuery = useQuery({
    queryKey: ['ai-model', modelId],
    queryFn: () => enterpriseApi.ai.getModel(modelId),
  });
  const versionsQuery = useQuery({
    queryKey: ['ai-model-versions', modelId],
    queryFn: () => enterpriseApi.ai.listVersions(modelId),
  });
  const historyQuery = useQuery({
    queryKey: ['ai-model-history', modelId],
    queryFn: () => enterpriseApi.ai.lifecycleHistory(modelId),
  });
  const latestComparisonQuery = useQuery({
    queryKey: ['ai-model-shadow-latest', modelId],
    queryFn: () => enterpriseApi.ai.latestComparison(modelId),
  });
  const comparisonHistoryQuery = useQuery({
    queryKey: ['ai-model-shadow-history', modelId],
    queryFn: () => enterpriseApi.ai.comparisonHistory(modelId),
  });
  const divergencesQuery = useQuery({
    queryKey: ['ai-model-divergences', modelId],
    queryFn: () =>
      enterpriseApi.ai.divergences(modelId, {
        page: 1,
        per_page: 10,
        sort: 'created_at',
        order: 'desc',
      }),
  });
  const latestDriftQuery = useQuery({
    queryKey: ['ai-model-drift-latest', modelId],
    queryFn: () => enterpriseApi.ai.latestDrift(modelId),
  });
  const driftHistoryQuery = useQuery({
    queryKey: ['ai-model-drift-history', modelId],
    queryFn: () => enterpriseApi.ai.driftHistory(modelId),
  });
  const performanceQuery = useQuery({
    queryKey: ['ai-model-performance', modelId],
    queryFn: () => enterpriseApi.ai.performance(modelId),
  });
  const predictionStatsQuery = useQuery({
    queryKey: ['ai-prediction-stats', modelId],
    queryFn: () => enterpriseApi.ai.predictionStats(),
  });

  const model = modelQuery.data?.model ?? null;
  const productionVersion = modelQuery.data?.production_version ?? null;
  const versions = versionsQuery.data ?? [];
  const lifecycleHistory = historyQuery.data ?? [];
  const predictionStats = useMemo(
    () => (predictionStatsQuery.data ?? []).filter((item) => item.model_id === modelId),
    [predictionStatsQuery.data, modelId],
  );
  const feedbackSummary = summarizeFeedback(predictionStats);

  const refreshAll = async () => {
    await Promise.all([
      modelQuery.refetch(),
      versionsQuery.refetch(),
      historyQuery.refetch(),
      latestComparisonQuery.refetch(),
      comparisonHistoryQuery.refetch(),
      divergencesQuery.refetch(),
      latestDriftQuery.refetch(),
      driftHistoryQuery.refetch(),
      performanceQuery.refetch(),
      predictionStatsQuery.refetch(),
    ]);
  };

  const startShadow = async (version: AIModelVersion) => {
    try {
      setBusyVersionId(version.id);
      await enterpriseApi.ai.startShadow(modelId, { version_id: version.id });
      showSuccess('Shadow mode started.', `${model?.slug ?? 'model'} v${version.version_number} is now in shadow.`);
      await refreshAll();
    } catch (error) {
      showApiError(error);
    } finally {
      setBusyVersionId(null);
    }
  };

  return (
    <PermissionRedirect permission="users:read">
      <div className="space-y-6">
        <PageHeader
          title={model?.name ?? 'AI Model'}
          description={model?.description ?? 'Governance detail for a registered model.'}
          actions={
            <div className="flex items-center gap-2">
              {productionVersion ? (
                <Button variant="outline" onClick={() => setRollbackOpen(true)}>
                  <ArrowLeftRight className="mr-1.5 h-3.5 w-3.5" />
                  Rollback
                </Button>
              ) : null}
              <Button variant="outline" onClick={() => void refreshAll()}>
                Refresh
              </Button>
            </div>
          }
        />

        {model ? (
          <Card className="overflow-hidden border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(30,64,175,0.12),_transparent_40%),radial-gradient(circle_at_bottom_right,_rgba(180,83,9,0.12),_transparent_36%)]">
            <CardContent className="grid gap-4 p-6 md:grid-cols-4">
              <div>
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Suite</p>
                <div className="mt-2"><Badge variant="secondary">{model.suite}</Badge></div>
              </div>
              <div>
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Type</p>
                <div className="mt-2"><Badge variant="outline">{model.model_type.replaceAll('_', ' ')}</Badge></div>
              </div>
              <div>
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Risk Tier</p>
                <div className="mt-2"><Badge variant={model.risk_tier === 'critical' ? 'destructive' : model.risk_tier === 'high' ? 'warning' : 'secondary'}>{model.risk_tier}</Badge></div>
              </div>
              <div>
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Owner</p>
                <div className="mt-2 text-sm font-medium">{model.owner_team || model.owner_user_id || 'Unassigned'}</div>
              </div>
            </CardContent>
          </Card>
        ) : null}

        <Tabs defaultValue="versions" className="space-y-4">
          <TabsList className="grid w-full grid-cols-3 md:grid-cols-6">
            <TabsTrigger value="versions">Versions</TabsTrigger>
            <TabsTrigger value="predictions">Predictions</TabsTrigger>
            <TabsTrigger value="shadow">Shadow</TabsTrigger>
            <TabsTrigger value="drift">Drift</TabsTrigger>
            <TabsTrigger value="performance">Performance</TabsTrigger>
            <TabsTrigger value="feedback">Feedback</TabsTrigger>
          </TabsList>

          <TabsContent value="versions" className="space-y-4">
            <VersionTimeline
              versions={versions}
              history={lifecycleHistory}
              busyVersionId={busyVersionId}
              onPromote={(version) => setPromoteTarget(version)}
              onStartShadow={(version) => void startShadow(version)}
            />
          </TabsContent>

          <TabsContent value="predictions" className="space-y-4">
            <PredictionLogTable modelId={modelId} />
          </TabsContent>

          <TabsContent value="shadow" className="space-y-4" id="shadow">
            <ShadowComparisonChart
              latest={latestComparisonQuery.data ?? null}
              history={comparisonHistoryQuery.data ?? []}
            />
            <Card className="border-border/70">
              <CardHeader>
                <CardTitle>Divergence Samples</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {(divergencesQuery.data?.data ?? []).map((item) => (
                  <div key={item.prediction_id} className="rounded-lg border border-border/70 p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div className="font-medium">{item.use_case}</div>
                      <RelativeTime date={item.created_at} className="text-xs text-muted-foreground" />
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">{item.reason}</p>
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="drift" className="space-y-4">
            <DriftChart latest={latestDriftQuery.data ?? null} history={driftHistoryQuery.data ?? []} />
          </TabsContent>

          <TabsContent value="performance" className="space-y-4">
            <PerformanceChart points={performanceQuery.data ?? []} />
          </TabsContent>

          <TabsContent value="feedback" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <Card className="border-border/70">
                <CardHeader><CardTitle>Correct Feedback</CardTitle></CardHeader>
                <CardContent className="text-3xl font-semibold">{feedbackSummary.correct.toLocaleString()}</CardContent>
              </Card>
              <Card className="border-border/70">
                <CardHeader><CardTitle>Incorrect Feedback</CardTitle></CardHeader>
                <CardContent className="text-3xl font-semibold">{feedbackSummary.incorrect.toLocaleString()}</CardContent>
              </Card>
              <Card className="border-border/70">
                <CardHeader><CardTitle>Accuracy from Feedback</CardTitle></CardHeader>
                <CardContent className="text-3xl font-semibold">{feedbackSummary.accuracy}%</CardContent>
              </Card>
            </div>
            <Card className="border-border/70">
              <CardHeader>
                <CardTitle>Use Cases</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {predictionStats.map((item: AIPredictionStats) => (
                  <div key={`${item.use_case}-${item.model_id}`} className="flex items-center justify-between rounded-lg border border-border/70 p-4">
                    <div>
                      <div className="font-medium">{item.use_case}</div>
                      <div className="text-sm text-muted-foreground">
                        {item.total.toLocaleString()} prod predictions • {item.shadow_total.toLocaleString()} shadow
                      </div>
                    </div>
                    <div className="text-right text-sm">
                      <div>Correct {item.correct_feedback}</div>
                      <div className="text-muted-foreground">Incorrect {item.wrong_feedback}</div>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      <PromoteDialog
        open={Boolean(promoteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setPromoteTarget(null);
          }
        }}
        model={model as AIRegisteredModel | null}
        version={promoteTarget}
        onSaved={() => {
          void refreshAll();
        }}
      />

      <RollbackDialog
        open={rollbackOpen}
        onOpenChange={setRollbackOpen}
        model={model as AIRegisteredModel | null}
        onSaved={() => {
          void refreshAll();
        }}
      />
    </PermissionRedirect>
  );
}

function summarizeFeedback(items: AIPredictionStats[]) {
  const summary = items.reduce(
    (acc, item) => {
      acc.correct += item.correct_feedback;
      acc.incorrect += item.wrong_feedback;
      return acc;
    },
    { correct: 0, incorrect: 0 },
  );
  const total = summary.correct + summary.incorrect;
  return {
    ...summary,
    accuracy: total === 0 ? 0 : Math.round((summary.correct / total) * 100),
  };
}
