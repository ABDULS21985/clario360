'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { Bot } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { RelativeTime } from '@/components/shared/relative-time';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import type { AuditLog } from '@/types/models';

const serviceColors: Record<string, string> = {
  'iam-service': 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  'audit-service': 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  'workflow-engine': 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  'notification-service': 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  'file-service': 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400',
  'cyber-service': 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  'data-service': 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
};

export function formatAuditAction(action: string): string {
  return action
    .split('.')
    .map((part) =>
      part
        .split('_')
        .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
        .join(' ')
    )
    .join(' › ');
}

export function getAuditColumns(
  onRowClick: (log: AuditLog) => void
): ColumnDef<AuditLog>[] {
  return [
    {
      id: 'created_at',
      accessorKey: 'created_at',
      header: 'Timestamp',
      size: 140,
      cell: ({ row }) => <RelativeTime date={row.original.created_at} />,
      enableSorting: true,
    },
    {
      id: 'user',
      header: 'User',
      size: 180,
      cell: ({ row }) => {
        const log = row.original;
        const isSystem = !log.user_id || log.user_email === 'system';
        if (isSystem) {
          return (
            <span className="inline-flex items-center gap-1.5 text-sm text-muted-foreground">
              <Bot className="h-3.5 w-3.5" />
              System
            </span>
          );
        }
        return (
          <div className="min-w-0">
            <p className="text-sm truncate">{log.user_email}</p>
          </div>
        );
      },
      enableSorting: false,
    },
    {
      id: 'service',
      accessorKey: 'service',
      header: 'Service',
      size: 140,
      cell: ({ row }) => {
        const service = row.original.service ?? 'unknown';
        const colorClass = serviceColors[service] ?? 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400';
        return (
          <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${colorClass}`}>
            {service}
          </span>
        );
      },
      enableSorting: false,
    },
    {
      id: 'action',
      accessorKey: 'action',
      header: 'Action',
      size: 200,
      cell: ({ row }) => (
        <span className="text-sm">{formatAuditAction(row.original.action)}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'resource',
      header: 'Resource',
      size: 180,
      cell: ({ row }) => {
        const log = row.original;
        return (
          <div className="min-w-0">
            <span className="text-sm text-muted-foreground">{log.resource_type}</span>
            {log.resource_id && (
              <span className="text-xs font-mono text-muted-foreground ml-1">
                · {log.resource_id.slice(0, 8)}
              </span>
            )}
          </div>
        );
      },
      enableSorting: false,
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      size: 100,
      cell: ({ row }) => {
        const severity = row.original.severity;
        if (!severity) return <Badge variant="outline" className="text-xs">info</Badge>;
        return <SeverityIndicator severity={severity === 'warning' ? 'medium' : severity === 'high' ? 'high' : severity === 'critical' ? 'critical' : 'info'} />;
      },
      enableSorting: true,
    },
    {
      id: 'ip_address',
      accessorKey: 'ip_address',
      header: 'IP Address',
      size: 130,
      cell: ({ row }) => (
        <span className="font-mono text-xs text-muted-foreground">
          {row.original.ip_address}
        </span>
      ),
      enableSorting: false,
    },
  ];
}
