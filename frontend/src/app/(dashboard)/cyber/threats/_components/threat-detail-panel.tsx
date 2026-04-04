'use client';

import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { threatStatusConfig } from '@/lib/status-configs';
import { Badge } from '@/components/ui/badge';
import { timeAgo, cn } from '@/lib/utils';
import { X, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { Threat } from '@/types/cyber';
import { getIndicatorTypeLabel, getThreatTypeLabel } from '@/lib/cyber-threats';

const INDICATOR_TYPE_COLORS: Record<string, string> = {
  ip: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  domain: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  hash: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
  url: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  email: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
};

interface ThreatDetailPanelProps {
  threat: Threat;
  onClose: () => void;
}

export function ThreatDetailPanel({ threat, onClose }: ThreatDetailPanelProps) {
  return (
    <div className="flex h-full flex-col overflow-hidden rounded-xl border bg-card shadow-lg">
      {/* Header */}
      <div className="flex items-start justify-between border-b p-4">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 mb-1">
            <Shield className="h-4 w-4 text-muted-foreground shrink-0" />
            <SeverityIndicator severity={threat.severity} showLabel />
          </div>
          <h3 className="font-semibold truncate">{threat.name}</h3>
          <p className="text-xs text-muted-foreground mt-0.5">{getThreatTypeLabel(threat.type)}</p>
        </div>
        <Button variant="ghost" size="sm" className="h-7 w-7 p-0 shrink-0" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Status row */}
        <div className="flex items-center gap-3 flex-wrap">
          <StatusBadge status={threat.status} config={threatStatusConfig} />
          <span className="text-xs text-muted-foreground">
            {threat.indicator_count} indicators
          </span>
          <span className="text-xs text-muted-foreground">
            {threat.affected_asset_count} affected assets
          </span>
        </div>

        {/* Description */}
        <p className="text-sm leading-relaxed text-muted-foreground">{threat.description}</p>

        {/* Timeline */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground">First Seen</p>
            <p className="text-sm font-medium mt-0.5">{timeAgo(threat.first_seen_at)}</p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground">Last Seen</p>
            <p className="text-sm font-medium mt-0.5">{timeAgo(threat.last_seen_at)}</p>
          </div>
        </div>

        {/* Tags */}
        {(threat.tags?.length ?? 0) > 0 && (
          <div>
            <p className="text-xs font-semibold mb-2">Tags</p>
            <div className="flex flex-wrap gap-1.5">
              {threat.tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="text-xs">{tag}</Badge>
              ))}
            </div>
          </div>
        )}

        {/* Indicators */}
        {(threat.indicators?.length ?? 0) > 0 && (
          <div>
            <p className="text-xs font-semibold mb-2">
              Threat Indicators ({threat.indicators!.length})
            </p>
            <div className="space-y-1.5 max-h-64 overflow-y-auto">
              {threat.indicators!.map((ind) => (
                <div key={ind.id} className="flex items-center gap-2 rounded-lg border p-2">
                  <span className={cn('rounded px-1.5 py-0.5 text-xs font-medium', INDICATOR_TYPE_COLORS[ind.type] ?? 'bg-muted text-muted-foreground')}>
                    {getIndicatorTypeLabel(ind.type)}
                  </span>
                  <span className="font-mono text-xs flex-1 truncate">{ind.value}</span>
                  <SeverityIndicator severity={ind.severity} />
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
