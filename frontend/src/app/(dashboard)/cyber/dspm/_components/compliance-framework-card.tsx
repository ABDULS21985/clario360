'use client';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ShieldCheck, AlertTriangle } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { DSPMPolicyViolation } from '@/types/cyber';

interface ComplianceFrameworkCardProps {
  framework: string;
  violations: DSPMPolicyViolation[];
  totalPolicies: number;
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-700',
  high: 'bg-orange-100 text-orange-700',
  medium: 'bg-amber-100 text-amber-800',
  low: 'bg-blue-100 text-blue-700',
  info: 'bg-gray-100 text-gray-700',
};

function countBySeverity(violations: DSPMPolicyViolation[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const v of violations) {
    const sev = v.severity ?? 'info';
    counts[sev] = (counts[sev] ?? 0) + 1;
  }
  return counts;
}

export function ComplianceFrameworkCard({
  framework,
  violations,
  totalPolicies,
}: ComplianceFrameworkCardProps) {
  const complianceScore =
    totalPolicies === 0
      ? 100
      : Math.round(((totalPolicies - violations.length) / totalPolicies) * 100);

  const severityCounts = countBySeverity(violations);
  const topViolations = violations.slice(0, 5);
  const hasMore = violations.length > 5;

  const borderColor =
    complianceScore >= 90
      ? 'border-l-green-500'
      : complianceScore >= 70
        ? 'border-l-amber-500'
        : 'border-l-red-500';

  const scoreColor =
    complianceScore >= 90
      ? 'text-green-600'
      : complianceScore >= 70
        ? 'text-amber-600'
        : 'text-red-600';

  const progressColor =
    complianceScore >= 90
      ? 'bg-green-500'
      : complianceScore >= 70
        ? 'bg-amber-500'
        : 'bg-red-500';

  return (
    <Card className={cn('border-l-4', borderColor)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{framework.toUpperCase()}</CardTitle>
          {violations.length === 0 ? (
            <ShieldCheck className="h-5 w-5 text-green-500" />
          ) : (
            <AlertTriangle className={`h-5 w-5 ${complianceScore >= 70 ? 'text-amber-500' : 'text-red-500'}`} />
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Compliance Score */}
        <div>
          <div className="mb-1 flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Compliance Score</span>
            <span className={cn('text-lg font-bold tabular-nums', scoreColor)}>
              {totalPolicies === 0 ? 'No policies' : `${complianceScore}%`}
            </span>
          </div>
          {totalPolicies > 0 && (
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className={cn('h-full rounded-full transition-all', progressColor)}
                style={{ width: `${complianceScore}%` }}
              />
            </div>
          )}
        </div>

        {/* Violation count with severity breakdown */}
        {violations.length > 0 && (
          <div>
            <p className="mb-2 text-sm font-medium">
              {violations.length} violation{violations.length !== 1 ? 's' : ''}
            </p>
            <div className="flex flex-wrap gap-2">
              {['critical', 'high', 'medium', 'low'].map((sev) => {
                const count = severityCounts[sev];
                if (!count) return null;
                return (
                  <span
                    key={sev}
                    className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[sev]}`}
                  >
                    {sev}: {count}
                  </span>
                );
              })}
            </div>
          </div>
        )}

        {/* Top violations */}
        {topViolations.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">Top Violations</p>
            {topViolations.map((v, idx) => (
              <div
                key={`${v.asset_id}-${v.policy_id}-${idx}`}
                className="flex items-start gap-2 rounded-md border p-2 text-xs"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium truncate">{v.asset_name}</span>
                    <Badge variant="outline" className="shrink-0 text-xs px-1.5 py-0 capitalize">
                      {v.category.replace(/_/g, ' ')}
                    </Badge>
                  </div>
                  <p className="mt-0.5 truncate text-muted-foreground">{v.description}</p>
                </div>
                <span className={`shrink-0 inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[v.severity] ?? 'bg-gray-100 text-gray-700'}`}>
                  {v.severity}
                </span>
              </div>
            ))}
          </div>
        )}

        {/* View All */}
        {hasMore && (
          <Button type="button" variant="outline" size="sm" className="w-full text-xs">
            View All {violations.length} Violations
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
