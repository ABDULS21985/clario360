'use client';

import Link from 'next/link';
import { type ColumnDef } from '@tanstack/react-table';
import { ArrowRightLeft, GitCompareArrows, History, ShieldCheck } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { AIDashboardModelRow, AIDriftLevel } from '@/types/ai-governance';

interface ModelColumnOptions {
  busyModelId?: string | null;
  onPromote: (row: AIDashboardModelRow) => void;
  onRollback: (row: AIDashboardModelRow) => void;
  onStartShadow: (row: AIDashboardModelRow) => void;
}

function driftVariant(level: AIDriftLevel) {
  switch (level) {
    case 'significant':
      return 'destructive';
    case 'moderate':
      return 'warning';
    case 'low':
      return 'secondary';
    default:
      return 'success';
  }
}

function riskVariant(level: string) {
  switch (level) {
    case 'critical':
      return 'destructive';
    case 'high':
      return 'warning';
    case 'medium':
      return 'secondary';
    default:
      return 'outline';
  }
}

export function createModelColumns({
  busyModelId,
  onPromote,
  onRollback,
  onStartShadow,
}: ModelColumnOptions): ColumnDef<AIDashboardModelRow>[] {
  return [
    {
      id: 'name',
      header: 'Model',
      accessorKey: 'name',
      enableSorting: true,
      cell: ({ row }) => (
        <div className="space-y-1">
          <div className="font-medium">{row.original.name}</div>
          <div className="font-mono text-xs text-muted-foreground">{row.original.slug}</div>
        </div>
      ),
    },
    {
      id: 'suite',
      header: 'Suite',
      accessorKey: 'suite',
      cell: ({ row }) => <Badge variant="secondary">{row.original.suite}</Badge>,
    },
    {
      id: 'type',
      header: 'Type',
      accessorKey: 'type',
      cell: ({ row }) => <Badge variant="outline">{row.original.type.replaceAll('_', ' ')}</Badge>,
    },
    {
      id: 'versions',
      header: 'Versions',
      cell: ({ row }) => (
        <div className="space-y-1 text-sm">
          <div>
            Prod:{' '}
            <span className="font-medium">
              {row.original.production_version ? `v${row.original.production_version.version_number}` : 'None'}
            </span>
          </div>
          <div className="text-muted-foreground">
            Shadow:{' '}
            <span className="font-medium text-foreground">
              {row.original.shadow_version ? `v${row.original.shadow_version.version_number}` : 'Inactive'}
            </span>
          </div>
        </div>
      ),
    },
    {
      id: 'predictions_24h',
      header: 'Predictions',
      accessorKey: 'predictions_24h',
      enableSorting: true,
      cell: ({ row }) => (
        <div className="space-y-1 text-sm">
          <div className="font-medium">{row.original.predictions_24h.toLocaleString()}</div>
          <div className="text-muted-foreground">
            Avg conf {row.original.avg_confidence ? `${Math.round(row.original.avg_confidence * 100)}%` : 'n/a'}
          </div>
        </div>
      ),
    },
    {
      id: 'drift_status',
      header: 'Drift',
      accessorKey: 'drift_status',
      cell: ({ row }) => <Badge variant={driftVariant(row.original.drift_status)}>{row.original.drift_status}</Badge>,
    },
    {
      id: 'risk_tier',
      header: 'Risk Tier',
      accessorKey: 'risk_tier',
      cell: ({ row }) => <Badge variant={riskVariant(row.original.risk_tier)}>{row.original.risk_tier}</Badge>,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const isBusy = busyModelId === row.original.id;
        return (
          <div className="flex flex-wrap justify-end gap-2">
            <Button asChild variant="ghost" size="sm">
              <Link href={`/admin/ai-governance/${row.original.id}`}>Details</Link>
            </Button>
            {row.original.shadow_version ? (
              <Button variant="outline" size="sm" disabled={isBusy} onClick={() => onPromote(row.original)}>
                <ShieldCheck className="mr-1.5 h-3.5 w-3.5" />
                Promote
              </Button>
            ) : (
              <Button variant="outline" size="sm" disabled={isBusy} onClick={() => onStartShadow(row.original)}>
                <GitCompareArrows className="mr-1.5 h-3.5 w-3.5" />
                Start Shadow
              </Button>
            )}
            {row.original.production_version ? (
              <Button variant="outline" size="sm" disabled={isBusy} onClick={() => onRollback(row.original)}>
                <History className="mr-1.5 h-3.5 w-3.5" />
                Rollback
              </Button>
            ) : null}
            <Button asChild variant="ghost" size="sm">
              <Link href={`/admin/ai-governance/${row.original.id}#shadow`}>
                <ArrowRightLeft className="mr-1.5 h-3.5 w-3.5" />
                Shadow
              </Link>
            </Button>
          </div>
        );
      },
    },
  ];
}
