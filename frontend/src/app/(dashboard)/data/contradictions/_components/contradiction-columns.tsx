'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { type Contradiction } from '@/lib/data-suite';
import { formatMaybeDateTime, qualitySeverityVisuals } from '@/lib/data-suite/utils';

interface ContradictionColumnOptions {
  onOpen: (contradiction: Contradiction) => void;
}

export function buildContradictionColumns({
  onOpen,
}: ContradictionColumnOptions): ColumnDef<Contradiction>[] {
  return [
    {
      id: 'type',
      header: 'Type',
      cell: ({ row }) => <Badge variant="outline">{row.original.type}</Badge>,
    },
    {
      id: 'title',
      header: 'Title',
      accessorKey: 'title',
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.title}</div>
          <div className="text-xs text-muted-foreground">{row.original.description}</div>
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
      id: 'sources',
      header: 'Sources',
      cell: ({ row }) => (
        <div className="text-sm text-muted-foreground">
          <div>{row.original.source_a.source_name}</div>
          <div>{row.original.source_b.source_name}</div>
        </div>
      ),
    },
    {
      id: 'affected_records',
      header: 'Affected',
      cell: ({ row }) => row.original.affected_records.toLocaleString(),
    },
    {
      id: 'confidence',
      header: 'Confidence',
      cell: ({ row }) => `${(row.original.confidence_score * 100).toFixed(0)}%`,
    },
    {
      id: 'status',
      header: 'Status',
      cell: ({ row }) => <Badge variant="outline">{row.original.status}</Badge>,
    },
    {
      id: 'created_at',
      header: 'Created',
      cell: ({ row }) => formatMaybeDateTime(row.original.created_at),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button type="button" size="sm" variant="outline" onClick={() => onOpen(row.original)}>
          Investigate
        </Button>
      ),
    },
  ];
}
