'use client';

import Link from 'next/link';
import { useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginationMeta } from '@/types/api';

import { SignalEvidenceViewer } from '../_components/signal-evidence-viewer';
import { AlertActions } from '../_components/alert-actions';
import { BulkAlertActions } from '../_components/bulk-alert-actions';
import type { UebaAlert } from '../_components/types';
import { UEBA_ALERT_STATUSES } from '../_components/types';

const PER_PAGE = 15;

export default function UebaAlertsPage() {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const params: Record<string, unknown> = { page, per_page: PER_PAGE };
  if (statusFilter && statusFilter !== 'all') {
    params.status = statusFilter;
  }

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['cyber-ueba-alerts', page, statusFilter],
    queryFn: () =>
      apiGet<{ data: UebaAlert[]; meta: PaginationMeta }>(API_ENDPOINTS.CYBER_UEBA_ALERTS, params),
  });

  const alerts = data?.data ?? [];
  const meta = data?.meta;

  const selectableAlerts = alerts.filter(
    (a) => a.status !== 'resolved' && a.status !== 'false_positive',
  );

  const toggleSelect = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const toggleSelectAll = useCallback(() => {
    setSelectedIds((prev) => {
      if (prev.size === selectableAlerts.length && selectableAlerts.length > 0) {
        return new Set();
      }
      return new Set(selectableAlerts.map((a) => a.id));
    });
  }, [selectableAlerts]);

  const clearSelection = useCallback(() => setSelectedIds(new Set()), []);

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

  const allSelected = selectableAlerts.length > 0 && selectedIds.size === selectableAlerts.length;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="UEBA Alerts"
          description="Multi-signal behavioral alerts linked back to their raw triggering events."
        />

        <div className="flex flex-wrap items-center gap-3">
          {selectableAlerts.length > 0 && (
            <div className="flex items-center gap-2">
              <Checkbox
                checked={allSelected}
                onCheckedChange={toggleSelectAll}
                aria-label="Select all alerts"
              />
              <span className="text-sm text-muted-foreground">Select all</span>
            </div>
          )}
          <Select
            value={statusFilter}
            onValueChange={(value) => {
              setStatusFilter(value);
              setPage(1);
              clearSelection();
            }}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Filter by status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Statuses</SelectItem>
              {UEBA_ALERT_STATUSES.map((status) => (
                <SelectItem key={status} value={status}>
                  {status.replace('_', ' ')}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {meta && (
            <span className="text-sm text-muted-foreground">
              {meta.total} alert{meta.total !== 1 ? 's' : ''}
            </span>
          )}
        </div>

        {selectedIds.size > 0 && (
          <BulkAlertActions selectedIds={Array.from(selectedIds)} onComplete={clearSelection} />
        )}

        <div className="grid gap-4">
          {alerts.map((alert) => {
            const isTerminal = alert.status === 'resolved' || alert.status === 'false_positive';
            return (
              <Card key={alert.id} className="border-border/70">
                <CardHeader className="gap-3">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="flex items-start gap-3">
                      {!isTerminal && (
                        <Checkbox
                          checked={selectedIds.has(alert.id)}
                          onCheckedChange={() => toggleSelect(alert.id)}
                          className="mt-1"
                          aria-label={`Select alert ${alert.title}`}
                        />
                      )}
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
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant={alert.severity === 'critical' ? 'destructive' : alert.severity === 'high' ? 'warning' : 'outline'}>
                        {alert.severity}
                      </Badge>
                      <Badge variant="secondary">
                        {(alert.confidence * 100).toFixed(0)}% confidence
                      </Badge>
                      <AlertActions alert={alert} />
                    </div>
                  </div>
                  <div className="text-sm text-muted-foreground">{alert.description}</div>
                </CardHeader>
                <CardContent className="grid grid-cols-1 gap-4 xl:grid-cols-[0.95fr_1.05fr]">
                  <div className="rounded-lg border bg-muted/20 p-3">
                    <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Risk Impact</div>
                    <div className="text-sm">
                      {alert.risk_score_before.toFixed(0)} → {alert.risk_score_after.toFixed(0)} ({alert.risk_score_delta >= 0 ? '+' : ''}{alert.risk_score_delta.toFixed(0)})
                    </div>
                    <div className="mt-3 text-xs text-muted-foreground">
                      Triggered by {alert.correlated_signal_count} signals across {(alert.triggering_event_ids ?? []).length} events.
                    </div>
                    <pre className="mt-3 overflow-auto rounded-md bg-background p-3 text-xs">
                      {JSON.stringify(alert.baseline_comparison, null, 2)}
                    </pre>
                  </div>
                  <SignalEvidenceViewer alert={alert} />
                </CardContent>
              </Card>
            );
          })}
          {alerts.length === 0 && (
            <Card>
              <CardContent className="p-8 text-center text-muted-foreground">
                No UEBA alerts{statusFilter !== 'all' ? ` with status "${statusFilter.replace('_', ' ')}"` : ''}.
              </CardContent>
            </Card>
          )}
        </div>

        {meta && meta.total_pages > 1 && (
          <div className="flex items-center justify-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => {
                setPage((p) => Math.max(1, p - 1));
                clearSelection();
              }}
            >
              Previous
            </Button>
            <span className="text-sm text-muted-foreground">
              Page {meta.page} of {meta.total_pages}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= meta.total_pages}
              onClick={() => {
                setPage((p) => p + 1);
                clearSelection();
              }}
            >
              Next
            </Button>
          </div>
        )}
      </div>
    </PermissionRedirect>
  );
}
