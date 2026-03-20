'use client';

import Link from 'next/link';
import { MoreHorizontal, PlayCircle, RefreshCcw, TestTubeDiagonal, Trash2 } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import { type ConnectionTestResult, type DataSource } from '@/lib/data-suite';
import {
  formatMaybeBytes,
  formatMaybeCompact,
  formatMaybeRelative,
  getSourceTypeVisual,
  getStatusTone,
  humanizeCronOrFrequency,
} from '@/lib/data-suite/utils';
import { truncate } from '@/lib/utils';
import { TestConnectionInline } from '@/app/(dashboard)/data/sources/_components/test-connection-inline';

interface SourceCardProps {
  source: DataSource;
  testing: boolean;
  testResult?: ConnectionTestResult | null;
  testError?: string | null;
  onTest: (source: DataSource) => void;
  onSync: (source: DataSource) => void;
  onEdit: (source: DataSource) => void;
  onDelete: (source: DataSource) => void;
}

export function SourceCard({
  source,
  testing,
  testResult,
  testError,
  onTest,
  onSync,
  onEdit,
  onDelete,
}: SourceCardProps) {
  const typeVisual = getSourceTypeVisual(source.type);
  const Icon = typeVisual.icon;

  return (
    <div className="rounded-xl border bg-card p-5 shadow-sm transition-colors hover:border-primary/30">
      <div className="flex items-start justify-between gap-4">
        <Link href={`/data/sources/${source.id}`} className="min-w-0 flex-1">
          <div className="flex items-center gap-3">
            <div className={`rounded-full bg-muted p-2 ${typeVisual.accentClass}`}>
              <Icon className="h-5 w-5" data-testid={`source-type-icon-${source.type}`} />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="truncate font-semibold">{source.name}</h3>
                <span className="inline-flex items-center gap-1 text-xs font-medium text-muted-foreground">
                  <span
                    className={`h-2.5 w-2.5 rounded-full ${getStatusTone(source.status)}`}
                    data-testid={`source-status-dot-${source.status}`}
                  />
                  {source.status}
                </span>
              </div>
              <p className="text-sm text-muted-foreground">{typeVisual.label}</p>
            </div>
          </div>
        </Link>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="ghost" size="icon">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => onEdit(source)}>Edit</DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href={`/data/sources/${source.id}?tab=schema`}>View schema</Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href={`/data/sources/${source.id}?tab=pipelines`}>View pipelines</Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-rose-700 focus:text-rose-700" onClick={() => onDelete(source)}>
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Link href={`/data/sources/${source.id}`} className="mt-4 block">
        <p className="text-sm text-muted-foreground">
          {source.description ? truncate(source.description, 100) : 'No description provided.'}
        </p>
      </Link>

      <div className="mt-4 flex flex-wrap gap-3 text-sm text-muted-foreground">
        <span>{source.table_count ?? 0} tables</span>
        <span>{formatMaybeCompact(source.total_row_count)} rows</span>
        <span>{formatMaybeBytes(source.total_size_bytes)}</span>
      </div>

      <div className="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground">
        <span>Last sync: {formatMaybeRelative(source.last_synced_at)}</span>
        <span>{humanizeCronOrFrequency(source.sync_frequency)}</span>
      </div>

      {(source.last_error || source.last_sync_error) && source.status === 'error' ? (
        <div className="mt-3 text-xs text-rose-700">{truncate(source.last_error || source.last_sync_error || '', 96)}</div>
      ) : null}

      <div className="mt-4 flex items-center gap-2">
        <Button type="button" variant="outline" size="sm" onClick={() => onTest(source)} disabled={testing}>
          <TestTubeDiagonal className="mr-1.5 h-4 w-4" />
          {testing ? 'Testing…' : 'Test'}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={() => onSync(source)}>
          <RefreshCcw className="mr-1.5 h-4 w-4" />
          Sync
        </Button>
        <Button type="button" variant="ghost" size="sm" asChild>
          <Link href={`/data/sources/${source.id}`}>
            <PlayCircle className="mr-1.5 h-4 w-4" />
            Open
          </Link>
        </Button>
      </div>

      <TestConnectionInline
        loading={testing}
        result={testResult}
        error={testError}
        onEdit={() => onEdit(source)}
      />
    </div>
  );
}
