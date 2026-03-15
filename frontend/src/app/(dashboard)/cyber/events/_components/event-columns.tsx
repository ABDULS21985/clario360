'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import type { SecurityEvent } from '@/types/cyber';

const PROTOCOL_COLORS: Record<string, string> = {
  TCP: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  UDP: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
  ICMP: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
  HTTP: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  DNS: 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-400',
};

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
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }) => (
        <SeverityIndicator severity={row.original.severity} showLabel size="sm" />
      ),
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
      header: 'Dest',
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
      id: 'protocol',
      accessorKey: 'protocol',
      header: 'Proto',
      cell: ({ row }) => {
        const proto = row.original.protocol;
        if (!proto) return <span className="text-xs text-muted-foreground">—</span>;
        const cls = PROTOCOL_COLORS[proto.toUpperCase()] ?? 'bg-gray-100 text-gray-700';
        return (
          <span className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${cls}`}>
            {proto}
          </span>
        );
      },
      enableSorting: false,
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
          <code className="block max-w-[80px] truncate text-xs sm:max-w-[120px]">
            {row.original.process}
          </code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: true,
    },
    {
      id: 'parent_process',
      accessorKey: 'parent_process',
      header: 'Parent',
      cell: ({ row }) =>
        row.original.parent_process ? (
          <code className="block max-w-[100px] truncate text-xs">{row.original.parent_process}</code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: false,
    },
    {
      id: 'command_line',
      accessorKey: 'command_line',
      header: 'Command',
      cell: ({ row }) =>
        row.original.command_line ? (
          <code className="block max-w-[160px] truncate text-xs" title={row.original.command_line}>
            {row.original.command_line}
          </code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: false,
    },
    {
      id: 'file_hash',
      accessorKey: 'file_hash',
      header: 'File Hash',
      cell: ({ row }) =>
        row.original.file_hash ? (
          <code className="block max-w-[100px] truncate text-xs" title={row.original.file_hash}>
            {row.original.file_hash}
          </code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: false,
    },
    {
      id: 'asset_id',
      accessorKey: 'asset_id',
      header: 'Asset',
      cell: ({ row }) =>
        row.original.asset_id ? (
          <code className="block max-w-[80px] truncate text-xs" title={row.original.asset_id}>
            {row.original.asset_id.slice(0, 8)}…
          </code>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: false,
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
