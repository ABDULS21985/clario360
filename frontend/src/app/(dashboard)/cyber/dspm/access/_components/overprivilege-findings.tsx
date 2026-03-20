'use client';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import type { OverprivilegeResult, CyberSeverity } from '@/types/cyber';

function severityBadgeVariant(severity: CyberSeverity) {
  switch (severity) {
    case 'critical':
      return 'destructive' as const;
    case 'high':
      return 'destructive' as const;
    case 'medium':
      return 'warning' as const;
    case 'low':
      return 'secondary' as const;
    default:
      return 'outline' as const;
  }
}

function formatLabel(value: string): string {
  return value
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export function OverprivilegeFindings() {
  const {
    data: envelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<{ data: OverprivilegeResult[] }>(
    API_ENDPOINTS.CYBER_DSPM_ACCESS_OVERPRIVILEGED,
    { pollInterval: 60000 },
  );

  const findings = envelope?.data ?? [];
  const topFindings = findings.slice(0, 10);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Overprivileged Access</CardTitle>
        </CardHeader>
        <CardContent>
          <LoadingSkeleton variant="list-item" />
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Overprivileged Access</CardTitle>
        </CardHeader>
        <CardContent>
          <ErrorState
            message="Failed to load overprivilege findings"
            onRetry={() => void refetch()}
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
        <CardTitle className="text-sm">Overprivileged Access</CardTitle>
        {findings.length > 0 && (
          <Badge variant="destructive">{findings.length}</Badge>
        )}
      </CardHeader>
      <CardContent>
        {topFindings.length === 0 ? (
          <div className="rounded-lg border bg-muted/20 p-4 text-center text-sm text-muted-foreground">
            No overprivileged access findings detected.
          </div>
        ) : (
          <div className="space-y-3">
            {topFindings.map((finding) => (
              <div
                key={`${finding.identity_id}-${finding.data_asset_id}`}
                className="rounded-lg border p-3"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium">{finding.identity_name}</p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {finding.data_asset_name} &middot;{' '}
                      {formatLabel(finding.permission_type)} &middot;{' '}
                      {formatLabel(finding.data_classification)}
                    </p>
                  </div>
                  <Badge variant={severityBadgeVariant(finding.severity)}>
                    {finding.severity}
                  </Badge>
                </div>
                {finding.recommendation && (
                  <p className="mt-2 text-xs text-muted-foreground">
                    {finding.recommendation}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
