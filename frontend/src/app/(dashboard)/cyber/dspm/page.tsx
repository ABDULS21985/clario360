'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Database, ScanSearch } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { DSPMKpiCards } from './_components/dspm-kpi-cards';
import { ClassificationChart } from './_components/classification-chart';
import { dataAssetColumns } from './_components/data-asset-columns';
import { ScanTriggerDialog } from './_components/scan-trigger-dialog';
import type { DataAsset, DSPMDashboard, ShadowDetectionResult } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';

export default function CyberDspmPage() {
  const [scanOpen, setScanOpen] = useState(false);

  const {
    data: dashEnvelope,
    isLoading: dashLoading,
    error: dashError,
    mutate: refetchDash,
  } = useRealtimeData<{ data: DSPMDashboard }>(API_ENDPOINTS.CYBER_DSPM_DASHBOARD, {
    pollInterval: 120000,
  });

  const { tableProps, refetch } = useDataTable<DataAsset>({
    queryKey: 'cyber-dspm',
    fetchFn: (params: FetchParams) => {
      const { filters, ...rest } = params;
      return apiGet<PaginatedResponse<DataAsset>>(API_ENDPOINTS.CYBER_DSPM_DATA_ASSETS, { ...rest, ...filters } as Record<string, unknown>);
    },
    defaultSort: { column: 'risk_score', direction: 'desc' },
  });
  const shadowQuery = useQuery({
    queryKey: ['cyber-dspm-shadow-copies'],
    queryFn: () => apiGet<{ data: ShadowDetectionResult }>(API_ENDPOINTS.CYBER_DSPM_SHADOW_COPIES),
    staleTime: 120000,
  });

  const dashboard = dashEnvelope?.data;
  const shadowResult = shadowQuery.data?.data;
  const shadowMatches = shadowResult?.matches ?? [];
  const highRiskShare = dashboard && dashboard.total_data_assets > 0
    ? Math.round((dashboard.high_risk_assets_count / dashboard.total_data_assets) * 100)
    : 0;
  const piiCoverage = dashboard && dashboard.total_data_assets > 0
    ? Math.round((dashboard.pii_assets_count / dashboard.total_data_assets) * 100)
    : 0;

  const filters = [
    {
      key: 'classification',
      label: 'Classification',
      type: 'multi-select' as const,
      options: ['public', 'internal', 'confidential', 'restricted', 'top_secret'].map((c) => ({
        label: c.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase()),
        value: c,
      })),
    },
    {
      key: 'asset_type',
      label: 'Asset Type',
      type: 'multi-select' as const,
      options: ['database', 'cloud_storage', 'file_server', 'api'].map((t) => ({
        label: t.replace(/_/g, ' '),
        value: t,
      })),
    },
    {
      key: 'encrypted',
      label: 'Encrypted',
      type: 'multi-select' as const,
      options: [
        { label: 'Encrypted', value: 'true' },
        { label: 'Unencrypted', value: 'false' },
      ],
    },
  ];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Security Posture Management"
          description="Monitor classification, encryption, access controls, and compliance posture of your data assets"
          actions={
            <Button size="sm" onClick={() => setScanOpen(true)}>
              <ScanSearch className="mr-1.5 h-3.5 w-3.5" />
              Trigger Scan
            </Button>
          }
        />

        {dashLoading ? (
          <LoadingSkeleton variant="card" />
        ) : dashError || !dashboard ? (
          <ErrorState message="Failed to load DSPM dashboard" onRetry={() => void refetchDash()} />
        ) : (
          <>
            <DSPMKpiCards dashboard={dashboard} />

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              {/* Classification chart */}
              <div className="rounded-xl border bg-card p-5">
                <ClassificationChart data={dashboard.classification_breakdown} />
              </div>

              {/* Posture bars */}
              <div className="rounded-xl border bg-card p-5">
                <h3 className="mb-4 text-sm font-semibold">Posture Overview</h3>
                <div className="space-y-3">
                  {[
                    {
                      label: 'PII Coverage',
                      value: piiCoverage,
                      good: true,
                    },
                    {
                      label: 'Encryption Coverage',
                      value: dashboard.total_data_assets > 0
                        ? Math.round(((dashboard.total_data_assets - dashboard.unencrypted_count) / dashboard.total_data_assets) * 100)
                        : 100,
                      good: true,
                    },
                    {
                      label: 'Access Control',
                      value: dashboard.total_data_assets > 0
                        ? Math.round(((dashboard.total_data_assets - dashboard.no_access_control_count) / dashboard.total_data_assets) * 100)
                        : 100,
                      good: true,
                    },
                    {
                      label: 'High Risk Assets',
                      value: highRiskShare,
                      good: false,
                    },
                  ].map(({ label, value, good }) => {
                    const isGood = good ? value >= 80 : value <= 20;
                    const barColor = isGood ? 'bg-green-500' : value >= 50 ? 'bg-amber-500' : 'bg-red-500';
                    return (
                      <div key={label}>
                        <div className="mb-1 flex justify-between text-xs">
                          <span className="text-muted-foreground">{label}</span>
                          <span className="font-medium">{value}%</span>
                        </div>
                        <div className="h-2 overflow-hidden rounded-full bg-muted">
                          <div className={`h-full rounded-full transition-all ${barColor}`} style={{ width: `${value}%` }} />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Scan activity */}
              <div className="rounded-xl border bg-card p-5">
                <h3 className="mb-4 text-sm font-semibold">Scan Activity</h3>
                <div className="flex items-center gap-4">
                  <div className="flex flex-col items-center">
                    <span className="text-3xl font-bold text-blue-600">{(dashboard.recent_scans ?? []).length}</span>
                    <span className="text-xs text-muted-foreground">Scans (30d)</span>
                  </div>
                </div>
                <div className="mt-4 rounded-lg border bg-muted/20 p-3 text-xs text-muted-foreground">
                  Continuous DSPM is now watching pipeline transit, at-rest drift, and shadow-copy activity in addition to manual full scans.
                </div>
                <Button className="mt-4 w-full" variant="outline" size="sm" onClick={() => setScanOpen(true)}>
                  <ScanSearch className="mr-1.5 h-3.5 w-3.5" />
                  Run New Scan
                </Button>
              </div>
            </div>
          </>
        )}

        <div className="rounded-xl border bg-card">
          <div className="border-b px-5 py-4">
            <h3 className="text-sm font-semibold">Shadow Copy Detection</h3>
            <p className="text-xs text-muted-foreground">Structural fingerprint matches without lineage-backed copy paths.</p>
          </div>
          <div className="p-5">
            {shadowQuery.isLoading ? (
              <LoadingSkeleton variant="table-row" count={3} />
            ) : shadowQuery.error ? (
              <ErrorState
                message="Failed to load shadow copy matches"
                onRetry={() => void shadowQuery.refetch()}
              />
            ) : shadowMatches.length === 0 ? (
              <div className="rounded-lg border bg-muted/20 p-4 text-sm text-muted-foreground">
                {shadowResult?.summary ?? 'No unauthorized shadow-copy candidates were detected in the latest structural scan.'}
              </div>
            ) : (
              <div className="space-y-3">
                {shadowResult?.summary ? (
                  <div className="rounded-lg border bg-muted/20 p-3 text-sm text-muted-foreground">
                    {shadowResult.summary}
                  </div>
                ) : null}
                {shadowMatches.slice(0, 6).map((match) => (
                  <div key={`${match.source_asset_id}-${match.target_asset_id}-${match.fingerprint}`} className="rounded-lg border p-4">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <p className="text-sm font-medium">
                          {match.source_asset_name}.{match.source_table} → {match.target_asset_name}.{match.target_table}
                        </p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          {match.match_type.replace(/_/g, ' ')} match · {Math.round(match.similarity * 100)}% similarity
                        </p>
                      </div>
                      <div className="text-right text-xs text-muted-foreground">
                        <div>Sources: {shadowResult?.sources_count ?? 0}</div>
                        <div>Tables: {shadowResult?.tables_count ?? 0}</div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Data Assets Table */}
        <div className="rounded-xl border bg-card">
          <div className="border-b px-5 py-4">
            <h3 className="text-sm font-semibold">Data Assets</h3>
            <p className="text-xs text-muted-foreground">All discovered data assets with their security posture</p>
          </div>
          <div className="p-5">
            {tableProps.isLoading ? (
              <LoadingSkeleton variant="table-row" count={6} />
            ) : tableProps.error ? (
              <ErrorState message="Failed to load data assets" onRetry={refetch} />
            ) : (
              <DataTable
                {...tableProps}
                columns={dataAssetColumns}
                filters={filters}
                onSortChange={() => undefined}
                searchPlaceholder="Search data assets…"
                emptyState={{
                  icon: Database,
                  title: 'No data assets found',
                  description: 'Trigger a DSPM scan to discover and classify your data assets.',
                  action: { label: 'Trigger Scan', onClick: () => setScanOpen(true) },
                }}
              />
            )}
          </div>
        </div>

        <ScanTriggerDialog
          open={scanOpen}
          onOpenChange={setScanOpen}
          onSuccess={() => { refetch(); void refetchDash(); }}
        />
      </div>
    </PermissionRedirect>
  );
}
