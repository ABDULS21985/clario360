'use client';

import Link from 'next/link';
import { ColumnDef, Row } from '@tanstack/react-table';
import { MoreHorizontal, UserCheck, MessageSquare, ArrowUpCircle } from 'lucide-react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { timeAgo } from '@/lib/utils';
import type { CyberAlert } from '@/types/cyber';

interface AlertColumnOptions {
  onAssign?: (alert: CyberAlert) => void;
  onChangeStatus?: (alert: CyberAlert) => void;
  onEscalate?: (alert: CyberAlert) => void;
}

function AlertActions({ alert, onAssign, onChangeStatus, onEscalate }: AlertColumnOptions & { alert: CyberAlert }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => onAssign?.(alert)}>
          <UserCheck className="mr-2 h-3.5 w-3.5" /> Assign
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onChangeStatus?.(alert)}>
          <MessageSquare className="mr-2 h-3.5 w-3.5" /> Change Status
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="text-amber-600"
          onClick={() => onEscalate?.(alert)}
        >
          <ArrowUpCircle className="mr-2 h-3.5 w-3.5" /> Escalate
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function getAlertColumns(options: AlertColumnOptions = {}): ColumnDef<CyberAlert>[] {
  return [
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }: { row: Row<CyberAlert> }) => (
        <SeverityIndicator severity={row.original.severity} showLabel />
      ),
      enableSorting: true,
    },
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Alert',
      cell: ({ row }: { row: Row<CyberAlert> }) => {
        const alert = row.original;
        return (
          <div className="min-w-0">
            <Link
              href={`/cyber/alerts/${alert.id}`}
              className="block truncate font-medium hover:underline max-w-[260px]"
            >
              {alert.title}
            </Link>
            {alert.mitre_technique_id && (
              <span className="inline-flex items-center rounded-sm bg-muted px-1 py-0.5 text-xs text-muted-foreground">
                {alert.mitre_technique_id}
              </span>
            )}
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }: { row: Row<CyberAlert> }) => (
        <StatusBadge status={row.original.status} />
      ),
      enableSorting: true,
    },
    {
      id: 'asset',
      header: 'Asset',
      cell: ({ row }: { row: Row<CyberAlert> }) => {
        const alert = row.original;
        if (!alert.asset_name) return <span className="text-muted-foreground">—</span>;
        return (
          <Link
            href={alert.asset_id ? `/cyber/assets/${alert.asset_id}` : '#'}
            className="text-sm hover:underline truncate block max-w-[160px]"
          >
            {alert.asset_name}
          </Link>
        );
      },
    },
    {
      id: 'assigned_to',
      header: 'Assigned',
      cell: ({ row }: { row: Row<CyberAlert> }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.assigned_to_name ?? '—'}
        </span>
      ),
    },
    {
      id: 'confidence_score',
      accessorKey: 'confidence_score',
      header: 'Confidence',
      cell: ({ row }: { row: Row<CyberAlert> }) => {
        const score = row.original.confidence_score;
        const color = score >= 80 ? 'bg-red-500' : score >= 60 ? 'bg-orange-500' : 'bg-yellow-500';
        return (
          <div className="flex items-center gap-2">
            <div className="h-1.5 w-16 rounded-full bg-muted">
              <div className={`h-full rounded-full ${color}`} style={{ width: `${score}%` }} />
            </div>
            <span className="text-xs tabular-nums">{score}%</span>
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'created_at',
      accessorKey: 'created_at',
      header: 'Created',
      cell: ({ row }: { row: Row<CyberAlert> }) => (
        <span className="text-sm text-muted-foreground">{timeAgo(row.original.created_at)}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<CyberAlert> }) => (
        <AlertActions alert={row.original} {...options} />
      ),
      enableSorting: false,
    },
  ];
}
