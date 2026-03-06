'use client';

import { Bot, Copy } from 'lucide-react';
import { toast } from 'sonner';
import { DetailPanel } from '@/components/shared/detail-panel';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { formatDateTime, copyToClipboard } from '@/lib/utils';
import type { AuditLog } from '@/types/models';
import { formatAuditAction } from './audit-columns';
import { JsonDiffViewer } from './json-diff-viewer';

interface AuditDetailPanelProps {
  log: AuditLog;
  open: boolean;
  onClose: () => void;
}

function CopyButton({ value, label }: { value: string; label: string }) {
  return (
    <button
      onClick={async () => {
        await copyToClipboard(value);
        toast.success(`${label} copied.`);
      }}
      className="ml-1 inline-flex items-center text-muted-foreground hover:text-foreground"
      title={`Copy ${label}`}
    >
      <Copy className="h-3 w-3" />
    </button>
  );
}

export function AuditDetailPanel({ log, open, onClose }: AuditDetailPanelProps) {
  const isSystem = !log.user_id || log.user_email === 'system';

  return (
    <DetailPanel
      open={open}
      onOpenChange={(o) => { if (!o) onClose(); }}
      title={formatAuditAction(log.action)}
      description={formatDateTime(log.created_at)}
      width="lg"
    >
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center gap-3 flex-wrap">
          {log.severity && (
            <SeverityIndicator
              severity={
                log.severity === 'warning'
                  ? 'medium'
                  : log.severity === 'high'
                  ? 'high'
                  : log.severity === 'critical'
                  ? 'critical'
                  : 'info'
              }
            />
          )}
          <span className="text-xs font-mono text-muted-foreground">
            {log.id.slice(0, 16)}...
            <CopyButton value={log.id} label="Entry ID" />
          </span>
        </div>

        <Separator />

        {/* Context */}
        <div className="space-y-3">
          <h4 className="text-sm font-medium">Context</h4>
          <dl className="space-y-2 text-sm">
            <div className="flex gap-2">
              <dt className="text-muted-foreground w-28 shrink-0">User</dt>
              <dd>
                {isSystem ? (
                  <span className="inline-flex items-center gap-1 text-muted-foreground">
                    <Bot className="h-3.5 w-3.5" /> System
                  </span>
                ) : (
                  log.user_email
                )}
              </dd>
            </div>
            {log.service && (
              <div className="flex gap-2">
                <dt className="text-muted-foreground w-28 shrink-0">Service</dt>
                <dd>
                  <Badge variant="outline" className="text-xs">{log.service}</Badge>
                </dd>
              </div>
            )}
            <div className="flex gap-2">
              <dt className="text-muted-foreground w-28 shrink-0">Resource</dt>
              <dd>
                {log.resource_type}
                {log.resource_id && (
                  <span className="ml-1 font-mono text-xs text-muted-foreground">
                    {log.resource_id}
                    <CopyButton value={log.resource_id} label="Resource ID" />
                  </span>
                )}
              </dd>
            </div>
            <div className="flex gap-2">
              <dt className="text-muted-foreground w-28 shrink-0">IP Address</dt>
              <dd className="font-mono text-xs">{log.ip_address || '—'}</dd>
            </div>
            <div className="flex gap-2">
              <dt className="text-muted-foreground w-28 shrink-0">User Agent</dt>
              <dd className="text-xs text-muted-foreground truncate max-w-xs" title={log.user_agent}>
                {log.user_agent || '—'}
              </dd>
            </div>
            {log.correlation_id && (
              <div className="flex gap-2">
                <dt className="text-muted-foreground w-28 shrink-0">Correlation ID</dt>
                <dd className="font-mono text-xs">
                  {log.correlation_id}
                  <CopyButton value={log.correlation_id} label="Correlation ID" />
                </dd>
              </div>
            )}
          </dl>
        </div>

        {/* Changes */}
        {(log.old_value !== undefined || log.new_value !== undefined) && (
          <>
            <Separator />
            <div className="space-y-3">
              <h4 className="text-sm font-medium">Changes</h4>
              <JsonDiffViewer oldValue={log.old_value} newValue={log.new_value} />
            </div>
          </>
        )}

        {/* Hash Chain */}
        {log.entry_hash && (
          <>
            <Separator />
            <div className="space-y-3">
              <h4 className="text-sm font-medium">Hash Chain</h4>
              <dl className="space-y-2 text-xs">
                <div>
                  <dt className="text-muted-foreground mb-0.5">Entry Hash</dt>
                  <dd className="font-mono break-all text-foreground">
                    {log.entry_hash}
                    <CopyButton value={log.entry_hash} label="Entry Hash" />
                  </dd>
                </div>
                {log.prev_hash && (
                  <div>
                    <dt className="text-muted-foreground mb-0.5">Previous Hash</dt>
                    <dd className="font-mono break-all text-muted-foreground">
                      {log.prev_hash}
                    </dd>
                  </div>
                )}
              </dl>
            </div>
          </>
        )}

        {/* Metadata */}
        {Object.keys(log.metadata ?? {}).length > 0 && (
          <>
            <Separator />
            <div className="space-y-3">
              <h4 className="text-sm font-medium">Metadata</h4>
              <dl className="space-y-1.5 text-xs">
                {Object.entries(log.metadata).map(([key, val]) => (
                  <div key={key} className="flex gap-2">
                    <dt className="text-muted-foreground w-28 shrink-0 font-mono">{key}</dt>
                    <dd className="font-mono text-foreground break-all">
                      {typeof val === 'object' ? JSON.stringify(val) : String(val)}
                    </dd>
                  </div>
                ))}
              </dl>
            </div>
          </>
        )}
      </div>
    </DetailPanel>
  );
}
