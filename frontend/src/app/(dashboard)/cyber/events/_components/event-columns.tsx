'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import type { SecurityEvent } from '@/types/cyber';

export function getEventColumns(): ColumnDef<SecurityEvent>[] {
  return [
    {
      id: 'timestamp',
      accessorKey: 'timestamp',
      header: 'Timestamp',
      cell: ({ row }) => {
        const ts = row.original.timestamp;
        try {
          return (
            <span className="whitespace-nowrap text-xs tabular-nums">
              {new Date(ts).toLocaleString('en-US', {
                month: 'short',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false,
              })}
            </span>
          );
        } catch {
          return <span className="text-xs text-muted-foreground">—</span>;
        }
      },
      enableSorting: true,
    },
    {
      id: 'source',
      accessorKey: 'source',
      header: 'Source',
      cell: ({ row }) => (
        <span className="text-xs font-medium">{row.original.source}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'type',
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }) => (
        <Badge variant="outline" className="text-xs capitalize">
          {row.original.type.replace(/_/g, ' ')}
        </Badge>
      ),
      enableSorting: true,
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }) => (
        <SeverityIndicator severity={row.original.severity} showLabel size="sm" />
      ),
      enableSorting: true,
    },
    {
      id: 'source_ip',
      accessorKey: 'source_ip',
      header: 'Source IP',
      cell: ({ row }) =>
        row.original.source_ip ? (
          <code className="text-xs">{row.original.source_ip}</code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: true,
    },
    {
      id: 'dest_ip',
      accessorKey: 'dest_ip',
      header: 'Dest IP',
      cell: ({ row }) =>
        row.original.dest_ip ? (
          <code className="text-xs">
            {row.original.dest_ip}
            {row.original.dest_port ? `:${row.original.dest_port}` : ''}
          </code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: true,
    },
    {
      id: 'username',
      accessorKey: 'username',
      header: 'User',
      cell: ({ row }) =>
        row.original.username ? (
          <span className="text-xs">{row.original.username}</span>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: true,
    },
    {
      id: 'process',
      accessorKey: 'process',
      header: 'Process',
      cell: ({ row }) =>
        row.original.process ? (
          <code className="text-xs truncate max-w-[120px] block">{row.original.process}</code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: true,
    },
    {
      id: 'matched_rules',
      header: 'Rules',
      cell: ({ row }) => {
        const count = row.original.matched_rules?.length ?? 0;
        return count > 0 ? (
          <Badge variant="secondary" className="text-xs">
            {count}
          </Badge>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        );
      },
      enableSorting: false,
    },
  ];
}
