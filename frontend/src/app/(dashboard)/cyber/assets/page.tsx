'use client';

import { useState } from 'react';
import { LayoutGrid, List, Plus, Scan, Upload, ShieldAlert } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { EmptyState } from '@/components/common/empty-state';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';
import type { CyberAsset } from '@/types/cyber';

import { AssetKpiCards } from './_components/asset-kpi-cards';
import { getAssetColumns } from './_components/asset-columns';
import { ASSET_FILTERS, flattenAssetFetchParams } from './_components/asset-filters';
import { AssetGridView } from './_components/asset-grid-view';
import { CreateAssetDialog } from './_components/create-asset-dialog';
import { EditAssetDialog } from './_components/edit-asset-dialog';
import { DeleteAssetDialog } from './_components/delete-asset-dialog';
import { TagManagementDialog } from './_components/tag-management-dialog';
import { ScanDialog } from './_components/scan-dialog';
import { BulkImportDialog } from './_components/bulk-import-dialog';

type ViewMode = 'table' | 'grid';

function fetchAssets(params: FetchParams): Promise<PaginatedResponse<CyberAsset>> {
  return apiGet<PaginatedResponse<CyberAsset>>(API_ENDPOINTS.CYBER_ASSETS, flattenAssetFetchParams(params));
}

export default function AssetsPage() {
  const [view, setView] = useState<ViewMode>('table');
  const [createOpen, setCreateOpen] = useState(false);
  const [scanOpen, setScanOpen] = useState(false);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<CyberAsset | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CyberAsset | null>(null);
  const [tagTarget, setTagTarget] = useState<CyberAsset | null>(null);

  const { tableProps, data, refetch } = useDataTable<CyberAsset>({
    fetchFn: fetchAssets,
    queryKey: 'cyber-assets',
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['asset.created', 'asset.updated', 'asset.deleted', 'vulnerability.created'],
  });

  const columns = getAssetColumns({
    onEdit: setEditTarget,
    onDelete: setDeleteTarget,
    onTag: setTagTarget,
  });

  const emptyState = {
    icon: ShieldAlert,
    title: 'No assets found',
    description: 'Get started by creating an asset or running an automated discovery scan.',
    action: { label: 'Create Asset', onClick: () => setCreateOpen(true) },
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Asset Inventory"
          description="Manage and monitor all cyber assets across your environment"
          actions={
            <div className="flex items-center gap-2">
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
              <Button variant="outline" size="sm" onClick={() => setScanOpen(true)}>
                <Scan className="mr-1.5 h-3.5 w-3.5" />
                Scan
              </Button>
              <Button variant="outline" size="sm" onClick={() => setBulkOpen(true)}>
                <Upload className="mr-1.5 h-3.5 w-3.5" />
                Import
              </Button>
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                <Plus className="mr-1.5 h-3.5 w-3.5" />
                Add Asset
              </Button>
            </div>
          }
        />

        <AssetKpiCards />

        {view === 'table' ? (
          <DataTable
            columns={columns}
            filters={ASSET_FILTERS}
            searchPlaceholder="Search by name, hostname, IP…"
            emptyState={emptyState}
            getRowId={(row) => row.id}
            enableColumnToggle
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
                onEdit={setEditTarget}
                onDelete={setDeleteTarget}
                onTag={setTagTarget}
              />
            )}
          </div>
        )}
      </div>

      <CreateAssetDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={() => refetch()}
      />
      <ScanDialog open={scanOpen} onOpenChange={setScanOpen} />
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
    </PermissionRedirect>
  );
}
