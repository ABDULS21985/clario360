'use client';

import { AlertTriangle } from 'lucide-react';
import { CTI_SEVERITY_COLORS, type CTISectorThreatSummary } from '@/types/cti';
import { cn } from '@/lib/utils';

interface SectorThreatChartProps {
  sectors: CTISectorThreatSummary[];
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onSectorClick?: (sectorId: string) => void;
  selectedSectorId?: string;
  maxBarWidth?: number;
}

const SEGMENTS = [
  { key: 'severity_critical_count', label: 'Critical', color: CTI_SEVERITY_COLORS.critical },
  { key: 'severity_high_count', label: 'High', color: CTI_SEVERITY_COLORS.high },
  { key: 'severity_medium_count', label: 'Medium', color: CTI_SEVERITY_COLORS.medium },
  { key: 'severity_low_count', label: 'Low', color: CTI_SEVERITY_COLORS.low },
] as const;

export function SectorThreatChart({
  sectors,
  loading = false,
  error,
  onRetry,
  onSectorClick,
  selectedSectorId,
  maxBarWidth = 100,
}: SectorThreatChartProps) {
  if (loading) {
    return (
      <div className="rounded-[24px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, index) => (
            <div key={index} className="h-12 animate-pulse rounded-xl bg-muted/40" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-[24px] border border-dashed border-amber-500/30 bg-amber-500/5 p-5 text-sm text-muted-foreground">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-amber-500" />
          {error}
        </div>
        {onRetry && (
          <button type="button" className="mt-3 text-sm font-medium text-primary hover:underline" onClick={onRetry}>
            Retry
          </button>
        )}
      </div>
    );
  }

  const sorted = [...sectors].sort((left, right) => right.total_count - left.total_count);
  const maxTotal = sorted[0]?.total_count ?? 1;

  return (
    <div className="rounded-[24px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-950">Threat Pressure by Sector</h3>
          <p className="text-sm text-muted-foreground">
            Click a sector row to expand its deep dive and pivot into filtered investigations.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 text-[11px] text-muted-foreground">
          {SEGMENTS.map((segment) => (
            <span key={segment.key} className="inline-flex items-center gap-1">
              <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: segment.color }} />
              {segment.label}
            </span>
          ))}
        </div>
      </div>
      <div className="space-y-3">
        {sorted.map((sector) => {
          const width = `${Math.max((sector.total_count / maxTotal) * maxBarWidth, 12)}%`;

          return (
            <button
              key={sector.id}
              type="button"
              onClick={() => onSectorClick?.(sector.sector_id)}
              className={cn(
                'w-full rounded-2xl border p-4 text-left transition hover:bg-muted/30',
                selectedSectorId === sector.sector_id && 'border-primary/40 bg-primary/5',
              )}
            >
              <div className="mb-2 flex items-center justify-between gap-3">
                <div>
                  <p className="font-medium text-foreground">{sector.sector_label}</p>
                  <p className="text-xs text-muted-foreground">{sector.total_count.toLocaleString()} events</p>
                </div>
                <span className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                  {sector.sector_code || sector.sector_id.slice(0, 8)}
                </span>
              </div>
              <div className="h-4 overflow-hidden rounded-full bg-muted/40" style={{ width }}>
                <div className="flex h-full">
                  {SEGMENTS.map((segment) => {
                    const count = sector[segment.key];
                    const segmentWidth = sector.total_count > 0 ? `${(count / sector.total_count) * 100}%` : '0%';

                    return (
                      <div
                        key={segment.key}
                        className="h-full"
                        style={{ width: segmentWidth, backgroundColor: segment.color }}
                        title={`${segment.label}: ${count.toLocaleString()}`}
                      />
                    );
                  })}
                </div>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
