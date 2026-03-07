'use client';

import { useState } from 'react';
import { useQueries } from '@tanstack/react-query';
import { Plus, ShieldCheck } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { QualityModelCards } from '@/app/(dashboard)/data/quality/_components/quality-model-cards';
import { buildQualityRuleColumns } from '@/app/(dashboard)/data/quality/_components/quality-rule-columns';
import {
  buildQualityRulePayload,
  QualityRuleForm,
} from '@/app/(dashboard)/data/quality/_components/quality-rule-form';
import { QualityResultDialog } from '@/app/(dashboard)/data/quality/_components/quality-result-dialog';
import { QualityScoreGauge } from '@/app/(dashboard)/data/quality/_components/quality-score-gauge';
import { QualityTrendChart } from '@/app/(dashboard)/data/quality/_components/quality-trend-chart';
import { dataSuiteApi, type QualityResult, type QualityRule } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';

const QUALITY_RULE_FILTERS = [
  {
    key: 'severity',
    label: 'Severity',
    type: 'multi-select' as const,
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
    ],
  },
  {
    key: 'status',
    label: 'Last Status',
    type: 'multi-select' as const,
    options: [
      { label: 'Passed', value: 'passed' },
      { label: 'Failed', value: 'failed' },
      { label: 'Warning', value: 'warning' },
      { label: 'Error', value: 'error' },
    ],
  },
];

export default function DataQualityPage() {
  const [runningId, setRunningId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [selectedResult, setSelectedResult] = useState<QualityResult | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<QualityRule | null>(null);
  const [submittingRule, setSubmittingRule] = useState(false);

  const { tableProps, searchValue, setSearch, refetch } = useDataTable<QualityRule>({
    queryKey: 'data-quality-rules',
    fetchFn: (params) => dataSuiteApi.listQualityRules(params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
    wsTopics: ['quality.check_failed'],
  });

  const [dashboardQuery, scoreQuery, trendQuery, modelsQuery, sourcesQuery] = useQueries({
    queries: [
      { queryKey: ['data-quality-dashboard'], queryFn: () => dataSuiteApi.getQualityDashboard() },
      { queryKey: ['data-quality-score'], queryFn: () => dataSuiteApi.getQualityScore() },
      { queryKey: ['data-quality-trend'], queryFn: () => dataSuiteApi.getQualityTrend(30) },
      {
        queryKey: ['data-quality-models'],
        queryFn: () =>
          dataSuiteApi.listModels({ page: 1, per_page: 200, sort: 'name', order: 'asc' }),
      },
      {
        queryKey: ['data-quality-sources'],
        queryFn: () =>
          dataSuiteApi.listSources({ page: 1, per_page: 200, sort: 'name', order: 'asc' }),
      },
    ],
  });

  const isLoading = [dashboardQuery, scoreQuery, trendQuery, modelsQuery, sourcesQuery].some((query) => query.isLoading);
  const error = [dashboardQuery, scoreQuery, trendQuery, modelsQuery, sourcesQuery].find((query) => query.error)?.error;

  const runRule = async (rule: QualityRule) => {
    try {
      setRunningId(rule.id);
      const result = await dataSuiteApi.runQualityRule(rule.id);
      setSelectedResult(result);
      showSuccess('Quality rule executed.', `${rule.name} finished with status ${result.status}.`);
      void refetch();
    } catch (err) {
      showApiError(err);
    } finally {
      setRunningId(null);
    }
  };

  const toggleRule = async (rule: QualityRule, enabled: boolean) => {
    try {
      setTogglingId(rule.id);
      await dataSuiteApi.updateQualityRule(rule.id, { enabled });
      showSuccess(enabled ? 'Rule enabled.' : 'Rule disabled.');
      void refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setTogglingId(null);
    }
  };

  const submitRule = async (values: Parameters<typeof buildQualityRulePayload>[0]) => {
    try {
      setSubmittingRule(true);
      const payload = buildQualityRulePayload(values);
      if (editingRule) {
        await dataSuiteApi.updateQualityRule(editingRule.id, {
          name: payload.name,
          description: payload.description,
          severity: payload.severity,
          column_name: payload.column_name || null,
          config: payload.config,
          schedule: payload.schedule || null,
          enabled: payload.enabled,
          tags: payload.tags,
        });
        showSuccess('Quality rule updated.');
      } else {
        await dataSuiteApi.createQualityRule({
          model_id: payload.model_id,
          name: payload.name,
          description: payload.description,
          rule_type: payload.rule_type,
          severity: payload.severity,
          column_name: payload.column_name || null,
          config: payload.config,
          schedule: payload.schedule || null,
          enabled: payload.enabled,
          tags: payload.tags,
        });
        showSuccess('Quality rule created.');
      }
      setFormOpen(false);
      setEditingRule(null);
      void refetch();
      void dashboardQuery.refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setSubmittingRule(false);
    }
  };

  if (isLoading || !scoreQuery.data) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Data Quality" description="Loading score, trend, and live rule telemetry." />
          <LoadingSkeleton variant="card" />
          <LoadingSkeleton variant="chart" />
        </div>
      </PermissionRedirect>
    );
  }

  if (error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message={error instanceof Error ? error.message : 'Failed to load quality metrics.'} onRetry={() => void scoreQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Quality"
          description="Live quality posture across governed models, rule execution, and recent trend movement."
          actions={
            <Button
              type="button"
              onClick={() => {
                setEditingRule(null);
                setFormOpen(true);
              }}
            >
              <Plus className="mr-2 h-4 w-4" />
              Create rule
            </Button>
          }
        />

        <div className="grid gap-4 xl:grid-cols-[0.42fr_0.58fr]">
          <QualityScoreGauge score={scoreQuery.data} />
          <div className="rounded-lg border bg-card p-4">
            <div className="mb-4 flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-primary" />
              <h3 className="font-medium">30-Day Trend</h3>
            </div>
            <QualityTrendChart trend={trendQuery.data ?? []} />
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-semibold">Model Quality Scores</h3>
          <QualityModelCards items={scoreQuery.data.model_scores} />
        </div>

        <DataTable
          {...tableProps}
          columns={buildQualityRuleColumns({
            runningId,
            togglingId,
            onRun: (rule) => void runRule(rule),
            onEdit: (rule) => {
              setEditingRule(rule);
              setFormOpen(true);
            },
            onToggleEnabled: (rule, enabled) => void toggleRule(rule, enabled),
          })}
          filters={QUALITY_RULE_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search quality rules..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: ShieldCheck,
            title: 'No quality rules found',
            description: 'No quality rules matched the current filters.',
          }}
        />

        <QualityResultDialog
          open={Boolean(selectedResult)}
          onOpenChange={(open) => {
            if (!open) {
              setSelectedResult(null);
            }
          }}
          result={selectedResult}
        />

        <QualityRuleForm
          open={formOpen}
          onOpenChange={(open) => {
            setFormOpen(open);
            if (!open) {
              setEditingRule(null);
            }
          }}
          models={modelsQuery.data?.data ?? []}
          sources={sourcesQuery.data?.data ?? []}
          rule={editingRule}
          submitting={submittingRule}
          onSubmit={(values) => void submitRule(values)}
        />
      </div>
    </PermissionRedirect>
  );
}
