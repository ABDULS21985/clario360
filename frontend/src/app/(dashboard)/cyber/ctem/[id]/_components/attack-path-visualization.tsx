'use client';

import { ArrowRight, AlertTriangle } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import type { CTEMFinding } from '@/types/cyber';

export function AttackPathVisualization({ findings }: { findings: CTEMFinding[] }) {
  const withPaths = findings.filter((f) => f.attack_path && f.attack_path.length > 0);

  if (withPaths.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-10 text-center">
        <AlertTriangle className="h-8 w-8 text-muted-foreground/40" aria-hidden />
        <p className="text-sm text-muted-foreground">No attack path data available</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {withPaths.map((finding) => (
        <div key={finding.id} className="rounded-xl border bg-card p-4">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <SeverityIndicator severity={finding.severity} showLabel size="sm" />
            <span className="text-sm font-semibold">{finding.title}</span>
            {finding.asset_name && (
              <span className="text-xs text-muted-foreground">— {finding.asset_name}</span>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            {finding.attack_path!.map((node, idx) => (
              <div key={idx} className="flex items-center gap-1.5">
                <Badge
                  variant="outline"
                  className="whitespace-nowrap rounded-full border-primary/30 bg-primary/5 px-2.5 py-1 text-xs font-medium text-primary"
                >
                  {node}
                </Badge>
                {idx < finding.attack_path!.length - 1 && (
                  <ArrowRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden />
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
