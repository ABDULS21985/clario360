'use client';

import { AlertTriangle, Clock3, Database, ShieldCheck } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { type DataSource, type SourceStats, type SyncHistory } from '@/lib/data-suite';
import {
  formatMaybeBytes,
  formatMaybeCompact,
  formatMaybeDateTime,
  formatMaybeDurationMs,
  getSourceTypeVisual,
} from '@/lib/data-suite/utils';

interface SourceOverviewTabProps {
  source: DataSource;
  stats: SourceStats | null;
  syncHistory: SyncHistory[];
}

export function SourceOverviewTab({
  source,
  stats,
  syncHistory,
}: SourceOverviewTabProps) {
  const typeVisual = getSourceTypeVisual(source.type);
  const latestSync = syncHistory[0];

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
        <MetricCard label="Tables" value={formatMaybeCompact(stats?.table_count ?? source.table_count)} icon={Database} />
        <MetricCard label="Rows" value={formatMaybeCompact(stats?.total_row_count ?? source.total_row_count)} icon={Database} />
        <MetricCard label="Size" value={formatMaybeBytes(stats?.total_size_bytes ?? source.total_size_bytes)} icon={Database} />
        <MetricCard label="Last Sync" value={formatMaybeDateTime(stats?.last_synced_at ?? source.last_synced_at)} icon={Clock3} />
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader>
            <CardTitle>Source Properties</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <PropertyRow label="Type" value={typeVisual.label} />
            <PropertyRow label="Status" value={source.status} />
            <PropertyRow label="Sync Frequency" value={source.sync_frequency ?? 'Manual only'} />
            <PropertyRow label="Schema Discovered" value={formatMaybeDateTime(source.schema_discovered_at)} />
            <PropertyRow label="Created" value={formatMaybeDateTime(source.created_at)} />
            <PropertyRow label="Updated" value={formatMaybeDateTime(source.updated_at)} />
            <PropertyRow
              label="Tags"
              value={source.tags.length > 0 ? source.tags.join(', ') : '—'}
              className="md:col-span-2"
            />
            <PropertyRow
              label="Description"
              value={source.description || 'No description provided.'}
              className="md:col-span-2"
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Connection Health</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-emerald-600" />
              <span className="text-sm">
                {source.status === 'active' ? 'Connection validated and active.' : `Current status: ${source.status}`}
              </span>
            </div>
            {source.last_error || source.last_sync_error ? (
              <Alert className="border-rose-200 bg-rose-50">
                <AlertTriangle className="h-4 w-4 text-rose-600" />
                <AlertTitle className="text-rose-700">Latest error</AlertTitle>
                <AlertDescription className="text-rose-700">
                  {source.last_error || source.last_sync_error}
                </AlertDescription>
              </Alert>
            ) : (
              <Alert className="border-emerald-200 bg-emerald-50">
                <ShieldCheck className="h-4 w-4 text-emerald-600" />
                <AlertTitle className="text-emerald-700">No active connector errors</AlertTitle>
                <AlertDescription className="text-emerald-700">
                  The source does not report recent connection or sync failures.
                </AlertDescription>
              </Alert>
            )}
            {latestSync ? (
              <div className="rounded-lg border bg-muted/20 p-3 text-sm">
                <div className="font-medium">Latest sync</div>
                <div className="mt-2 grid gap-2 text-muted-foreground">
                  <span>Status: {latestSync.status}</span>
                  <span>Rows written: {formatMaybeCompact(latestSync.rows_written)}</span>
                  <span>Duration: {formatMaybeDurationMs(latestSync.duration_ms)}</span>
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Sync History</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {syncHistory.length === 0 ? (
            <p className="text-sm text-muted-foreground">No sync history is available yet.</p>
          ) : (
            syncHistory.slice(0, 10).map((sync) => (
              <div key={sync.id} className="rounded-lg border px-4 py-3">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="outline">{sync.sync_type}</Badge>
                    <Badge variant="outline" className="capitalize">
                      {sync.status}
                    </Badge>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {formatMaybeDateTime(sync.completed_at ?? sync.started_at)}
                  </span>
                </div>
                <div className="mt-2 flex flex-wrap gap-4 text-sm text-muted-foreground">
                  <span>{formatMaybeCompact(sync.rows_read)} rows read</span>
                  <span>{formatMaybeCompact(sync.rows_written)} rows written</span>
                  <span>{sync.tables_synced} tables</span>
                  <span>{formatMaybeDurationMs(sync.duration_ms)}</span>
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function MetricCard({
  label,
  value,
  icon: Icon,
}: {
  label: string;
  value: string;
  icon: typeof Database;
}) {
  return (
    <Card>
      <CardContent className="flex items-center gap-3 py-5">
        <div className="rounded-full bg-primary/10 p-2">
          <Icon className="h-4 w-4 text-primary" />
        </div>
        <div>
          <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
          <div className="text-lg font-semibold">{value}</div>
        </div>
      </CardContent>
    </Card>
  );
}

function PropertyRow({
  label,
  value,
  className,
}: {
  label: string;
  value: string;
  className?: string;
}) {
  return (
    <div className={className}>
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm">{value}</div>
    </div>
  );
}
