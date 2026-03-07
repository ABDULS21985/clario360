'use client';
import { Shield } from 'lucide-react';
import type { ThreatLandscape } from '@/types/cyber';

interface MetricRowProps {
  label: string;
  value: string | number;
  highlight?: boolean;
}

function MetricRow({ label, value, highlight = false }: MetricRowProps) {
  return (
    <div className="flex items-center justify-between py-2 border-b last:border-b-0">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span
        className={`text-sm font-semibold tabular-nums ${
          highlight ? 'text-orange-600' : 'text-foreground'
        }`}
      >
        {value}
      </span>
    </div>
  );
}

export function ThreatLandscapeSection({ landscape }: { landscape: ThreatLandscape }) {
  const threatTypes = Object.entries(landscape.threat_by_type);
  const maxCount = threatTypes.reduce((max, [, count]) => Math.max(max, count), 1);

  return (
    <div className="rounded-lg border bg-white p-6 space-y-5">
      <div className="flex items-center gap-2">
        <Shield className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-semibold text-foreground">Threat Landscape</h3>
      </div>

      {/* Metric rows */}
      <div>
        <MetricRow
          label="Active Threats"
          value={landscape.active_threat_count}
          highlight
        />
        <MetricRow label="Top Tactic" value={landscape.top_tactic} />
        <MetricRow label="Top Technique" value={landscape.top_technique} />
        <MetricRow
          label="Recent Indicators"
          value={landscape.recent_indicators}
        />
      </div>

      {/* Threat by type breakdown */}
      {threatTypes.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Threats by Type
          </p>
          {threatTypes.map(([type, count]) => (
            <div key={type} className="flex items-center gap-3">
              <span className="w-32 flex-shrink-0 truncate text-xs text-muted-foreground capitalize">
                {type.replace(/_/g, ' ')}
              </span>
              <div className="flex-1 h-2 overflow-hidden rounded-full bg-muted">
                <div
                  className="h-2 rounded-full bg-orange-500 transition-all duration-500"
                  style={{ width: `${(count / maxCount) * 100}%` }}
                />
              </div>
              <span className="w-8 text-right text-xs font-medium tabular-nums text-foreground">
                {count}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
