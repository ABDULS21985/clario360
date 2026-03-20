'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Bot, Plus } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { Button } from '@/components/ui/button';
import { useDataTable } from '@/hooks/use-data-table';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { AIDashboardModelRow, AIModelVersion, AIRegisteredModel } from '@/types/ai-governance';
import { createModelColumns } from './_components/model-columns';
import { ModelCard } from './_components/model-card';
import { ModelFormDialog } from './_components/model-form-dialog';
import { PromoteDialog } from './_components/promote-dialog';
import { RollbackDialog } from './_components/rollback-dialog';

export default function AIGovernancePage() {
  const dashboardQuery = useQuery({
    queryKey: ['ai-dashboard'],
    queryFn: () => enterpriseApi.ai.getDashboard(),
  });

  const [busyModelId, setBusyModelId] = useState<string | null>(null);
  const [modelFormOpen, setModelFormOpen] = useState(false);
  const [promoteTarget, setPromoteTarget] = useState<{ model: AIRegisteredModel; version: AIModelVersion } | null>(null);
  const [rollbackTarget, setRollbackTarget] = useState<AIRegisteredModel | null>(null);

  const { tableProps, refetch } = useDataTable<AIDashboardModelRow>({
    queryKey: 'ai-models',
    fetchFn: (params) => enterpriseApi.ai.listModels(params).then((response) => ({
      data: response.data.map((item) => ({
        id: item.model.id,
        name: item.model.name,
        slug: item.model.slug,
        suite: item.model.suite,
        type: item.model.model_type,
        risk_tier: item.model.risk_tier,
        status: item.model.status,
        production_version: item.production_version ?? undefined,
        shadow_version: item.shadow_version ?? undefined,
        predictions_24h: dashboardQuery.data?.models.find((row) => row.id === item.model.id)?.predictions_24h ?? 0,
        avg_confidence: dashboardQuery.data?.models.find((row) => row.id === item.model.id)?.avg_confidence ?? undefined,
        drift_status: dashboardQuery.data?.models.find((row) => row.id === item.model.id)?.drift_status ?? 'none',
      })),
      meta: response.meta,
    })),
    defaultPageSize: 20,
    defaultSort: { column: 'name', direction: 'asc' },
  });

  const handlePromote = async (row: AIDashboardModelRow) => {
    const targetVersion = row.shadow_version;
    if (!targetVersion) {
      return;
    }
    setPromoteTarget({
      model: {
        id: row.id,
        tenant_id: '',
        name: row.name,
        slug: row.slug,
        description: '',
        model_type: row.type,
        suite: row.suite,
        risk_tier: row.risk_tier,
        status: row.status,
        tags: [],
        metadata: {},
        created_by: '',
        created_at: '',
        updated_at: '',
      },
      version: targetVersion,
    });
  };

  const handleStartShadow = async (row: AIDashboardModelRow) => {
    try {
      setBusyModelId(row.id);
      const versions = await enterpriseApi.ai.listVersions(row.id);
      const candidate = versions.find((version) => version.status === 'staging')
        ?? versions.find((version) => version.status === 'development');
      if (!candidate) {
        throw new Error('No staging or development version is available to start shadow mode.');
      }
      await enterpriseApi.ai.startShadow(row.id, { version_id: candidate.id });
      showSuccess('Shadow mode started.', `${row.slug} v${candidate.version_number} is now receiving asynchronous shadow traffic.`);
      await Promise.all([dashboardQuery.refetch(), refetch()]);
    } catch (error) {
      showApiError(error);
    } finally {
      setBusyModelId(null);
    }
  };

  const columns = useMemo(
    () =>
      createModelColumns({
        busyModelId,
        onPromote: handlePromote,
        onRollback: (row) =>
          setRollbackTarget({
            id: row.id,
            tenant_id: '',
            name: row.name,
            slug: row.slug,
            description: '',
            model_type: row.type,
            suite: row.suite,
            risk_tier: row.risk_tier,
            status: row.status,
            tags: [],
            metadata: {},
            created_by: '',
            created_at: '',
            updated_at: '',
          }),
        onStartShadow: handleStartShadow,
      }),
    [busyModelId],
  );

  const kpis = dashboardQuery.data?.kpis;

  return (
    <PermissionRedirect permission="users:read">
      <div className="space-y-6">
        <PageHeader
          title="AI Governance"
          description="Registered models, lifecycle controls, shadow comparisons, drift signals, and prediction feedback across every suite."
          actions={
            <div className="flex items-center gap-2">
              <Button onClick={() => setModelFormOpen(true)}>
                <Plus className="mr-1.5 h-4 w-4" />
                Register Model
              </Button>
              <Button variant="outline" onClick={() => void Promise.all([dashboardQuery.refetch(), refetch()])}>
                Refresh
              </Button>
            </div>
          }
        />

        <section className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
          <ModelCard label="Total Models" value={kpis?.total_models ?? 0} helper="All registered transparent models." />
          <ModelCard label="In Production" value={kpis?.in_production ?? 0} helper="Active production versions." />
          <ModelCard label="Shadow Testing" value={kpis?.shadow_testing ?? 0} helper="Versions receiving async shadow traffic." />
          <ModelCard label="Predictions 24h" value={(kpis?.predictions_24h ?? 0).toLocaleString()} helper="Governed inferences captured in the last day." />
          <ModelCard label="Drift Alerts" value={kpis?.drift_alerts ?? 0} helper="Open PSI, latency, volume, or accuracy alerts." />
        </section>

        <div className="rounded-3xl border border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(15,118,110,0.14),_transparent_38%),radial-gradient(circle_at_bottom_right,_rgba(180,83,9,0.12),_transparent_34%)] p-6">
          <div className="mb-4 flex items-center gap-3">
            <div className="rounded-2xl bg-primary/10 p-3 text-primary">
              <Bot className="h-6 w-6" />
            </div>
            <div>
              <h2 className="text-xl font-semibold">Model Registry</h2>
              <p className="text-sm text-muted-foreground">
                Every production-facing model on Clario 360 must remain explainable, versioned, and auditable.
              </p>
            </div>
          </div>
          <DataTable
            {...tableProps}
            columns={columns}
            emptyState={{
              icon: Bot,
              title: 'No governed models found',
              description: 'Seeded and tenant-registered models will appear here once the AI governance registry is provisioned.',
            }}
          />
        </div>
      </div>

      <PromoteDialog
        open={Boolean(promoteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setPromoteTarget(null);
          }
        }}
        model={promoteTarget?.model ?? null}
        version={promoteTarget?.version ?? null}
        onSaved={() => {
          void Promise.all([dashboardQuery.refetch(), refetch()]);
        }}
      />

      <ModelFormDialog
        open={modelFormOpen}
        onOpenChange={setModelFormOpen}
        onSaved={() => {
          void Promise.all([dashboardQuery.refetch(), refetch()]);
        }}
      />

      <RollbackDialog
        open={Boolean(rollbackTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setRollbackTarget(null);
          }
        }}
        model={rollbackTarget}
        onSaved={() => {
          void Promise.all([dashboardQuery.refetch(), refetch()]);
        }}
      />
    </PermissionRedirect>
  );
}
