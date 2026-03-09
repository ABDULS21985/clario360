'use client';

import Link from 'next/link';
import { Ban, CirclePause, FlaskConical } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { RelativeTime } from '@/components/shared/relative-time';
import type { AILifecycleHistoryEntry, AIModelVersion } from '@/types/ai-governance';

interface VersionTimelineProps {
  versions: AIModelVersion[];
  history: AILifecycleHistoryEntry[];
  busyVersionId?: string | null;
  onPromote: (version: AIModelVersion) => void;
  onStartShadow: (version: AIModelVersion) => void;
  onStopShadow: (version: AIModelVersion) => void;
  onRetire: (version: AIModelVersion) => void;
  onFail: (version: AIModelVersion) => void;
}

function statusVariant(status: string) {
  switch (status) {
    case 'production':
      return 'success';
    case 'shadow':
      return 'warning';
    case 'failed':
      return 'destructive';
    case 'retired':
    case 'rolled_back':
      return 'secondary';
    default:
      return 'outline';
  }
}

export function VersionTimeline({
  versions,
  history,
  busyVersionId,
  onPromote,
  onStartShadow,
  onStopShadow,
  onRetire,
  onFail,
}: VersionTimelineProps) {
  return (
    <div className="grid gap-4 lg:grid-cols-[1.4fr_1fr]">
      <Card className="border-border/70">
        <CardHeader>
          <CardTitle>Versions</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {versions.map((version) => {
            const isBusy = busyVersionId === version.id;
            return (
              <div key={version.id} className="rounded-xl border border-border/70 bg-muted/20 p-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="text-lg font-semibold">v{version.version_number}</span>
                      <Badge variant={statusVariant(version.status)}>{version.status}</Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">{version.description}</p>
                    <div className="font-mono text-xs text-muted-foreground">{version.artifact_hash.slice(0, 12)}…</div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button asChild variant="ghost" size="sm">
                      <Link href={`/admin/ai-governance/${version.model_id}/validate?versionId=${version.id}`}>
                        <FlaskConical className="mr-1.5 h-3.5 w-3.5" />
                        Validate
                      </Link>
                    </Button>
                    {(version.status === 'development' || version.status === 'staging' || version.status === 'shadow') ? (
                      <Button variant="outline" size="sm" disabled={isBusy} onClick={() => onPromote(version)}>
                        {version.status === 'shadow' ? 'Promote to Prod' : 'Promote'}
                      </Button>
                    ) : null}
                    {version.status === 'staging' ? (
                      <Button variant="ghost" size="sm" disabled={isBusy} onClick={() => onStartShadow(version)}>
                        Start Shadow
                      </Button>
                    ) : null}
                    {version.status === 'shadow' ? (
                      <Button variant="ghost" size="sm" disabled={isBusy} onClick={() => onStopShadow(version)}>
                        <CirclePause className="mr-1.5 h-3.5 w-3.5" />
                        Stop Shadow
                      </Button>
                    ) : null}
                    {(version.status === 'development' || version.status === 'staging' || version.status === 'shadow') ? (
                      <Button variant="ghost" size="sm" disabled={isBusy} onClick={() => onFail(version)}>
                        <Ban className="mr-1.5 h-3.5 w-3.5" />
                        Mark Failed
                      </Button>
                    ) : null}
                    {(version.status !== 'retired' && version.status !== 'rolled_back') ? (
                      <Button variant="ghost" size="sm" disabled={isBusy} onClick={() => onRetire(version)}>
                        Retire
                      </Button>
                    ) : null}
                  </div>
                </div>
                <div className="mt-4 grid gap-2 text-sm text-muted-foreground md:grid-cols-3">
                  <div>Predictions: {version.prediction_count.toLocaleString()}</div>
                  <div>Avg latency: {version.avg_latency_ms ? `${Math.round(version.avg_latency_ms)} ms` : 'n/a'}</div>
                  <div>Avg confidence: {version.avg_confidence ? `${Math.round(version.avg_confidence * 100)}%` : 'n/a'}</div>
                </div>
              </div>
            );
          })}
        </CardContent>
      </Card>

      <Card className="border-border/70">
        <CardHeader>
          <CardTitle>Lifecycle Audit</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {history.map((entry) => (
            <div key={`${entry.version_id}-${entry.changed_at}`} className="border-l-2 border-border pl-4">
              <div className="flex items-center gap-2">
                <Badge variant={statusVariant(entry.to_status)}>{entry.to_status}</Badge>
                <span className="text-sm font-medium">v{entry.version_number}</span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {entry.from_status ? `${entry.from_status} -> ${entry.to_status}` : `Entered ${entry.to_status}`}
              </p>
              {entry.reason ? <p className="mt-1 text-sm">{entry.reason}</p> : null}
              {entry.changed_by ? <p className="mt-1 text-xs text-muted-foreground">Changed by {entry.changed_by}</p> : null}
              <RelativeTime date={entry.changed_at} className="mt-1 text-xs text-muted-foreground" />
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
