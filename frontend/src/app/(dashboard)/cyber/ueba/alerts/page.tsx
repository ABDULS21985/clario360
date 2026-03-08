'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';

import { SignalEvidenceViewer } from '../_components/signal-evidence-viewer';
import type { UebaAlert } from '../_components/types';

export default function UebaAlertsPage() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['cyber-ueba-alerts'],
    queryFn: () => apiGet<{ data: UebaAlert[]; total: number }>(API_ENDPOINTS.CYBER_UEBA_ALERTS),
  });

  const alerts = data?.data ?? [];

  if (isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <div className="space-y-4">
          <PageHeader title="UEBA Alerts" description="Correlated behavioral findings with event-level evidence." />
          {Array.from({ length: 3 }).map((_, index) => <LoadingSkeleton key={index} variant="card" />)}
        </div>
      </PermissionRedirect>
    );
  }

  if (error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState message="Failed to load UEBA alerts." onRetry={() => void refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="UEBA Alerts"
          description="Multi-signal behavioral alerts linked back to their raw triggering events."
        />

        <div className="grid gap-4">
          {alerts.map((alert) => (
            <Card key={alert.id} className="border-border/70">
              <CardHeader className="gap-3">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <CardTitle className="text-base">{alert.title}</CardTitle>
                    <div className="mt-1 text-sm text-muted-foreground">
                      <Link href={`/cyber/ueba/profiles/${encodeURIComponent(alert.entity_id)}`} className="hover:underline">
                        {alert.entity_name ?? alert.entity_id}
                      </Link>
                      {' · '}
                      {alert.alert_type.replaceAll('_', ' ')}
                    </div>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={alert.severity === 'critical' ? 'destructive' : alert.severity === 'high' ? 'warning' : 'outline'}>
                      {alert.severity}
                    </Badge>
                    <Badge variant="secondary">
                      {(alert.confidence * 100).toFixed(0)}% confidence
                    </Badge>
                    <Badge variant="outline">{alert.status}</Badge>
                  </div>
                </div>
                <div className="text-sm text-muted-foreground">{alert.description}</div>
              </CardHeader>
              <CardContent className="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
                <div className="rounded-lg border bg-muted/20 p-3">
                  <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Risk Impact</div>
                  <div className="text-sm">
                    {alert.risk_score_before.toFixed(0)} → {alert.risk_score_after.toFixed(0)} ({alert.risk_score_delta >= 0 ? '+' : ''}{alert.risk_score_delta.toFixed(0)})
                  </div>
                  <div className="mt-3 text-xs text-muted-foreground">
                    Triggered by {alert.correlated_signal_count} signals across {alert.triggering_event_ids.length} events.
                  </div>
                  <pre className="mt-3 overflow-auto rounded-md bg-background p-3 text-xs">
                    {JSON.stringify(alert.baseline_comparison, null, 2)}
                  </pre>
                </div>
                <SignalEvidenceViewer alert={alert} />
              </CardContent>
            </Card>
          ))}
          {alerts.length === 0 && (
            <Card>
              <CardContent className="p-8 text-center text-muted-foreground">
                No UEBA alerts yet.
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </PermissionRedirect>
  );
}
