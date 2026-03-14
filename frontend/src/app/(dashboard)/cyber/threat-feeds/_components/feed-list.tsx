'use client';

import { useMemo } from 'react';
import { useQueries } from '@tanstack/react-query';
import { MoreHorizontal, Play } from 'lucide-react';
import type { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { DataTable } from '@/components/shared/data-table/data-table';
import { RelativeTime } from '@/components/shared/relative-time';
import { apiGet } from '@/lib/api';
import {
  getThreatFeedTypeLabel,
} from '@/lib/cyber-indicators';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDateTime, truncate } from '@/lib/utils';
import type { ThreatFeedConfig, ThreatFeedSyncHistory } from '@/types/cyber';
import type { DataTableControlledProps } from '@/types/table';

interface FeedListProps {
  tableProps: DataTableControlledProps<ThreatFeedConfig>;
  onSelect: (feed: ThreatFeedConfig) => void;
  onEdit: (feed: ThreatFeedConfig) => void;
  onSync: (feed: ThreatFeedConfig) => void;
}

export function FeedList({
  tableProps,
  onSelect,
  onEdit,
  onSync,
}: FeedListProps) {
  const historyQueries = useQueries({
    queries: tableProps.data.map((feed) => ({
      queryKey: ['cyber-threat-feed-last-history', feed.id],
      queryFn: () => apiGet<{ data: ThreatFeedSyncHistory[] }>(API_ENDPOINTS.CYBER_THREAT_FEED_HISTORY(feed.id)),
      staleTime: 120_000,
    })),
  });

  const lastImportedMap = useMemo(() => {
    const entries = new Map<string, number | null>();
    tableProps.data.forEach((feed, index) => {
      entries.set(feed.id, historyQueries[index]?.data?.data?.[0]?.indicators_imported ?? null);
    });
    return entries;
  }, [historyQueries, tableProps.data]);

  const columns = useMemo<ColumnDef<ThreatFeedConfig>[]>(() => [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Feed',
      cell: ({ row }) => (
        <button
          type="button"
          className="text-left font-medium hover:underline"
          onClick={(event) => {
            event.stopPropagation();
            onSelect(row.original);
          }}
        >
          {row.original.name}
        </button>
      ),
    },
    {
      id: 'type',
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }) => (
        <Badge variant="outline">{getThreatFeedTypeLabel(row.original.type)}</Badge>
      ),
    },
    {
      id: 'url',
      accessorKey: 'url',
      header: 'URL',
      cell: ({ row }) => (
        <span className="max-w-[320px] truncate text-sm text-muted-foreground">
          {row.original.url ? truncate(row.original.url, 48) : 'Manual feed'}
        </span>
      ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => (
        <Badge variant={row.original.status === 'error' ? 'destructive' : row.original.enabled ? 'default' : 'secondary'}>
          {row.original.status}
        </Badge>
      ),
    },
    {
      id: 'last_sync_at',
      accessorKey: 'last_sync_at',
      header: 'Last Sync',
      cell: ({ row }) => (
        row.original.last_sync_at ? <RelativeTime date={row.original.last_sync_at} /> : <span className="text-sm text-muted-foreground">Never</span>
      ),
    },
    {
      id: 'indicators_imported',
      header: 'Imported',
      cell: ({ row }) => (
        <span className="text-sm">
          {lastImportedMap.get(row.original.id) ?? '—'}
        </span>
      ),
    },
    {
      id: 'next_sync_at',
      accessorKey: 'next_sync_at',
      header: 'Next Sync',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.next_sync_at ? formatDateTime(row.original.next_sync_at) : 'Manual'}
        </span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0"
              onClick={(event) => event.stopPropagation()}
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => onSelect(row.original)}>
              View details
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onEdit(row.original)}>
              Edit feed
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onSync(row.original)}>
              <Play className="mr-2 h-4 w-4" />
              Sync now
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ], [lastImportedMap, onEdit, onSelect, onSync]);

  return (
    <DataTable
      {...tableProps}
      columns={columns}
      searchPlaceholder="Search feeds…"
      getRowId={(row) => row.id}
      onRowClick={(row) => onSelect(row)}
      enableColumnToggle={false}
    />
  );
}
