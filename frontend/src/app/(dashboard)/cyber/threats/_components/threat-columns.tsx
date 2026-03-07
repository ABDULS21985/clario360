'use client';

import { ColumnDef, Row } from '@tanstack/react-table';
import { MoreHorizontal } from 'lucide-react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Badge } from '@/components/ui/badge';
import { timeAgo } from '@/lib/utils';
import type { Threat } from '@/types/cyber';

interface ThreatColumnOptions {
  onViewDetail?: (threat: Threat) => void;
}

export function getThreatColumns(options: ThreatColumnOptions = {}): ColumnDef<Threat>[] {
  return [
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }: { row: Row<Threat> }) => (
        <SeverityIndicator severity={row.original.severity} showLabel />
      ),
      enableSorting: true,
    },
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Threat',
      cell: ({ row }: { row: Row<Threat> }) => {
        const threat = row.original;
        return (
          <div>
            <button
              className="font-medium hover:underline text-left"
              onClick={() => options.onViewDetail?.(threat)}
            >
              {threat.name}
            </button>
            <p className="text-xs text-muted-foreground capitalize">{threat.type}</p>
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }: { row: Row<Threat> }) => (
        <StatusBadge status={row.original.status} />
      ),
      enableSorting: true,
    },
    {
      id: 'indicator_count',
      accessorKey: 'indicator_count',
      header: 'Indicators',
      cell: ({ row }: { row: Row<Threat> }) => (
        <span className="tabular-nums text-sm">{row.original.indicator_count}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'affected_asset_count',
      accessorKey: 'affected_asset_count',
      header: 'Affected Assets',
      cell: ({ row }: { row: Row<Threat> }) => {
        const count = row.original.affected_asset_count;
        return (
          <span className={`tabular-nums text-sm ${count > 0 ? 'font-medium text-orange-600' : 'text-muted-foreground'}`}>
            {count}
          </span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'tags',
      header: 'Tags',
      cell: ({ row }: { row: Row<Threat> }) => (
        <div className="flex flex-wrap gap-1">
          {(row.original.tags ?? []).slice(0, 2).map((tag) => (
            <Badge key={tag} variant="secondary" className="text-xs">{tag}</Badge>
          ))}
        </div>
      ),
    },
    {
      id: 'last_seen',
      accessorKey: 'last_seen',
      header: 'Last Seen',
      cell: ({ row }: { row: Row<Threat> }) => (
        <span className="text-sm text-muted-foreground">{timeAgo(row.original.last_seen)}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<Threat> }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => options.onViewDetail?.(row.original)}>
              View Details
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
      enableSorting: false,
    },
  ];
}
