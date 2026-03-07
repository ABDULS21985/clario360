'use client';

import { useRouter } from 'next/navigation';
import { ArrowLeft, ScanLine } from 'lucide-react';
import type { ColumnDef, Row } from '@tanstack/react-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDateTime, timeAgo } from '@/lib/utils';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';
import type { AssetScan } from '@/types/cyber';

const SCAN_TYPE_COLORS: Record<string, string> = {
  network: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  cloud: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  agent: 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300',
  manual: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
};

function ScanStatusBadge({ status }: { status: AssetScan['status'] }) {
  if (status === 'pending') {
    return (
      <Badge variant="secondary" className="gap-1.5 text-xs capitalize">
        {status}
      </Badge>
    );
  }
  if (status === 'running') {
    return (
      <Badge
        variant="secondary"
        className="gap-1.5 bg-blue-100 text-blue-800 hover:bg-blue-100 dark:bg-blue-900/30 dark:text-blue-300 text-xs capitalize"
      >
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-blue-500" />
        {status}
      </Badge>
    );
  }
  if (status === 'completed') {
    return (
      <Badge
        variant="secondary"
        className="bg-green-100 text-green-800 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-300 text-xs capitalize"
      >
        {status}
      </Badge>
    );
  }
  if (status === 'failed') {
    return (
      <Badge variant="destructive" className="text-xs capitalize">
        {status}
      </Badge>
    );
  }
  // cancelled
  return (
    <Badge variant="outline" className="text-xs capitalize">
      {status}
    </Badge>
  );
}

const SCAN_COLUMNS: ColumnDef<AssetScan>[] = [
  {
    id: 'scan_type',
    accessorKey: 'scan_type',
    header: 'Type',
    cell: ({ row }: { row: Row<AssetScan> }) => (
      <span
        className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${SCAN_TYPE_COLORS[row.original.scan_type] ?? 'bg-gray-100 text-gray-800'}`}
      >
        {row.original.scan_type}
      </span>
    ),
    enableSorting: true,
  },
  {
    id: 'status',
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }: { row: Row<AssetScan> }) => (
      <ScanStatusBadge status={row.original.status} />
    ),
    enableSorting: true,
  },
  {
    id: 'target',
    accessorKey: 'target',
    header: 'Target',
    cell: ({ row }: { row: Row<AssetScan> }) => (
      <span className="font-mono text-xs text-muted-foreground">
        {row.original.target ?? '—'}
      </span>
    ),
  },
  {
    id: 'assets_found',
    accessorKey: 'assets_found',
    header: 'Found',
    cell: ({ row }: { row: Row<AssetScan> }) => (
      <span className="tabular-nums text-sm font-medium text-green-600 dark:text-green-400">
        {row.original.assets_found.toLocaleString()}
      </span>
    ),
    enableSorting: true,
  },
  {
    id: 'assets_updated',
    accessorKey: 'assets_updated',
    header: 'Updated',
    cell: ({ row }: { row: Row<AssetScan> }) => (
      <span className="tabular-nums text-sm font-medium text-blue-600 dark:text-blue-400">
        {row.original.assets_updated.toLocaleString()}
      </span>
    ),
    enableSorting: true,
  },
  {
    id: 'started_at',
    accessorKey: 'started_at',
    header: 'Started',
    cell: ({ row }: { row: Row<AssetScan> }) => (
      <span className="text-sm text-muted-foreground">
        {row.original.started_at ? timeAgo(row.original.started_at) : '—'}
      </span>
    ),
    enableSorting: true,
  },
  {
    id: 'completed_at',
    accessorKey: 'completed_at',
    header: 'Completed',
    cell: ({ row }: { row: Row<AssetScan> }) => (
      <span className="text-sm text-muted-foreground">
        {row.original.completed_at ? formatDateTime(row.original.completed_at) : '—'}
      </span>
    ),
    enableSorting: true,
  },
  {
    id: 'error',
    header: 'Error',
    cell: ({ row }: { row: Row<AssetScan> }) => {
      if (!row.original.error) return null;
      return (
        <span className="max-w-xs truncate text-xs text-destructive" title={row.original.error}>
          {row.original.error}
        </span>
      );
    },
  },
];

function fetchScans(params: FetchParams): Promise<PaginatedResponse<AssetScan>> {
  return apiGet<PaginatedResponse<AssetScan>>(
    API_ENDPOINTS.CYBER_ASSETS_SCANS,
    params as unknown as Record<string, unknown>,
  );
}

export default function AssetScansPage() {
  const router = useRouter();

  const { tableProps } = useDataTable<AssetScan>({
    fetchFn: fetchScans,
    queryKey: 'cyber-asset-scans',
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['asset.scan.started', 'asset.scan.completed', 'asset.scan.failed'],
  });

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Asset Scans"
          description="Network and cloud asset discovery scan history"
          actions={
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push('/cyber/assets')}
            >
              <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
              Back to Assets
            </Button>
          }
        />

        <DataTable
          columns={SCAN_COLUMNS}
          searchPlaceholder="Search scans…"
          emptyState={{
            icon: ScanLine,
            title: 'No scans found',
            description: 'Run an asset discovery scan to populate this list.',
          }}
          getRowId={(row) => row.id}
          onRowClick={(row) => router.push(`/cyber/assets/scans/${row.id}`)}
          {...tableProps}
          onSortChange={() => undefined}
        />
      </div>
    </PermissionRedirect>
  );
}
