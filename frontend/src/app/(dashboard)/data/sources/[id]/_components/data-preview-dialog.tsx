'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Lock, SearchX } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { type DataModel, type DiscoveredTable, dataSuiteApi } from '@/lib/data-suite';

interface DataPreviewDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  model?: DataModel | null;
  table: DiscoveredTable | null;
}

export function DataPreviewDialog({
  open,
  onOpenChange,
  model,
  table,
}: DataPreviewDialogProps) {
  const enabled = open && Boolean(model?.id) && Boolean(table);

  const previewQuery = useQuery({
    queryKey: ['data-preview', model?.id, table?.name],
    enabled,
    queryFn: () =>
      dataSuiteApi.exploreModel(model!.id, {
        columns: table?.columns.map((column) => column.name) ?? [],
        limit: 20,
      }),
  });

  const maskedColumns = useMemo(
    () => previewQuery.data?.metadata.columns_masked ?? [],
    [previewQuery.data?.metadata.columns_masked],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl">
        <DialogHeader>
          <DialogTitle>Preview Data{table ? `: ${table.name}` : ''}</DialogTitle>
        </DialogHeader>

        {!model ? (
          <Alert>
            <SearchX className="h-4 w-4" />
            <AlertTitle>Preview unavailable</AlertTitle>
            <AlertDescription>
              This source table does not have a derived model yet. Derive a model first, then preview rows through the governed analytics API.
            </AlertDescription>
          </Alert>
        ) : previewQuery.isLoading ? (
          <LoadingSkeleton variant="chart" />
        ) : previewQuery.error ? (
          <ErrorState message={previewQuery.error.message} onRetry={() => void previewQuery.refetch()} />
        ) : previewQuery.data ? (
          <div className="space-y-3">
            {previewQuery.data.metadata.pii_masking_applied ? (
              <Alert className="border-sky-200 bg-sky-50">
                <Lock className="h-4 w-4 text-sky-600" />
                <AlertTitle className="text-sky-700">PII masked</AlertTitle>
                <AlertDescription className="text-sky-700">
                  Masked columns: {maskedColumns.join(', ') || 'None'}
                </AlertDescription>
              </Alert>
            ) : null}

            <ScrollArea className="h-[520px] rounded-lg border">
              <table className="min-w-full text-sm">
                <thead className="sticky top-0 z-10 bg-background">
                  <tr>
                    {previewQuery.data.columns.map((column) => (
                      <th key={column.name} className="border-b px-3 py-2 text-left font-medium">
                        <div className="flex items-center gap-2">
                          <span>{column.name}</span>
                          {column.masked ? <Lock className="h-3.5 w-3.5 text-sky-600" /> : null}
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {previewQuery.data.rows.map((row, rowIndex) => (
                    <tr key={`${rowIndex}-${Object.keys(row).join('-')}`} className="border-b">
                      {previewQuery.data.columns.map((column) => (
                        <td key={`${rowIndex}-${column.name}`} className="px-3 py-2 align-top font-mono text-xs">
                          {`${row[column.name] ?? '—'}`}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </ScrollArea>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
