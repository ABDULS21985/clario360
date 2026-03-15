'use client';

import { useRouter } from 'next/navigation';
import { Database, HardDrive, Lock, ShieldAlert, Fingerprint } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { dataAssetColumns } from '../_components/data-asset-columns';
import type { DataAsset, DSPMDashboard } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';

export default function DataAssetsPage() {
  const router = useRouter();

  const {
    data: dashEnvelope,
    isLoading: dashLoading,
    error: dashError,
    mutate: refetchDash,
  } = useRealtimeData<{ data: DSPMDashboard }>(API_ENDPOINTS.CYBER_DSPM_DASHBOARD, {
    pollInterval: 120000,
  });

  const { tableProps, refetch } = useDataTable<DataAsset>({
    queryKey: 'cyber-dspm-assets',
    fetchFn: (params: FetchParams) => {
      const { filters, ...rest } = params;
      return apiGet<PaginatedResponse<DataAsset>>(API_ENDPOINTS.CYBER_DSPM_DATA_ASSETS, { ...rest, ...filters } as Record<string, unknown>);
    },
    defaultSort: { column: 'risk_score', direction: 'desc' },
  });

  const dashboard = dashEnvelope?.data;

  const totalAssets = dashboard?.total_data_assets ?? 0;
  const encryptedCount = totalAssets - (dashboard?.unencrypted_count ?? 0);
  const piiCount = dashboard?.pii_assets_count ?? 0;
  const highRiskCount = dashboard?.high_risk_assets_count ?? 0;

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
        label: t.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase()),
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

  const kpis = [
    { label: 'Total Assets', value: totalAssets, icon: HardDrive, color: 'text-blue-600' },
    { label: 'Encrypted', value: encryptedCount, icon: Lock, color: 'text-green-600' },
    { label: 'PII Assets', value: piiCount, icon: Fingerprint, color: 'text-amber-600' },
    { label: 'High Risk', value: highRiskCount, icon: ShieldAlert, color: 'text-red-600' },
  ];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Assets"
          description="Discover, classify, and monitor the security posture of all data assets across your environment"
        />

        {dashLoading ? (
          <LoadingSkeleton variant="card" count={4} />
        ) : dashError ? (
          <ErrorState message="Failed to load asset statistics" onRetry={() => void refetchDash()} />
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {kpis.map((kpi) => {
              const Icon = kpi.icon;
              return (
                <Card key={kpi.label}>
                  <CardContent className="flex items-center gap-4 p-5">
                    <div className={`rounded-lg bg-muted p-2.5 ${kpi.color}`}>
                      <Icon className="h-5 w-5" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">{kpi.label}</p>
                      <p className="text-2xl font-bold tabular-nums">{kpi.value.toLocaleString()}</p>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}

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
                searchPlaceholder="Search data assets..."
                onRowClick={(row) => router.push(`/cyber/dspm/assets/${row.id}`)}
                emptyState={{
                  icon: Database,
                  title: 'No data assets found',
                  description: 'Trigger a DSPM scan to discover and classify your data assets.',
                }}
              />
            )}
          </div>
        </div>
      </div>
    </PermissionRedirect>
  );
}
