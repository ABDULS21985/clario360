'use client';

import { useParams } from 'next/navigation';
import { AlertTriangle } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { formatDateTime, cn } from '@/lib/utils';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { Alert } from '@/types/models';

export default function AlertDetailPage() {
  const { id } = useParams<{ id: string }>();

  const { data: alert, isLoading, isError, refetch } = useQuery({
    queryKey: ['cyber', 'alert', id],
    queryFn: () => apiGet<Alert>(`/api/v1/cyber/alerts/${id}`),
    enabled: !!id,
  });

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {isLoading ? (
          <LoadingSkeleton variant="card" count={2} />
        ) : isError ? (
          <ErrorState message="Failed to load alert details" onRetry={() => refetch()} />
        ) : alert ? (
          <>
            <PageHeader
              title={alert.title}
              description={`Source: ${alert.source}`}
            />
            <div className="grid gap-4 md:grid-cols-2">
              <Card>
                <CardHeader><CardTitle className="text-sm">Details</CardTitle></CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Severity</span>
                    <span className={cn(
                      'rounded-full px-2 py-0.5 text-xs font-semibold',
                      alert.severity === 'critical' && 'bg-red-100 text-red-800',
                      alert.severity === 'high' && 'bg-orange-100 text-orange-800',
                      alert.severity === 'medium' && 'bg-yellow-100 text-yellow-800',
                    )}>{alert.severity}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Status</span>
                    <span>{alert.status.replace('_', ' ')}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Created</span>
                    <span>{formatDateTime(alert.created_at)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Updated</span>
                    <span>{formatDateTime(alert.updated_at)}</span>
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader><CardTitle className="text-sm">Description</CardTitle></CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground">{alert.description}</p>
                </CardContent>
              </Card>
            </div>
          </>
        ) : null}
      </div>
    </PermissionRedirect>
  );
}
