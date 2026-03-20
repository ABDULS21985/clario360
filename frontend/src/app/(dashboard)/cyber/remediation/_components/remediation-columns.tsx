'use client';

import { ColumnDef, Row } from '@tanstack/react-table';
import Link from 'next/link';
import { MoreHorizontal, CheckCircle, PlayCircle } from 'lucide-react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { timeAgo } from '@/lib/utils';
import { RemediationLifecycleBadge } from './remediation-lifecycle-badge';
import type { RemediationAction } from '@/types/cyber';

interface RemediationColumnOptions {
  onApprove?: (action: RemediationAction) => void;
  onExecute?: (action: RemediationAction) => void;
}

export function getRemediationColumns(options: RemediationColumnOptions = {}): ColumnDef<RemediationAction>[] {
  return [
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }: { row: Row<RemediationAction> }) => (
        <SeverityIndicator severity={row.original.severity} showLabel />
      ),
      enableSorting: true,
    },
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Remediation',
      cell: ({ row }: { row: Row<RemediationAction> }) => {
        const action = row.original;
        return (
          <div>
            <Link href={`/cyber/remediation/${action.id}`} className="font-medium hover:underline">
              {action.title}
            </Link>
            <p className="text-xs text-muted-foreground capitalize">{action.type.replace(/_/g, ' ')}</p>
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }: { row: Row<RemediationAction> }) => (
        <RemediationLifecycleBadge status={row.original.status} />
      ),
      enableSorting: true,
    },
    {
      id: 'plan.reversible',
      header: 'Reversible',
      cell: ({ row }: { row: Row<RemediationAction> }) => (
        <span className={`text-xs ${row.original.plan.reversible ? 'text-green-600' : 'text-orange-600'}`}>
          {row.original.plan.reversible ? '✓ Yes' : '✗ No'}
        </span>
      ),
    },
    {
      id: 'created_by_name',
      header: 'Created By',
      cell: ({ row }: { row: Row<RemediationAction> }) => (
        <span className="text-sm text-muted-foreground">{row.original.created_by_name ?? '—'}</span>
      ),
    },
    {
      id: 'created_at',
      accessorKey: 'created_at',
      header: 'Created',
      cell: ({ row }: { row: Row<RemediationAction> }) => (
        <span className="text-sm text-muted-foreground">{timeAgo(row.original.created_at)}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<RemediationAction> }) => {
        const action = row.original;
        const canApprove = action.status === 'pending_approval';
        const canExecute = action.status === 'dry_run_completed' || action.status === 'approved';

        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {canApprove && (
                <DropdownMenuItem onClick={() => options.onApprove?.(action)}>
                  <CheckCircle className="mr-2 h-3.5 w-3.5 text-green-600" /> Approve
                </DropdownMenuItem>
              )}
              {canExecute && (
                <DropdownMenuItem onClick={() => options.onExecute?.(action)}>
                  <PlayCircle className="mr-2 h-3.5 w-3.5 text-blue-600" /> Execute
                </DropdownMenuItem>
              )}
              <DropdownMenuItem asChild>
                <Link href={`/cyber/remediation/${action.id}`}>View Details</Link>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
      enableSorting: false,
    },
  ];
}
