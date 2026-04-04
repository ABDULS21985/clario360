'use client';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { CheckCircle2, AlertTriangle, FileSearch } from 'lucide-react';
import type { DSPMPolicyImpact } from '@/types/cyber';

interface PolicyImpactPreviewProps {
  impact: DSPMPolicyImpact | null;
  isLoading: boolean;
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-700',
  high: 'bg-orange-100 text-orange-700',
  medium: 'bg-amber-100 text-amber-800',
  low: 'bg-blue-100 text-blue-700',
  info: 'bg-gray-100 text-gray-700',
};

function SkeletonRow() {
  return (
    <tr className="border-b">
      <td className="px-4 py-3"><div className="h-4 w-24 animate-pulse rounded bg-muted" /></td>
      <td className="px-4 py-3"><div className="h-4 w-16 animate-pulse rounded bg-muted" /></td>
      <td className="px-4 py-3"><div className="h-4 w-20 animate-pulse rounded bg-muted" /></td>
      <td className="px-4 py-3"><div className="h-4 w-12 animate-pulse rounded bg-muted" /></td>
      <td className="px-4 py-3"><div className="h-4 w-32 animate-pulse rounded bg-muted" /></td>
      <td className="px-4 py-3"><div className="h-4 w-16 animate-pulse rounded bg-muted" /></td>
    </tr>
  );
}

export function PolicyImpactPreview({ impact, isLoading }: PolicyImpactPreviewProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Policy Impact Preview</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="h-5 w-48 animate-pulse rounded bg-muted" />
            <div className="overflow-x-auto rounded-lg border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Asset</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Type</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Classification</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Severity</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Description</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Enforcement</th>
                  </tr>
                </thead>
                <tbody>
                  <SkeletonRow />
                  <SkeletonRow />
                  <SkeletonRow />
                </tbody>
              </table>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!impact) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Policy Impact Preview</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <FileSearch className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">Run a dry-run to preview policy impact</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  const severityBreakdown: Record<string, number> = {};
  for (const v of impact.affected_assets) {
    const sev = v.severity ?? 'info';
    severityBreakdown[sev] = (severityBreakdown[sev] ?? 0) + 1;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Policy Impact Preview</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm">
          <span className="font-semibold">{impact.total_assets_evaluated}</span>{' '}
          assets evaluated,{' '}
          <span className="font-semibold">{impact.violations_found}</span>{' '}
          violations found
        </p>

        {impact.violations_found === 0 ? (
          <div className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-900 dark:bg-green-950/20">
            <CheckCircle2 className="h-5 w-5 text-green-600" />
            <p className="text-sm font-medium text-green-700 dark:text-green-400">No violations detected</p>
          </div>
        ) : (
          <>
            {/* Severity breakdown */}
            <div className="flex flex-wrap gap-2">
              {Object.entries(severityBreakdown)
                .sort(([a], [b]) => {
                  const order = ['critical', 'high', 'medium', 'low', 'info'];
                  return order.indexOf(a) - order.indexOf(b);
                })
                .map(([sev, count]) => (
                  <div key={sev} className="flex items-center gap-1.5">
                    <AlertTriangle className={`h-3.5 w-3.5 ${sev === 'critical' ? 'text-red-500' : sev === 'high' ? 'text-orange-500' : sev === 'medium' ? 'text-amber-500' : sev === 'low' ? 'text-blue-500' : 'text-gray-500'}`} />
                    <span className="text-sm capitalize">{sev}:</span>
                    <span className="text-sm font-semibold">{count}</span>
                  </div>
                ))}
            </div>

            {/* Affected assets table */}
            <div className="overflow-x-auto rounded-lg border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Asset</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Type</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Classification</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Severity</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Description</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Enforcement</th>
                  </tr>
                </thead>
                <tbody>
                  {impact.affected_assets.map((violation) => (
                    <tr key={`${violation.policy_id}-${violation.asset_id}`} className="border-b">
                      <td className="px-4 py-3 font-medium">{violation.asset_name}</td>
                      <td className="px-4 py-3 capitalize text-muted-foreground">
                        {violation.asset_type.replace(/_/g, ' ')}
                      </td>
                      <td className="px-4 py-3 capitalize">{violation.classification}</td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[violation.severity] ?? 'bg-gray-100 text-gray-700'}`}>
                          {violation.severity}
                        </span>
                      </td>
                      <td className="max-w-xs truncate px-4 py-3 text-muted-foreground">
                        {violation.description}
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant="outline" className="text-xs capitalize">
                          {violation.enforcement.replace(/_/g, ' ')}
                        </Badge>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
