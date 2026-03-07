'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { type QualityRule } from '@/lib/data-suite';
import { formatMaybeDateTime, qualitySeverityVisuals } from '@/lib/data-suite/utils';

interface QualityRuleColumnOptions {
  runningId: string | null;
  onRun: (rule: QualityRule) => void;
}

export function buildQualityRuleColumns({
  runningId,
  onRun,
}: QualityRuleColumnOptions): ColumnDef<QualityRule>[] {
  return [
    {
      id: 'name',
      header: 'Rule',
      accessorKey: 'name',
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.name}</div>
          <div className="text-xs text-muted-foreground">
            {row.original.rule_type} {row.original.column_name ? `• ${row.original.column_name}` : ''}
          </div>
        </div>
      ),
    },
    {
      id: 'severity',
      header: 'Severity',
      cell: ({ row }) => {
        const severity = qualitySeverityVisuals[row.original.severity];
        return (
          <Badge variant="outline" className={severity.className}>
            {severity.label}
          </Badge>
        );
      },
    },
    {
      id: 'enabled',
      header: 'Enabled',
      cell: ({ row }) => (row.original.enabled ? 'Yes' : 'No'),
    },
    {
      id: 'last_status',
      header: 'Last status',
      cell: ({ row }) => row.original.last_status ?? 'never run',
    },
    {
      id: 'last_run_at',
      header: 'Last run',
      cell: ({ row }) => formatMaybeDateTime(row.original.last_run_at),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button type="button" size="sm" variant="outline" onClick={() => onRun(row.original)} disabled={runningId === row.original.id}>
          {runningId === row.original.id ? 'Running…' : 'Run now'}
        </Button>
      ),
    },
  ];
}
