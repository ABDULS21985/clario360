'use client';

import Link from 'next/link';
import { type ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { type DataModel } from '@/lib/data-suite';
import { formatMaybeDateTime, getClassificationBadge } from '@/lib/data-suite/utils';

export function buildModelColumns(): ColumnDef<DataModel>[] {
  return [
    {
      id: 'name',
      header: 'Model',
      accessorKey: 'name',
      cell: ({ row }) => (
        <div>
          <Link href={`/data/models/${row.original.id}`} className="font-medium hover:text-primary">
            {row.original.display_name || row.original.name}
          </Link>
          <div className="text-xs text-muted-foreground">
            {row.original.source_table ?? 'Unmapped source table'} • v{row.original.version}
          </div>
        </div>
      ),
    },
    {
      id: 'status',
      header: 'Status',
      accessorKey: 'status',
      cell: ({ row }) => <Badge variant="outline">{row.original.status}</Badge>,
    },
    {
      id: 'classification',
      header: 'Classification',
      cell: ({ row }) => {
        const badge = getClassificationBadge(row.original.data_classification);
        return (
          <Badge variant="outline" className={badge.className}>
            {badge.label}
          </Badge>
        );
      },
    },
    {
      id: 'fields',
      header: 'Fields',
      cell: ({ row }) => row.original.field_count.toLocaleString(),
    },
    {
      id: 'pii',
      header: 'PII',
      cell: ({ row }) => (row.original.contains_pii ? row.original.pii_columns.length.toLocaleString() : '—'),
    },
    {
      id: 'updated_at',
      header: 'Updated',
      accessorKey: 'updated_at',
      cell: ({ row }) => formatMaybeDateTime(row.original.updated_at),
    },
  ];
}
