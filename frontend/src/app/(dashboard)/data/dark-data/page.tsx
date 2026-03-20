'use client';

import { useState } from 'react';
import { FileQuestion } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { buildDarkDataColumns } from '@/app/(dashboard)/data/dark-data/_components/darkdata-columns';
import { DarkDataDetailPanel } from '@/app/(dashboard)/data/dark-data/_components/darkdata-detail-panel';
import { DarkDataGovernDialog } from '@/app/(dashboard)/data/dark-data/_components/darkdata-govern-dialog';
import { DarkDataKpiCards } from '@/app/(dashboard)/data/dark-data/_components/darkdata-kpi-cards';
import { DarkDataScanDialog } from '@/app/(dashboard)/data/dark-data/_components/darkdata-scan-dialog';
import { DarkDataStatusDialog } from '@/app/(dashboard)/data/dark-data/_components/darkdata-status-dialog';
import { dataSuiteApi, type DarkDataAsset } from '@/lib/data-suite';
import type { DarkDataGovernValues } from '@/lib/data-suite/forms';
import { showApiError, showSuccess } from '@/lib/toast';

const DARK_DATA_FILTERS = [
  {
    key: 'reason',
    label: 'Reason',
    type: 'multi-select' as const,
    options: [
      { label: 'Unmodeled', value: 'unmodeled' },
      { label: 'Orphaned', value: 'orphaned_file' },
      { label: 'Stale', value: 'stale' },
      { label: 'Ungoverned', value: 'ungoverned' },
      { label: 'Unclassified', value: 'unclassified' },
    ],
  },
  {
    key: 'governance_status',
    label: 'Governance',
    type: 'multi-select' as const,
    options: [
      { label: 'Unmanaged', value: 'unmanaged' },
      { label: 'Under Review', value: 'under_review' },
      { label: 'Governed', value: 'governed' },
      { label: 'Archived', value: 'archived' },
      { label: 'Scheduled Deletion', value: 'scheduled_deletion' },
    ],
  },
];

export default function DataDarkDataPage() {
  const [selected, setSelected] = useState<DarkDataAsset | null>(null);
  const [governing, setGoverning] = useState<DarkDataAsset | null>(null);
  const [submittingGovern, setSubmittingGovern] = useState(false);
  const [statusAsset, setStatusAsset] = useState<DarkDataAsset | null>(null);
  const [targetStatus, setTargetStatus] = useState<'archived' | 'scheduled_deletion' | null>(null);
  const [submittingStatus, setSubmittingStatus] = useState(false);
  const [scanOpen, setScanOpen] = useState(false);

  const statsQuery = useQuery({
    queryKey: ['data-dark-data-stats'],
    queryFn: () => dataSuiteApi.getDarkDataStats(),
  });

  const { tableProps, searchValue, setSearch, refetch } = useDataTable<DarkDataAsset>({
    queryKey: 'data-dark-data',
    fetchFn: (params) => dataSuiteApi.listDarkDataAssets(params),
    defaultPageSize: 25,
    defaultSort: { column: 'risk_score', direction: 'desc' },
  });

  const governAsset = async (values: DarkDataGovernValues) => {
    if (!governing) {
      return;
    }
    try {
      setSubmittingGovern(true);
      await dataSuiteApi.governDarkData(governing.id, values);
      showSuccess('Asset brought under governance.');
      setGoverning(null);
      void refetch();
      void statsQuery.refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setSubmittingGovern(false);
    }
  };

  const updateStatus = async (notes: string) => {
    if (!statusAsset || !targetStatus) {
      return;
    }
    try {
      setSubmittingStatus(true);
      await dataSuiteApi.updateDarkDataStatus(statusAsset.id, {
        governance_status: targetStatus,
        governance_notes: notes,
      });
      showSuccess(targetStatus === 'archived' ? 'Asset archived.' : 'Deletion scheduled.');
      setStatusAsset(null);
      setTargetStatus(null);
      void refetch();
      void statsQuery.refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setSubmittingStatus(false);
    }
  };

  if (statsQuery.isLoading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Dark Data" description="Loading dark data inventory and governance posture." />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (statsQuery.error || !statsQuery.data) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message="Failed to load dark data statistics." onRetry={() => void statsQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Dark Data"
          description="Discovery and governance workflow for unmodeled, stale, or unmanaged data assets."
          actions={
            <Button type="button" onClick={() => setScanOpen(true)}>
              Scan now
            </Button>
          }
        />

        <DarkDataKpiCards stats={statsQuery.data} />

        <DataTable
          {...tableProps}
          columns={buildDarkDataColumns({
            onReview: setSelected,
            onGovern: setGoverning,
            onArchive: (asset) => {
              setStatusAsset(asset);
              setTargetStatus('archived');
            },
            onScheduleDeletion: (asset) => {
              setStatusAsset(asset);
              setTargetStatus('scheduled_deletion');
            },
          })}
          filters={DARK_DATA_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search dark data assets..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: FileQuestion,
            title: 'No dark data assets found',
            description: 'No dark data assets matched the current filters.',
          }}
        />

        <DarkDataDetailPanel
          open={Boolean(selected)}
          onOpenChange={(open) => {
            if (!open) {
              setSelected(null);
            }
          }}
          asset={selected}
        />

        <DarkDataGovernDialog
          open={Boolean(governing)}
          onOpenChange={(open) => {
            if (!open) {
              setGoverning(null);
            }
          }}
          asset={governing}
          submitting={submittingGovern}
          onSubmit={(values) => void governAsset(values)}
        />

        <DarkDataScanDialog
          open={scanOpen}
          onOpenChange={setScanOpen}
          onComplete={() => {
            void refetch();
            void statsQuery.refetch();
          }}
        />

        <DarkDataStatusDialog
          open={Boolean(statusAsset && targetStatus)}
          onOpenChange={(open) => {
            if (!open) {
              setStatusAsset(null);
              setTargetStatus(null);
            }
          }}
          asset={statusAsset}
          targetStatus={targetStatus}
          submitting={submittingStatus}
          onSubmit={(values) => void updateStatus(values.governance_notes)}
        />
      </div>
    </PermissionRedirect>
  );
}
