'use client';

import { User, Bot } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { timeAgo } from '@/lib/utils';
import type { RemediationAuditEntry } from '@/types/cyber';

function formatAction(action: string): string {
  return action
    .split(/[_\s]+/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join(' ');
}

function hasDetails(details: Record<string, unknown> | undefined): details is Record<string, unknown> {
  return details != null && Object.keys(details).length > 0;
}

function renderDetailValue(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  return JSON.stringify(value);
}

export function AuditTrailTimeline({ entries }: { entries: RemediationAuditEntry[] }) {
  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-10 text-center">
        <User className="h-8 w-8 text-muted-foreground/40" aria-hidden />
        <p className="text-sm text-muted-foreground">No audit trail available</p>
      </div>
    );
  }

  return (
    <div className="space-y-0">
      {entries.map((entry, idx) => {
        const isLast = idx === entries.length - 1;
        const isSystem = !entry.actor_name;

        return (
          <div key={entry.id} className="relative flex gap-3 pb-6 last:pb-0">
            {/* Vertical line */}
            {!isLast && (
              <div
                className="absolute left-4 top-8 h-full w-px bg-border"
                aria-hidden
              />
            )}

            {/* Avatar / icon */}
            <div className="relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border bg-card shadow-sm">
              {isSystem
                ? <Bot className="h-4 w-4 text-muted-foreground" aria-hidden />
                : <User className="h-4 w-4 text-muted-foreground" aria-hidden />
              }
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0 pt-0.5">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium">
                  {entry.actor_name ?? 'System'}
                </span>
                <Badge variant="outline" className="text-xs">
                  {formatAction(entry.action)}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  {timeAgo(entry.created_at)}
                </span>
              </div>

              {hasDetails(entry.details) && (
                <div className="mt-2 rounded-lg border bg-muted/30 px-3 py-2 space-y-1">
                  {Object.entries(entry.details).map(([key, value]) => (
                    <div key={key} className="flex gap-2 text-xs">
                      <span className="shrink-0 font-medium capitalize text-muted-foreground">
                        {key.replace(/_/g, ' ')}:
                      </span>
                      <span className="break-all">{renderDetailValue(value)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
