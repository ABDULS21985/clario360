'use client';

import { useMemo } from 'react';
import { AlertTriangle, CheckCircle2, RefreshCcw } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { usePollingOperation } from '@/lib/data-suite';
import { dataSuiteApi, formatMaybeBytes, formatMaybeCompact, formatMaybeDurationMs, formatMaybeRelative, type DataSource, type SyncHistory } from '@/lib/data-suite';

interface SyncProgressIndicatorProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  source: DataSource | null;
  onComplete?: () => void;
}

export function SyncProgressIndicator({
  open,
  onOpenChange,
  source,
  onComplete,
}: SyncProgressIndicatorProps) {
  const operation = usePollingOperation<SyncHistory[]>({
    enabled: open && Boolean(source?.id),
    intervalMs: 3000,
    fetcher: () => dataSuiteApi.listSourceSyncHistory(source!.id, 1),
    isDone: (items) => {
      const latest = items[0];
      return latest ? latest.status !== 'running' : false;
    },
    onData: (items) => {
      const latest = items[0];
      if (latest && latest.status !== 'running') {
        onComplete?.();
      }
    },
  });

  const latest = useMemo(() => operation.data?.[0] ?? null, [operation.data]);
  const progressValue = latest
    ? Math.min(100, latest.status === 'running' ? 30 + Math.min(60, latest.tables_synced * 10) : 100)
    : 10;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Sync progress{source ? `: ${source.name}` : ''}</DialogTitle>
        </DialogHeader>

        {!latest ? (
          <div className="flex items-center gap-3 rounded-lg border bg-muted/20 p-4">
            <Spinner />
            <div>
              <p className="font-medium">Fetching latest sync run…</p>
              <p className="text-sm text-muted-foreground">Polling sync history every 3 seconds.</p>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium capitalize">{latest.status}</p>
                  <p className="text-xs text-muted-foreground">
                    Started {formatMaybeRelative(latest.started_at)} • {latest.sync_type}
                  </p>
                </div>
                {latest.status === 'running' ? <Spinner size="sm" /> : latest.status === 'success' ? <CheckCircle2 className="h-5 w-5 text-emerald-600" /> : null}
              </div>
              <Progress className="mt-4 h-2" value={progressValue} />
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <Metric label="Rows read" value={formatMaybeCompact(latest.rows_read)} />
                <Metric label="Rows written" value={formatMaybeCompact(latest.rows_written)} />
                <Metric label="Tables synced" value={`${latest.tables_synced}`} />
                <Metric label="Duration" value={formatMaybeDurationMs(latest.duration_ms)} />
                <Metric label="Transferred" value={formatMaybeBytes(latest.bytes_transferred)} />
                <Metric label="Errors" value={`${latest.error_count}`} />
              </div>
            </div>

            {latest.status === 'failed' || latest.status === 'partial' ? (
              <Alert className="border-rose-200 bg-rose-50">
                <AlertTriangle className="h-4 w-4 text-rose-600" />
                <AlertTitle className="text-rose-700">Sync failed</AlertTitle>
                <AlertDescription className="space-y-2 text-rose-700">
                  <p>
                    {latest.error_count > 0
                      ? `${latest.error_count} error(s) were reported during sync.`
                      : 'The sync did not complete successfully.'}
                  </p>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={async () => {
                      if (!source) {
                        return;
                      }
                      await dataSuiteApi.syncSource(source.id, 'full');
                      operation.start();
                    }}
                  >
                    <RefreshCcw className="mr-1.5 h-4 w-4" />
                    Retry sync
                  </Button>
                </AlertDescription>
              </Alert>
            ) : null}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-muted/20 p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  );
}
