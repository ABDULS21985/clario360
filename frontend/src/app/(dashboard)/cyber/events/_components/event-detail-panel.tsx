'use client';

import Link from 'next/link';
import { Copy } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DetailPanel } from '@/components/shared/detail-panel';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { showSuccess } from '@/lib/toast';
import type { SecurityEvent } from '@/types/cyber';

interface EventDetailPanelProps {
  event: SecurityEvent | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function EventDetailPanel({ event, open, onOpenChange }: EventDetailPanelProps) {
  if (!event) return null;

  const copyRaw = () => {
    navigator.clipboard.writeText(JSON.stringify(event.raw_event, null, 2));
    showSuccess('Raw JSON copied');
  };

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title="Event Detail"
      description={`${event.source} — ${event.type}`}
      width="lg"
    >
      <div className="space-y-6">
        {/* Header grid */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <p className="text-xs text-muted-foreground">Timestamp</p>
            <p className="text-sm font-medium tabular-nums">
              {new Date(event.timestamp).toLocaleString()}
            </p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Severity</p>
            <SeverityIndicator severity={event.severity} showLabel />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Source</p>
            <p className="text-sm font-medium">{event.source}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Event Type</p>
            <Badge variant="outline" className="capitalize">
              {event.type.replace(/_/g, ' ')}
            </Badge>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Event ID</p>
            <code className="text-xs break-all">{event.id}</code>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Processed At</p>
            <p className="text-xs tabular-nums text-muted-foreground">
              {new Date(event.processed_at).toLocaleString()}
            </p>
          </div>
        </div>

        {/* Network Context */}
        {(event.source_ip || event.dest_ip) && (
          <div>
            <h4 className="mb-2 text-sm font-semibold">Network Context</h4>
            <div className="flex items-center gap-3 rounded-md border bg-muted/50 p-3">
              <code className="text-xs">{event.source_ip ?? '—'}</code>
              <span className="text-muted-foreground">→</span>
              <code className="text-xs">
                {event.dest_ip ?? '—'}
                {event.dest_port ? `:${event.dest_port}` : ''}
              </code>
              {event.protocol && (
                <Badge variant="secondary" className="ml-auto text-xs">
                  {event.protocol}
                </Badge>
              )}
            </div>
          </div>
        )}

        {/* Process Info */}
        {(event.process || event.command_line) && (
          <div>
            <h4 className="mb-2 text-sm font-semibold">Process Information</h4>
            <div className="space-y-2 rounded-md border bg-muted/50 p-3">
              {event.process && (
                <div>
                  <span className="text-xs text-muted-foreground">Process: </span>
                  <code className="text-xs">{event.process}</code>
                </div>
              )}
              {event.parent_process && (
                <div>
                  <span className="text-xs text-muted-foreground">Parent: </span>
                  <code className="text-xs">{event.parent_process}</code>
                </div>
              )}
              {event.command_line && (
                <div>
                  <span className="text-xs text-muted-foreground">Command: </span>
                  <pre className="mt-1 max-h-40 overflow-auto rounded bg-background p-2 text-xs whitespace-pre-wrap break-all">
                    {event.command_line}
                  </pre>
                </div>
              )}
            </div>
          </div>
        )}

        {/* File Info */}
        {(event.file_path || event.file_hash) && (
          <div>
            <h4 className="mb-2 text-sm font-semibold">File Details</h4>
            <div className="space-y-1 rounded-md border bg-muted/50 p-3">
              {event.file_path && (
                <div>
                  <span className="text-xs text-muted-foreground">Path: </span>
                  <code className="text-xs">{event.file_path}</code>
                </div>
              )}
              {event.file_hash && (
                <div>
                  <span className="text-xs text-muted-foreground">Hash: </span>
                  <code className="text-xs break-all">{event.file_hash}</code>
                </div>
              )}
            </div>
          </div>
        )}

        {/* User / Asset */}
        {(event.username || event.asset_id) && (
          <div className="grid grid-cols-2 gap-4">
            {event.username && (
              <div>
                <p className="text-xs text-muted-foreground">Username</p>
                <p className="text-sm">{event.username}</p>
              </div>
            )}
            {event.asset_id && (
              <div>
                <p className="text-xs text-muted-foreground">Asset</p>
                <Link
                  href={`/cyber/assets/${event.asset_id}`}
                  className="text-xs text-primary underline-offset-2 hover:underline"
                >
                  {event.asset_id.slice(0, 8)}…
                </Link>
              </div>
            )}
          </div>
        )}

        {/* Matched Rules */}
        {event.matched_rules?.length > 0 && (
          <div>
            <h4 className="mb-2 text-sm font-semibold">
              Matched Rules ({event.matched_rules.length})
            </h4>
            <div className="space-y-1">
              {event.matched_rules.map((ruleId) => (
                <Link
                  key={ruleId}
                  href={`/cyber/rules/${ruleId}`}
                  className="block text-xs text-primary underline-offset-2 hover:underline"
                >
                  {ruleId}
                </Link>
              ))}
            </div>
          </div>
        )}

        {/* Raw Event */}
        <div>
          <div className="mb-2 flex items-center justify-between">
            <h4 className="text-sm font-semibold">Raw Event</h4>
            <Button variant="ghost" size="sm" className="h-7 px-2" onClick={copyRaw}>
              <Copy className="mr-1 h-3 w-3" />
              Copy
            </Button>
          </div>
          <pre className="max-h-80 overflow-auto rounded-md bg-muted p-3 text-xs whitespace-pre-wrap break-all">
            {JSON.stringify(event.raw_event, null, 2)}
          </pre>
        </div>
      </div>
    </DetailPanel>
  );
}
