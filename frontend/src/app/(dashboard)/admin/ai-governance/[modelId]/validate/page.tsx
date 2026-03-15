'use client';

import { ChangeEvent, useEffect, useState } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, Play, RefreshCw, ShieldCheck, SplitSquareVertical } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { enterpriseApi } from '@/lib/enterprise';
import {
  downloadBlob,
  formatDateTime,
  formatNumber,
  formatPercentage,
  parseApiError,
  titleCase,
} from '@/lib/format';
import { showApiError, showSuccess, showWarning } from '@/lib/toast';
import type {
  AIModelVersion,
  AIRegisteredModel,
  AIValidationLabel,
  AIValidationMetricsSummary,
  AIValidationPreview,
  AIValidationResult,
} from '@/types/ai-governance';
import { ComparisonIndicator } from './_components/comparison-indicator';
import { ConfusionMatrix } from './_components/confusion-matrix';
import { DatasetSelector } from './_components/dataset-selector';
import { FNSampleTable } from './_components/fn-sample-table';
import { FPSampleTable } from './_components/fp-sample-table';
import { MetricsCards } from './_components/metrics-cards';
import { RecommendationBanner } from './_components/recommendation-banner';
import { ROCCurveChart } from './_components/roc-curve-chart';
import { SeverityBreakdownTable } from './_components/severity-breakdown-table';

type CustomValidationRow = {
  input_hash: string;
  expected_label: AIValidationLabel;
};

type ExportFormat = 'json' | 'markdown';

export default function AIModelValidationPage() {
  const params = useParams<{ modelId: string }>();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const modelId = params?.modelId ?? '';

  const [selectedVersionId, setSelectedVersionId] = useState('');
  const [datasetType, setDatasetType] = useState<'historical' | 'custom' | 'live_replay'>('historical');
  const [timeRange, setTimeRange] = useState('30d');
  const [customText, setCustomText] = useState('');
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [showPreviousDiff, setShowPreviousDiff] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);
  const [exportFormat, setExportFormat] = useState<ExportFormat>('json');

  const modelQuery = useQuery({
    queryKey: ['ai-model', modelId],
    enabled: Boolean(modelId),
    queryFn: () => enterpriseApi.ai.getModel(modelId),
  });

  const versionsQuery = useQuery({
    queryKey: ['ai-model-versions', modelId],
    enabled: Boolean(modelId),
    queryFn: () => enterpriseApi.ai.listVersions(modelId),
  });

  useEffect(() => {
    if (selectedVersionId || !versionsQuery.data?.length) {
      return;
    }
    const requestedVersionId = searchParams?.get('versionId');
    const fallbackVersion =
      versionsQuery.data.find((item) => item.id === requestedVersionId) ?? versionsQuery.data[0];
    setSelectedVersionId(fallbackVersion.id);
  }, [selectedVersionId, searchParams, versionsQuery.data]);

  useEffect(() => {
    setShowPreviousDiff(false);
    setRejectOpen(false);
    setRejectReason('');
    setExportOpen(false);
    setExportFormat('json');
  }, [selectedVersionId]);

  const selectedVersion = versionsQuery.data?.find((item) => item.id === selectedVersionId) ?? null;
  const customParse = parseCustomData(customText);
  const customRowCount = customParse.data?.length ?? 0;

  const previewQuery = useQuery({
    queryKey: ['ai-validation-preview', modelId, selectedVersionId, datasetType, timeRange, customText],
    enabled:
      Boolean(selectedVersionId) &&
      datasetType !== 'live_replay' &&
      (datasetType !== 'custom' || (customParse.data !== null && customRowCount > 0)),
    queryFn: () =>
      enterpriseApi.ai.previewValidation(modelId, selectedVersionId, {
        dataset_type: datasetType,
        time_range: datasetType === 'historical' ? timeRange : undefined,
        custom_data: datasetType === 'custom' ? customParse.data : undefined,
      }),
    retry: false,
  });

  const latestValidationQuery = useQuery({
    queryKey: ['ai-validation-latest', modelId, selectedVersionId],
    enabled: Boolean(selectedVersionId),
    queryFn: async () => {
      try {
        return await enterpriseApi.ai.latestValidation(modelId, selectedVersionId);
      } catch (error) {
        if (isNotFound(error)) {
          return null;
        }
        throw error;
      }
    },
  });

  const historyQuery = useQuery({
    queryKey: ['ai-validation-history', modelId, selectedVersionId],
    enabled: Boolean(selectedVersionId),
    queryFn: async () => {
      try {
        return await enterpriseApi.ai.validationHistory(modelId, selectedVersionId, 8);
      } catch (error) {
        if (isNotFound(error)) {
          return [];
        }
        throw error;
      }
    },
  });

  const runValidationMutation = useMutation({
    mutationFn: () =>
      enterpriseApi.ai.validate(modelId, selectedVersionId, {
        dataset_type: datasetType,
        time_range: datasetType === 'historical' ? timeRange : undefined,
        custom_data: datasetType === 'custom' ? customParse.data : undefined,
      }),
    onSuccess: (nextResult) => {
      queryClient.setQueryData(['ai-validation-latest', modelId, selectedVersionId], nextResult);
      void queryClient.invalidateQueries({ queryKey: ['ai-validation-history', modelId, selectedVersionId] });
      void queryClient.invalidateQueries({ queryKey: ['ai-model', modelId] });
      void queryClient.invalidateQueries({ queryKey: ['ai-model-versions', modelId] });
      showSuccess(
        'Validation completed.',
        `${selectedVersion?.model_slug ?? 'Model'} v${selectedVersion?.version_number ?? ''} has new validation results.`,
      );
    },
    onError: showApiError,
  });

  const promoteShadowMutation = useMutation({
    mutationFn: () => enterpriseApi.ai.startShadow(modelId, { version_id: selectedVersionId }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ai-model', modelId] });
      void queryClient.invalidateQueries({ queryKey: ['ai-model-versions', modelId] });
      showSuccess('Shadow mode started.', 'The validated version is now running in shadow mode.');
    },
    onError: showApiError,
  });

  const failVersionMutation = useMutation({
    mutationFn: (reason: string) => enterpriseApi.ai.failVersion(modelId, selectedVersionId, { reason }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ai-model', modelId] });
      void queryClient.invalidateQueries({ queryKey: ['ai-model-versions', modelId] });
      setRejectOpen(false);
      setRejectReason('');
      showSuccess('Version marked as failed.', 'The version status has been updated and the rejection notes were saved.');
    },
    onError: showApiError,
  });

  const model = modelQuery.data?.model ?? null;
  const result = latestValidationQuery.data ?? null;
  const preview = previewQuery.data ?? null;
  const previewError = previewQuery.isError ? parseApiError(previewQuery.error) : null;
  const validationHistory = historyQuery.data ?? [];
  const previousValidation = result
    ? validationHistory.find((item) => item.id !== result.id) ?? null
    : validationHistory[0] ?? null;
  const hasRuleTypeBreakdown = Boolean(result?.by_rule_type && Object.keys(result.by_rule_type).length > 0);

  const runBlockedReason = validationBlockReason({
    selectedVersionId,
    datasetType,
    customParseError: customParse.error,
    customRowCount,
    preview,
    previewLoading: previewQuery.isFetching,
    previewError,
  });
  const rejectBlockedReason = failureBlockReason(selectedVersion);
  const shadowBlockedReason = shadowPromotionBlockReason(selectedVersion, result);

  const refreshPage = async () => {
    await Promise.all([
      modelQuery.refetch(),
      versionsQuery.refetch(),
      latestValidationQuery.refetch(),
      historyQuery.refetch(),
      previewQuery.refetch(),
    ]);
  };

  return (
    <PermissionRedirect permission="users:read">
      <div className="space-y-6">
        <PageHeader
          title={model?.name ?? 'Model Validation'}
          description="Evaluate precision, recall, F1, false-positive rate, ROC behavior, and promotion readiness against labeled data."
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <Select value={selectedVersionId} onValueChange={setSelectedVersionId}>
                <SelectTrigger className="min-w-[220px]">
                  <SelectValue placeholder="Select a version" />
                </SelectTrigger>
                <SelectContent>
                  {(versionsQuery.data ?? []).map((version) => (
                    <SelectItem key={version.id} value={version.id}>
                      v{version.version_number} · {titleCase(version.status)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button variant="outline" onClick={() => void refreshPage()}>
                <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                Refresh
              </Button>
              <Button
                onClick={() => {
                  if (runBlockedReason) {
                    showWarning('Validation blocked.', runBlockedReason);
                    return;
                  }
                  runValidationMutation.mutate();
                }}
                disabled={!selectedVersionId || runValidationMutation.isPending || Boolean(runBlockedReason)}
              >
                <Play className="mr-1.5 h-3.5 w-3.5" />
                Run Validation
              </Button>
            </div>
          }
        />

        {selectedVersion ? (
          <Card className="overflow-hidden border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.14),_transparent_36%),radial-gradient(circle_at_bottom_right,_rgba(14,116,144,0.12),_transparent_38%)]">
            <CardContent className="grid grid-cols-1 gap-4 p-4 sm:p-6 md:grid-cols-4">
              <div>
                <div className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Version</div>
                <div className="mt-2 text-3xl font-semibold tracking-[-0.05em] text-slate-950">
                  v{selectedVersion.version_number}
                </div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Status</div>
                <div className="mt-2">
                  <Badge variant={versionBadgeVariant(selectedVersion.status)}>
                    {titleCase(selectedVersion.status)}
                  </Badge>
                </div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Artifact</div>
                <div className="mt-2 text-sm font-medium">{titleCase(selectedVersion.artifact_type)}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Last Validation</div>
                <div className="mt-2 text-sm font-medium">
                  {result ? formatDateTime(result.validated_at) : 'No validation recorded'}
                </div>
              </div>
            </CardContent>
          </Card>
        ) : null}

        <DatasetSelector
          datasetType={datasetType}
          timeRange={timeRange}
          customText={customText}
          customParseError={customParse.error}
          preview={preview}
          previewError={previewError}
          previewLoading={previewQuery.isFetching}
          onDatasetTypeChange={setDatasetType}
          onTimeRangeChange={setTimeRange}
          onCustomTextChange={setCustomText}
          onCustomFileLoad={(event) => handleCustomFile(event, setCustomText)}
        />

        {runBlockedReason ? (
          <Alert variant="warning">
            <AlertTitle>Validation is blocked</AlertTitle>
            <AlertDescription>{runBlockedReason}</AlertDescription>
          </Alert>
        ) : null}

        {result ? (
          <div className="space-y-6">
            {result.warnings.map((warning) => (
              <Alert key={warning} variant="warning">
                <AlertTitle>Validation Warning</AlertTitle>
                <AlertDescription>{warning}</AlertDescription>
              </Alert>
            ))}

            <MetricsCards result={result} />

            <div className="grid grid-cols-1 gap-4 xl:grid-cols-[0.95fr_1.05fr]">
              <ConfusionMatrix result={result} />
              <ROCCurveChart result={result} />
            </div>

            <div className={`grid gap-4 ${hasRuleTypeBreakdown ? 'xl:grid-cols-2' : ''}`}>
              <SeverityBreakdownTable breakdown={result.by_severity} />
              {hasRuleTypeBreakdown ? (
                <SeverityBreakdownTable
                  title="Rule Type Breakdown"
                  label="Rule Type"
                  breakdown={result.by_rule_type ?? {}}
                />
              ) : null}
            </div>

            <FPSampleTable samples={result.false_positive_samples} />
            <FNSampleTable samples={result.false_negative_samples} />

            <RecommendationBanner result={result} />

            <Card className="border-border/70">
              <CardHeader>
                <CardTitle>Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex flex-wrap items-center gap-3">
                  <Button
                    onClick={() => {
                      if (shadowBlockedReason) {
                        showWarning('Promotion blocked.', shadowBlockedReason);
                        return;
                      }
                      promoteShadowMutation.mutate();
                    }}
                    disabled={Boolean(shadowBlockedReason) || promoteShadowMutation.isPending}
                  >
                    <ShieldCheck className="mr-1.5 h-3.5 w-3.5" />
                    Promote to Shadow Mode
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setRejectOpen(true)}
                    disabled={Boolean(rejectBlockedReason)}
                  >
                    <SplitSquareVertical className="mr-1.5 h-3.5 w-3.5" />
                    Reject - Needs Improvement
                  </Button>
                  <Button variant="outline" onClick={() => setExportOpen(true)}>
                    <Download className="mr-1.5 h-3.5 w-3.5" />
                    Export Validation Report
                  </Button>
                  <Button
                    variant="outline"
                    disabled={!previousValidation}
                    onClick={() => setShowPreviousDiff((current) => !current)}
                  >
                    Compare with Previous Validation
                  </Button>
                </div>
                {(shadowBlockedReason || rejectBlockedReason) ? (
                  <div className="space-y-1 text-sm text-muted-foreground">
                    {shadowBlockedReason ? <div>Shadow promotion: {shadowBlockedReason}</div> : null}
                    {rejectBlockedReason ? <div>Rejection flow: {rejectBlockedReason}</div> : null}
                  </div>
                ) : null}
              </CardContent>
            </Card>

            {showPreviousDiff && previousValidation ? (
              <Card className="border-border/70">
                <CardHeader>
                  <CardTitle>Previous Validation Diff</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="text-sm text-muted-foreground">
                    Comparing against validation from {formatDateTime(previousValidation.validated_at)}.
                  </div>
                  <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                    {[
                      ['Precision', result.precision - previousValidation.precision, false],
                      ['Recall', result.recall - previousValidation.recall, false],
                      ['F1 Score', result.f1_score - previousValidation.f1_score, false],
                      ['FP Rate', result.false_positive_rate - previousValidation.false_positive_rate, true],
                    ].map(([label, delta, inverse]) => (
                      <div key={label as string} className="rounded-2xl border border-border/70 bg-slate-50/70 p-4">
                        <div className="text-sm font-medium text-slate-700">{label as string}</div>
                        <div className="mt-3">
                          <ComparisonIndicator
                            delta={delta as number}
                            inverse={inverse as boolean}
                            label="vs previous validation"
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            ) : null}
          </div>
        ) : (
          <Card className="border-border/70">
            <CardContent className="p-4 text-sm text-muted-foreground sm:p-6">
              No validation result has been recorded for this version yet.
            </CardContent>
          </Card>
        )}
      </div>

      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reject Version</DialogTitle>
            <DialogDescription>
              Mark this version as failed and store the remediation notes with the lifecycle record.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="reject-reason">Improvement Notes</Label>
            <Textarea
              id="reject-reason"
              value={rejectReason}
              onChange={(event) => setRejectReason(event.target.value)}
              rows={5}
              placeholder="Document the metric regressions or follow-up work required."
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => failVersionMutation.mutate(rejectReason.trim())}
              disabled={Boolean(rejectBlockedReason) || !rejectReason.trim() || failVersionMutation.isPending}
            >
              Save and Mark Failed
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={exportOpen} onOpenChange={setExportOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Export Validation Report</DialogTitle>
            <DialogDescription>
              Download the latest validation as raw JSON or as a structured markdown report.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Format</Label>
            <Select value={exportFormat} onValueChange={(value) => setExportFormat(value as ExportFormat)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="json">JSON</SelectItem>
                <SelectItem value="markdown">Structured report</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setExportOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                if (!result) {
                  showWarning('Export unavailable.', 'Run a validation first.');
                  return;
                }
                exportValidationReport(model, selectedVersion, result, previousValidation, exportFormat);
                setExportOpen(false);
              }}
            >
              Download
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PermissionRedirect>
  );
}

function parseCustomData(raw: string): { data: CustomValidationRow[] | null; error: string | null } {
  if (!raw.trim()) {
    return { data: [], error: null };
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return { data: null, error: 'Custom data must be a JSON array.' };
    }
    const data = parsed.map((item, index) => {
      const record = item as Record<string, unknown>;
      const inputHash = String(record.input_hash ?? '').trim();
      const expectedLabel = String(record.expected_label ?? '').trim();
      if (!inputHash) {
        throw new Error(`Row ${index + 1} must include a non-empty input_hash.`);
      }
      if (expectedLabel !== 'threat' && expectedLabel !== 'benign') {
        throw new Error(`Row ${index + 1} expected_label must be "threat" or "benign".`);
      }
      return {
        input_hash: inputHash,
        expected_label: expectedLabel,
      } satisfies CustomValidationRow;
    });
    return { data, error: null };
  } catch (error) {
    return { data: null, error: error instanceof Error ? error.message : 'Invalid JSON.' };
  }
}

function validationBlockReason(args: {
  selectedVersionId: string;
  datasetType: 'historical' | 'custom' | 'live_replay';
  customParseError: string | null;
  customRowCount: number;
  preview: AIValidationPreview | null;
  previewLoading: boolean;
  previewError: string | null;
}) {
  if (!args.selectedVersionId) {
    return 'Select a model version before running validation.';
  }
  if (args.datasetType === 'live_replay') {
    return 'Live replay is unavailable in the current deployment because model-version replay execution is not configured.';
  }
  if (args.customParseError) {
    return args.customParseError;
  }
  if (args.datasetType === 'custom' && args.customRowCount === 0) {
    return 'Provide custom labeled data before running validation.';
  }
  if (args.previewLoading) {
    return 'Dataset preview is still loading.';
  }
  if (args.previewError) {
    return args.previewError;
  }
  if (!args.preview) {
    return 'Dataset preview is not available yet.';
  }
  if (args.preview.dataset_size < 50) {
    return 'Insufficient labeled data. Need at least 50 samples.';
  }
  return null;
}

function failureBlockReason(version: AIModelVersion | null) {
  if (!version) {
    return 'Select a model version before rejecting it.';
  }
  switch (version.status) {
    case 'production':
      return 'Production versions must be rolled back or retired instead of marked failed.';
    case 'failed':
      return 'This version is already marked failed.';
    case 'retired':
    case 'rolled_back':
      return `This version is already ${titleCase(version.status)}.`;
    default:
      return null;
  }
}

function shadowPromotionBlockReason(version: AIModelVersion | null, result: AIValidationResult | null) {
  if (!version) {
    return 'Select a model version before promoting it.';
  }
  if (!result) {
    return 'Run validation before promoting to shadow mode.';
  }
  if (result.recommendation !== 'promote') {
    return 'Only versions with a promote recommendation can move into shadow mode.';
  }
  switch (version.status) {
    case 'shadow':
      return 'This version is already running in shadow mode.';
    case 'production':
      return 'This version is already in production.';
    case 'failed':
    case 'retired':
    case 'rolled_back':
      return `Versions in ${titleCase(version.status)} state cannot enter shadow mode.`;
    default:
      return null;
  }
}

function versionBadgeVariant(status: AIModelVersion['status']) {
  switch (status) {
    case 'production':
      return 'success';
    case 'shadow':
    case 'staging':
      return 'warning';
    case 'failed':
    case 'retired':
      return 'destructive';
    default:
      return 'outline';
  }
}

function handleCustomFile(event: ChangeEvent<HTMLInputElement>, setCustomText: (value: string) => void) {
  const file = event.target.files?.[0];
  if (!file) {
    return;
  }
  const reader = new FileReader();
  reader.onload = () => {
    const text = typeof reader.result === 'string' ? reader.result : '';
    setCustomText(text);
  };
  reader.readAsText(file);
}

function exportValidationReport(
  model: AIRegisteredModel | null,
  version: AIModelVersion | null,
  result: AIValidationResult,
  previousValidation: AIValidationResult | null,
  format: ExportFormat,
) {
  if (format === 'json') {
    const filename = `model-validation-${model?.slug ?? 'model'}-v${version?.version_number ?? 'latest'}.json`;
    downloadBlob(new Blob([JSON.stringify(result, null, 2)], { type: 'application/json' }), filename);
    return;
  }

  const filename = `model-validation-${model?.slug ?? 'model'}-v${version?.version_number ?? 'latest'}.md`;
  const report = buildStructuredValidationReport(model, version, result, previousValidation);
  downloadBlob(new Blob([report], { type: 'text/markdown;charset=utf-8' }), filename);
}

function buildStructuredValidationReport(
  model: AIRegisteredModel | null,
  version: AIModelVersion | null,
  result: AIValidationResult,
  previousValidation: AIValidationResult | null,
) {
  const lines = [
    '# Model Validation Report',
    '',
    `- Model: ${model?.name ?? 'Unknown model'} (${model?.slug ?? 'unknown'})`,
    `- Version: v${version?.version_number ?? 'unknown'}`,
    `- Status: ${titleCase(version?.status ?? 'unknown')}`,
    `- Dataset Type: ${titleCase(result.dataset_type)}`,
    `- Validated At: ${formatDateTime(result.validated_at)}`,
    `- Duration: ${formatNumber(result.duration_ms)} ms`,
    `- Recommendation: ${titleCase(result.recommendation)}`,
    `- Recommendation Reason: ${result.recommendation_reason}`,
    '',
    '## Dataset Summary',
    '',
    `- Total Samples: ${formatNumber(result.dataset_size)}`,
    `- Positive Samples: ${formatNumber(result.positive_count)}`,
    `- Negative Samples: ${formatNumber(result.negative_count)}`,
    '',
    '## Metrics',
    '',
    '| Metric | Value | Delta vs Production | Delta vs Previous |',
    '| --- | --- | --- | --- |',
    `| Precision | ${formatPercentage(result.precision, 1)} | ${formatDelta(result.deltas?.precision)} | ${formatDelta(previousValidation ? result.precision - previousValidation.precision : null)} |`,
    `| Recall | ${formatPercentage(result.recall, 1)} | ${formatDelta(result.deltas?.recall)} | ${formatDelta(previousValidation ? result.recall - previousValidation.recall : null)} |`,
    `| F1 Score | ${formatPercentage(result.f1_score, 1)} | ${formatDelta(result.deltas?.f1_score)} | ${formatDelta(previousValidation ? result.f1_score - previousValidation.f1_score : null)} |`,
    `| False Positive Rate | ${formatPercentage(result.false_positive_rate, 1)} | ${formatDelta(result.deltas?.false_positive_rate)} | ${formatDelta(previousValidation ? result.false_positive_rate - previousValidation.false_positive_rate : null)} |`,
    `| Accuracy | ${formatPercentage(result.accuracy, 1)} | ${result.production_metrics ? formatPercentage(result.accuracy - result.production_metrics.accuracy, 1) : 'N/A'} | ${formatDelta(previousValidation ? result.accuracy - previousValidation.accuracy : null)} |`,
    `| AUC | ${formatPercentage(result.auc, 1)} | ${result.production_metrics?.auc !== undefined ? formatPercentage(result.auc - (result.production_metrics.auc ?? 0), 1) : 'N/A'} | ${formatDelta(previousValidation ? result.auc - previousValidation.auc : null)} |`,
    '',
    '## Confusion Matrix',
    '',
    '| TP | FP | FN | TN |',
    '| --- | --- | --- | --- |',
    `| ${formatNumber(result.true_positives)} | ${formatNumber(result.false_positives)} | ${formatNumber(result.false_negatives)} | ${formatNumber(result.true_negatives)} |`,
    '',
    '## Warnings',
    '',
  ];

  if (result.warnings.length === 0) {
    lines.push('- None');
  } else {
    for (const warning of result.warnings) {
      lines.push(`- ${warning}`);
    }
  }

  lines.push('', '## Severity Breakdown', '');
  lines.push(...renderBreakdownTable(result.by_severity));

  if (result.by_rule_type && Object.keys(result.by_rule_type).length > 0) {
    lines.push('', '## Rule Type Breakdown', '');
    lines.push(...renderBreakdownTable(result.by_rule_type));
  }

  lines.push('', '## Sample False Positives', '');
  lines.push(...renderSampleList(result.false_positive_samples));
  lines.push('', '## Sample False Negatives', '');
  lines.push(...renderSampleList(result.false_negative_samples));

  return `${lines.join('\n')}\n`;
}

function renderBreakdownTable(breakdown: Record<string, AIValidationMetricsSummary>) {
  const entries = Object.entries(breakdown);
  if (entries.length === 0) {
    return ['No breakdown data recorded.'];
  }
  return [
    '| Group | Precision | Recall | F1 | Count |',
    '| --- | --- | --- | --- | --- |',
    ...entries.map(
      ([key, metrics]) =>
        `| ${titleCase(key)} | ${formatPercentage(metrics.precision, 1)} | ${formatPercentage(metrics.recall, 1)} | ${formatPercentage(metrics.f1_score, 1)} | ${formatNumber(metrics.dataset_size)} |`,
    ),
  ];
}

function renderSampleList(samples: AIValidationResult['false_positive_samples']) {
  if (samples.length === 0) {
    return ['No samples recorded.'];
  }
  return samples.slice(0, 10).map((sample) => {
    const ruleType = sample.rule_type || 'Unknown';
    const severity = sample.severity || 'unclassified';
    return `- ${sample.input_hash} | ${titleCase(sample.predicted_label)} vs ${titleCase(sample.expected_label)} | ${formatPercentage(sample.confidence, 1)} | ${titleCase(ruleType)} | ${titleCase(severity)} | ${sample.explanation || 'No explanation available.'}`;
  });
}

function formatDelta(value: number | null | undefined) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'N/A';
  }
  const sign = value > 0 ? '+' : '';
  return `${sign}${formatPercentage(value, 1)}`;
}

function isNotFound(error: unknown) {
  return typeof error === 'object' && error !== null && 'status' in error && (error as { status?: number }).status === 404;
}
