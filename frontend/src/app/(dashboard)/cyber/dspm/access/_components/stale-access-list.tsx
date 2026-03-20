'use client';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import type { StaleAccessResult } from '@/types/cyber';

function formatIdentityType(type: string): string {
  return type
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function riskColor(score: number): string {
  if (score >= 75) return 'text-red-600';
  if (score >= 50) return 'text-orange-600';
  if (score >= 25) return 'text-amber-600';
  return 'text-green-600';
}

export function StaleAccessList() {
  const {
    data: envelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<{ data: StaleAccessResult[] }>(
    API_ENDPOINTS.CYBER_DSPM_ACCESS_STALE,
    { pollInterval: 60000 },
  );

  const results = envelope?.data ?? [];

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Stale Permissions</CardTitle>
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
          <CardTitle className="text-sm">Stale Permissions</CardTitle>
        </CardHeader>
        <CardContent>
          <ErrorState
            message="Failed to load stale access data"
            onRetry={() => void refetch()}
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
        <CardTitle className="text-sm">Stale Permissions</CardTitle>
        {results.length > 0 && (
          <Badge variant="warning">{results.length}</Badge>
        )}
      </CardHeader>
      <CardContent>
        {results.length === 0 ? (
          <div className="rounded-lg border bg-muted/20 p-4 text-center text-sm text-muted-foreground">
            No stale permissions detected.
          </div>
        ) : (
          <div className="space-y-3">
            {results.map((result) => (
              <div
                key={result.identity_id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{result.identity_name}</p>
                  <div className="mt-1 flex items-center gap-2">
                    <Badge variant="outline">
                      {formatIdentityType(result.identity_type)}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {result.stale_count} stale permission{result.stale_count !== 1 ? 's' : ''}
                    </span>
                  </div>
                </div>
                <div className="text-right">
                  <p className={`text-sm font-semibold tabular-nums ${riskColor(result.total_sensitivity_risk)}`}>
                    {Math.round(result.total_sensitivity_risk)}
                  </p>
                  <p className="text-xs text-muted-foreground">sensitivity risk</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
