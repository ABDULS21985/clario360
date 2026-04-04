'use client';

import { useMemo, useState } from 'react';
import {
  LayoutGrid,
  List,
  Plus,
  Scan,
  Upload,
  ShieldAlert,
  Tag,
  Trash2,
  ShieldCheck,
  ShieldOff,
} from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { EmptyState } from '@/components/common/empty-state';
import { ExportMenu } from '@/components/cyber/export-menu';
import { useDataTable } from '@/hooks/use-data-table';
import { useAuth } from '@/hooks/use-auth';
import { apiGet, apiPut, apiDelete } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { BulkAction, FetchParams } from '@/types/table';
import type { CyberAsset } from '@/types/cyber';

import { AssetKpiCards } from './_components/asset-kpi-cards';
import { getAssetColumns } from './_components/asset-columns';
import { ASSET_FILTERS, flattenAssetFetchParams } from './_components/asset-filters';
import { AssetGridView } from './_components/asset-grid-view';
import { AssetTrendCharts } from './_components/asset-trend-charts';
import { CreateAssetDialog } from './_components/create-asset-dialog';
import { EditAssetDialog } from './_components/edit-asset-dialog';
import { DeleteAssetDialog } from './_components/delete-asset-dialog';
import { TagManagementDialog } from './_components/tag-management-dialog';
import { BulkTagDialog } from './_components/bulk-tag-dialog';
import { ScanDialog } from './_components/scan-dialog';
import { ScanScheduleDialog } from './_components/scan-schedule-dialog';
import { BulkImportDialog } from './_components/bulk-import-dialog';
import { AddRelationshipDialog } from './_components/add-relationship-dialog';

type ViewMode = 'table' | 'grid';

function fetchAssets(params: FetchParams): Promise<PaginatedResponse<CyberAsset>> {
  return apiGet<PaginatedResponse<CyberAsset>>(API_ENDPOINTS.CYBER_ASSETS, flattenAssetFetchParams(params));
}

export default function AssetsPage() {
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('cyber:write');

  const [view, setView] = useState<ViewMode>('table');
  const [createOpen, setCreateOpen] = useState(false);
  const [scanOpen, setScanOpen] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<CyberAsset | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CyberAsset | null>(null);
  const [tagTarget, setTagTarget] = useState<CyberAsset | null>(null);
  const [bulkTagIds, setBulkTagIds] = useState<string[]>([]);
  const [relationshipTarget, setRelationshipTarget] = useState<CyberAsset | null>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  const { tableProps, data, totalRows, activeFilters, refetch } = useDataTable<CyberAsset>({
    fetchFn: fetchAssets,
    queryKey: 'cyber-assets',
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['asset.created', 'asset.updated', 'asset.deleted', 'vulnerability.created'],
  });

  const columns = getAssetColumns({
    onEdit: canWrite ? setEditTarget : undefined,
    onDelete: canWrite ? setDeleteTarget : undefined,
    onTag: canWrite ? setTagTarget : undefined,
    onRelationship: canWrite ? setRelationshipTarget : undefined,
  });

  const handleBulkComplete = async () => {
    setSelectedIds([]);
    await refetch();
  };

  const bulkActions = useMemo<BulkAction[]>(() => {
    if (!canWrite) return [];
    return [
      {
        label: 'Bulk Tag',
        icon: Tag,
        onClick: async (ids) => {
          if (ids.length === 0) {
            toast.error('Select at least one asset');
            return;
          }
          setBulkTagIds(ids);
        },
      },
      {
        label: 'Set Active',
        icon: ShieldCheck,
        onClick: async (ids) => {
          if (ids.length === 0) {
            toast.error('Select at least one asset');
            return;
          }
          let updated = 0;
          for (const id of ids) {
            try {
              await apiPut(`${API_ENDPOINTS.CYBER_ASSETS}/${id}`, { status: 'active' });
              updated++;
            } catch {
              // continue on individual failures
            }
          }
          toast.success(`${updated} asset(s) set to active`);
          await handleBulkComplete();
        },
      },
      {
        label: 'Decommission',
        icon: ShieldOff,
        onClick: async (ids) => {
          if (ids.length === 0) {
            toast.error('Select at least one asset');
            return;
          }
          let updated = 0;
          for (const id of ids) {
            try {
              await apiPut(`${API_ENDPOINTS.CYBER_ASSETS}/${id}`, { status: 'decommissioned' });
              updated++;
            } catch {
              // continue on individual failures
            }
          }
          toast.success(`${updated} asset(s) decommissioned`);
          await handleBulkComplete();
        },
      },
      {
        label: 'Delete Selected',
        icon: Trash2,
        variant: 'destructive',
        confirmMessage: 'Are you sure you want to delete the selected assets? This action cannot be undone.',
        onClick: async (ids) => {
          if (ids.length === 0) {
            toast.error('Select at least one asset');
            return;
          }
          await apiDelete(API_ENDPOINTS.CYBER_ASSETS_BULK);
          toast.success(`${ids.length} asset(s) deleted`);
          await handleBulkComplete();
        },
      },
    ];
  }, [canWrite]);

  const emptyState = {
    icon: ShieldAlert,
    title: 'No assets found',
    description: 'Get started by creating an asset or running an automated discovery scan.',
    action: { label: 'Create Asset', onClick: () => setCreateOpen(true) },
  };

  // Build current filter params for export
  const exportFilters = useMemo(() => {
    const filters: Record<string, string | string[]> = {};
    for (const [key, value] of Object.entries(activeFilters ?? {})) {
      if (value) filters[key] = value;
    }
    return filters;
  }, [activeFilters]);

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Asset Inventory"
          description="Manage and monitor all cyber assets across your environment"
          actions={
            <div className="flex items-center gap-2">
              {/* View toggle */}
              <div className="flex rounded-md border">
                <Button
                  variant={view === 'table' ? 'secondary' : 'ghost'}
                  size="sm"
                  className="rounded-r-none border-r px-2"
                  onClick={() => setView('table')}
                >
                  <List className="h-4 w-4" />
                </Button>
                <Button
                  variant={view === 'grid' ? 'secondary' : 'ghost'}
                  size="sm"
                  className="rounded-l-none px-2"
                  onClick={() => setView('grid')}
                >
                  <LayoutGrid className="h-4 w-4" />
                </Button>
              </div>

              {/* Export */}
              <ExportMenu
                entityType="assets"
                baseUrl={API_ENDPOINTS.CYBER_ASSETS}
                currentFilters={exportFilters}
                totalCount={totalRows}
                enabledFormats={['csv', 'json']}
                selectedCount={selectedIds.length}
              />

              {/* Actions */}
              <Button variant="outline" size="sm" onClick={() => setScanOpen(true)}>
                <Scan className="mr-1.5 h-3.5 w-3.5" />
                Scan
              </Button>
              <Button variant="outline" size="sm" onClick={() => setScheduleOpen(true)}>
                Schedule
              </Button>
              <Button variant="outline" size="sm" onClick={() => setBulkOpen(true)}>
                <Upload className="mr-1.5 h-3.5 w-3.5" />
                Import
              </Button>
              {canWrite && (
                <Button size="sm" onClick={() => setCreateOpen(true)}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  Add Asset
                </Button>
              )}
            </div>
          }
        />

        <AssetKpiCards />

        {/* Asset Trend Charts */}
        <AssetTrendCharts />

        {view === 'table' ? (
          <DataTable
            columns={columns}
            filters={ASSET_FILTERS}
            searchPlaceholder="Search by name, hostname, IP, owner, department…"
            emptyState={emptyState}
            getRowId={(row) => row.id}
            enableColumnToggle
            enableSelection={canWrite}
            onSelectionChange={setSelectedIds}
            bulkActions={bulkActions}
            {...tableProps}
          />
        ) : (
          <div className="space-y-4">
            {tableProps.isLoading ? (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {Array.from({ length: 8 }).map((_, i) => (
                  <div key={i} className="h-40 animate-pulse rounded-lg border bg-muted" />
                ))}
              </div>
            ) : data.length === 0 ? (
              <EmptyState
                icon={ShieldAlert}
                title="No assets found"
                description="Get started by creating an asset or running a scan."
                action={{ label: 'Create Asset', onClick: () => setCreateOpen(true) }}
              />
            ) : (
              <AssetGridView
                assets={data}
                onEdit={canWrite ? setEditTarget : undefined}
                onDelete={canWrite ? setDeleteTarget : undefined}
                onTag={canWrite ? setTagTarget : undefined}
              />
            )}
          </div>
        )}
      </div>

      {/* Dialogs */}
      <CreateAssetDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={() => refetch()}
      />
      <ScanDialog open={scanOpen} onOpenChange={setScanOpen} />
      <ScanScheduleDialog open={scheduleOpen} onOpenChange={setScheduleOpen} />
      <BulkImportDialog
        open={bulkOpen}
        onOpenChange={setBulkOpen}
        onSuccess={() => refetch()}
      />
      {editTarget && (
        <EditAssetDialog
          open={!!editTarget}
          onOpenChange={(o) => { if (!o) setEditTarget(null); }}
          asset={editTarget}
          onSuccess={() => { setEditTarget(null); refetch(); }}
        />
      )}
      {deleteTarget && (
        <DeleteAssetDialog
          open={!!deleteTarget}
          onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}
          asset={deleteTarget}
          onSuccess={() => { setDeleteTarget(null); refetch(); }}
        />
      )}
      {tagTarget && (
        <TagManagementDialog
          open={!!tagTarget}
          onOpenChange={(o) => { if (!o) setTagTarget(null); }}
          asset={tagTarget}
          onSuccess={() => { setTagTarget(null); refetch(); }}
        />
      )}
      {bulkTagIds.length > 0 && (
        <BulkTagDialog
          open={bulkTagIds.length > 0}
          onOpenChange={(o) => { if (!o) setBulkTagIds([]); }}
          assetIds={bulkTagIds}
          onSuccess={() => { setBulkTagIds([]); handleBulkComplete(); }}
        />
      )}
      {relationshipTarget && (
        <AddRelationshipDialog
          open={!!relationshipTarget}
          onOpenChange={(o) => { if (!o) setRelationshipTarget(null); }}
          asset={relationshipTarget}
          onSuccess={() => { setRelationshipTarget(null); refetch(); }}
        />
      )}
    </PermissionRedirect>
  );
}
