'use client';

import Link from 'next/link';
import { MoreHorizontal } from 'lucide-react';
import { type ColumnDef } from '@tanstack/react-table';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { type Pipeline } from '@/lib/data-suite';
import { formatMaybeCompact, formatMaybeDateTime, humanizeCronOrFrequency } from '@/lib/data-suite/utils';
import {
  PipelineStatusIndicator,
  pipelineCanResume,
} from '@/app/(dashboard)/data/pipelines/_components/pipeline-status-indicator';

interface PipelineColumnOptions {
  runningId: string | null;
  mutatingId: string | null;
  onRun: (pipeline: Pipeline) => void;
  onPause: (pipeline: Pipeline) => void;
  onResume: (pipeline: Pipeline) => void;
  onDelete: (pipeline: Pipeline) => void;
}

export function buildPipelineColumns({
  runningId,
  mutatingId,
  onRun,
  onPause,
  onResume,
  onDelete,
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
      cell: ({ row }) => (
        <PipelineStatusIndicator status={row.original.status} lastRunStatus={row.original.last_run_status} compact />
      ),
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
        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => onRun(row.original)}
            disabled={runningId === row.original.id}
          >
            {runningId === row.original.id ? 'Starting…' : 'Run now'}
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button type="button" size="icon" variant="ghost" aria-label="Pipeline actions">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {pipelineCanResume(row.original) ? (
                <DropdownMenuItem
                  onClick={() => onResume(row.original)}
                  disabled={mutatingId === row.original.id}
                >
                  {mutatingId === row.original.id ? 'Resuming…' : 'Resume'}
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem
                  onClick={() => onPause(row.original)}
                  disabled={mutatingId === row.original.id || row.original.status !== 'active'}
                >
                  {mutatingId === row.original.id ? 'Pausing…' : 'Pause'}
                </DropdownMenuItem>
              )}
              <DropdownMenuItem
                onClick={() => onDelete(row.original)}
                disabled={mutatingId === row.original.id}
                className="text-destructive focus:text-destructive"
              >
                {mutatingId === row.original.id ? 'Deleting…' : 'Delete'}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      ),
    },
  ];
}
