'use client';

import { Badge } from '@/components/ui/badge';
import { DetailPanel } from '@/components/shared/detail-panel';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import type { SecurityEvent } from '@/types/cyber';

interface EventDetailPanelProps {
  event: SecurityEvent | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function EventDetailPanel({ event, open, onOpenChange }: EventDetailPanelProps) {
  if (!event) return null;

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title="Event Detail"
      description={`${event.source} — ${event.type}`}
      width="lg"
    >
      <div className="space-y-6">
        {/* Header */}
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
        </div>

        {/* Network Context */}
        {(event.source_ip || event.dest_ip) && (
          <div>
            <h4 className="text-sm font-semibold mb-2">Network Context</h4>
            <div className="flex items-center gap-3 rounded-md border p-3 bg-muted/50">
              <code className="text-xs">
                {event.source_ip ?? '—'}
              </code>
              <span className="text-muted-foreground">→</span>
              <code className="text-xs">
                {event.dest_ip ?? '—'}
                {event.dest_port ? `:${event.dest_port}` : ''}
              </code>
              {event.protocol && (
                <Badge variant="secondary" className="text-xs ml-auto">
                  {event.protocol}
                </Badge>
              )}
            </div>
          </div>
        )}

        {/* Process Info */}
        {(event.process || event.command_line) && (
          <div>
            <h4 className="text-sm font-semibold mb-2">Process Information</h4>
            <div className="space-y-2 rounded-md border p-3 bg-muted/50">
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
                  <pre className="text-xs whitespace-pre-wrap break-all mt-1 p-2 bg-background rounded">
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
            <h4 className="text-sm font-semibold mb-2">File Details</h4>
            <div className="space-y-1 rounded-md border p-3 bg-muted/50">
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
                <p className="text-xs text-muted-foreground">Asset ID</p>
                <code className="text-xs">{event.asset_id}</code>
              </div>
            )}
          </div>
        )}

        {/* Matched Rules */}
        {event.matched_rules?.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold mb-2">
              Matched Rules ({event.matched_rules.length})
            </h4>
            <div className="space-y-1">
              {event.matched_rules.map((ruleId) => (
                <code key={ruleId} className="block text-xs text-muted-foreground">
                  {ruleId}
                </code>
              ))}
            </div>
          </div>
        )}

        {/* Raw Event */}
        <div>
          <h4 className="text-sm font-semibold mb-2">Raw Event</h4>
          <pre className="text-xs p-3 rounded-md bg-muted overflow-auto max-h-80 whitespace-pre-wrap break-all">
            {JSON.stringify(event.raw_event, null, 2)}
          </pre>
        </div>
      </div>
    </DetailPanel>
  );
}
