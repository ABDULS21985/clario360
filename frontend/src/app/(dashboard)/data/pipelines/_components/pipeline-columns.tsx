'use client';

import Link from 'next/link';
import { type ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { type Pipeline } from '@/lib/data-suite';
import { formatMaybeCompact, formatMaybeDateTime, humanizeCronOrFrequency } from '@/lib/data-suite/utils';

interface PipelineColumnOptions {
  runningId: string | null;
  onRun: (pipeline: Pipeline) => void;
}

export function buildPipelineColumns({
  runningId,
  onRun,
}: PipelineColumnOptions): ColumnDef<Pipeline>[] {
  return [
    {
      id: 'name',
      header: 'Pipeline',
      accessorKey: 'name',
      cell: ({ row }) => (
        <div>
          <Link href={`/data/pipelines/${row.original.id}`} className="font-medium hover:text-primary">
            {row.original.name}
          </Link>
          <div className="text-xs text-muted-foreground">
            {row.original.config.source_table || row.original.config.source_query || 'Source configured'}
          </div>
        </div>
      ),
    },
    {
      id: 'type',
      header: 'Type',
      cell: ({ row }) => <span className="capitalize">{row.original.type}</span>,
    },
    {
      id: 'status',
      header: 'Status',
      cell: ({ row }) => <Badge variant="outline">{row.original.status}</Badge>,
    },
    {
      id: 'schedule',
      header: 'Schedule',
      cell: ({ row }) => humanizeCronOrFrequency(row.original.schedule),
    },
    {
      id: 'runs',
      header: 'Runs',
      cell: ({ row }) => row.original.total_runs.toLocaleString(),
    },
    {
      id: 'records',
      header: 'Processed',
      cell: ({ row }) => formatMaybeCompact(row.original.total_records_processed),
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
          {runningId === row.original.id ? 'Starting…' : 'Run now'}
        </Button>
      ),
    },
  ];
}
