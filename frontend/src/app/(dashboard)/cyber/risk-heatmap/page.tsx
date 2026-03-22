'use client';

import { LayoutGrid } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { ExportMenu } from '@/components/cyber/export-menu';
import type { RiskHeatmapData } from '@/types/cyber';

import { HeatmapGrid } from './_components/heatmap-grid';
import { HeatmapLegend } from './_components/heatmap-legend';
import { HeatmapSummaryTable } from './_components/heatmap-summary-table';

export default function RiskHeatmapPage() {
  const {
    data: envelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<{ data: RiskHeatmapData }>(API_ENDPOINTS.CYBER_RISK_HEATMAP, {
    pollInterval: 300000,
  });

  const heatmap = envelope?.data;
  const isEmpty =
    heatmap && heatmap.cells.every((c) => c.count === 0) && heatmap.total_vulnerabilities === 0;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-5">
        <PageHeader
          title="Risk Heatmap"
          description="Vulnerability distribution across asset types and severity levels"
          actions={
            heatmap ? (
              <ExportMenu
                entityType="risk-heatmap"
                baseUrl={API_ENDPOINTS.CYBER_RISK_HEATMAP}
                currentFilters={{}}
                totalCount={heatmap.total_vulnerabilities}
                enabledFormats={['csv', 'json']}
                csvDataKey="cells"
              />
            ) : undefined
          }
        />

        {isLoading ? (
          <div className="space-y-4">
            <LoadingSkeleton variant="card" />
            <LoadingSkeleton variant="chart" />
          </div>
        ) : error || !heatmap ? (
          <ErrorState message="Failed to load risk heatmap" onRetry={() => void refetch()} />
        ) : isEmpty ? (
          <EmptyState
            icon={LayoutGrid}
            title="No vulnerability data available"
            description="Run a CTEM assessment or asset scan to populate the risk heatmap."
          />
        ) : (
          <>
            <HeatmapGrid data={heatmap} />
            <HeatmapLegend />
            <HeatmapSummaryTable data={heatmap} />
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
