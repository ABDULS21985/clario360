'use client';

import { ArrowDown, ArrowUp, Shield, Scale } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { DetailPanel } from '@/components/shared/detail-panel';
import { SeverityIndicator, type Severity } from '@/components/shared/severity-indicator';
import { titleCase } from '@/lib/format';
import type { VCISOControlDependency } from '@/types/cyber';

interface DependencyDetailPanelProps {
  dependency: VCISOControlDependency;
  open: boolean;
  onClose: () => void;
}

const impactToSeverity: Record<string, Severity> = {
  critical: 'critical',
  high: 'high',
  medium: 'medium',
  low: 'low',
};

export function DependencyDetailPanel({
  dependency,
  open,
  onClose,
}: DependencyDetailPanelProps) {
  return (
    <DetailPanel
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={dependency.control_name}
      description={`Framework: ${dependency.framework}`}
      width="xl"
    >
      <div className="space-y-6">
        {/* Failure Impact */}
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Failure Impact
            </p>
            <SeverityIndicator
              severity={impactToSeverity[dependency.failure_impact] ?? 'medium'}
              size="lg"
            />
          </div>
          <div className="space-y-1 text-right">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Framework
            </p>
            <Badge variant="outline">{dependency.framework}</Badge>
          </div>
        </div>

        <Separator />

        {/* Depends On */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <ArrowDown className="h-4 w-4 text-blue-600" />
            <h3 className="text-sm font-semibold text-foreground">
              Depends On ({dependency.depends_on.length})
            </h3>
          </div>
          {dependency.depends_on.length > 0 ? (
            <div className="space-y-2">
              {dependency.depends_on.map((dep) => (
                <div
                  key={dep}
                  className="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2"
                >
                  <ArrowDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                  <span className="text-sm">{dep}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              This control has no upstream dependencies.
            </p>
          )}
        </div>

        <Separator />

        {/* Depended By */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <ArrowUp className="h-4 w-4 text-orange-600" />
            <h3 className="text-sm font-semibold text-foreground">
              Depended By ({dependency.depended_by.length})
            </h3>
          </div>
          {dependency.depended_by.length > 0 ? (
            <div className="space-y-2">
              {dependency.depended_by.map((dep) => (
                <div
                  key={dep}
                  className="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2"
                >
                  <ArrowUp className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                  <span className="text-sm">{dep}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No other controls depend on this control.
            </p>
          )}
        </div>

        <Separator />

        {/* Risk Domains */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <Shield className="h-4 w-4 text-red-600" />
            <h3 className="text-sm font-semibold text-foreground">
              Risk Domains ({dependency.risk_domains.length})
            </h3>
          </div>
          {dependency.risk_domains.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {dependency.risk_domains.map((domain) => (
                <Badge key={domain} variant="secondary">
                  {titleCase(domain)}
                </Badge>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No risk domains assigned.
            </p>
          )}
        </div>

        <Separator />

        {/* Compliance Domains */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <Scale className="h-4 w-4 text-blue-600" />
            <h3 className="text-sm font-semibold text-foreground">
              Compliance Domains ({dependency.compliance_domains.length})
            </h3>
          </div>
          {dependency.compliance_domains.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {dependency.compliance_domains.map((domain) => (
                <Badge key={domain} variant="outline">
                  {titleCase(domain)}
                </Badge>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No compliance domains assigned.
            </p>
          )}
        </div>
      </div>
    </DetailPanel>
  );
}
