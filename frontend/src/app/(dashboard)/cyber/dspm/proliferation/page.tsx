'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  TrendingUp,
  AlertTriangle,
  CheckCircle2,
  Shield,
  Copy,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { ProliferationOverview, DataProliferation, SpreadEvent } from '@/types/cyber';

const STATUS_CONFIG: Record<string, { label: string; color: string; icon: typeof CheckCircle2 }> = {
  contained: { label: 'Contained', color: 'bg-green-100 text-green-700', icon: CheckCircle2 },
  spreading: { label: 'Spreading', color: 'bg-amber-100 text-amber-800', icon: TrendingUp },
  uncontrolled: { label: 'Uncontrolled', color: 'bg-red-100 text-red-700', icon: AlertTriangle },
};

const CLASSIFICATION_COLORS: Record<string, string> = {
  public: 'bg-green-100 text-green-700',
  internal: 'bg-blue-100 text-blue-700',
  confidential: 'bg-amber-100 text-amber-800',
  restricted: 'bg-red-100 text-red-700',
  top_secret: 'bg-purple-100 text-purple-700',
};

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function DataProliferationPage() {
  const [expandedAssets, setExpandedAssets] = useState<Set<string>>(new Set());

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['dspm-proliferation-overview'],
    queryFn: () =>
      apiGet<{ data: ProliferationOverview }>(API_ENDPOINTS.CYBER_DSPM_PROLIFERATION_OVERVIEW),
  });

  const overview = data?.data;

  function toggleExpanded(assetId: string) {
    setExpandedAssets((prev) => {
      const next = new Set(prev);
      if (next.has(assetId)) {
        next.delete(assetId);
      } else {
        next.add(assetId);
      }
      return next;
    });
  }

  const kpis = overview
    ? [
        {
          label: 'Total Tracked Assets',
          value: overview.total_tracked_assets,
          icon: Shield,
          color: 'text-blue-600',
        },
        {
          label: 'Spreading',
          value: overview.spreading_count,
          icon: TrendingUp,
          color: 'text-amber-600',
        },
        {
          label: 'Uncontrolled',
          value: overview.uncontrolled_count,
          icon: AlertTriangle,
          color: 'text-red-600',
        },
        {
          label: 'Unauthorized Copies',
          value: overview.total_unauthorized_copies,
          icon: Copy,
          color: 'text-red-600',
        },
      ]
    : [];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Proliferation"
          description="Track data asset spread, detect unauthorized copies, and monitor proliferation status across your environment"
        />

        {isLoading ? (
          <LoadingSkeleton variant="card" count={4} />
        ) : error || !overview ? (
          <ErrorState
            message="Failed to load proliferation data"
            onRetry={() => void refetch()}
          />
        ) : (
          <>
            {/* KPI Cards */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {kpis.map((kpi) => {
                const Icon = kpi.icon;
                return (
                  <Card key={kpi.label}>
                    <CardContent className="flex items-center gap-4 p-5">
                      <Icon className={`h-5 w-5 ${kpi.color}`} />
                      <div>
                        <p className="text-xs text-muted-foreground">{kpi.label}</p>
                        <p className="text-2xl font-bold tabular-nums">
                          {kpi.value.toLocaleString()}
                        </p>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>

            {/* Proliferations List */}
            {overview.proliferations.length === 0 ? (
              <Card>
                <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                  <CheckCircle2 className="mb-3 h-8 w-8 text-green-500" />
                  <p className="text-sm font-medium">No Data Proliferation Detected</p>
                  <p className="text-xs text-muted-foreground">
                    All tracked data assets are contained with no unauthorized copies.
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className="rounded-xl border bg-card">
                <div className="border-b px-5 py-4">
                  <h3 className="text-sm font-semibold">Tracked Data Assets</h3>
                  <p className="text-xs text-muted-foreground">
                    {overview.proliferations.length} asset
                    {overview.proliferations.length !== 1 ? 's' : ''} tracked for proliferation
                  </p>
                </div>
                <div className="divide-y">
                  {overview.proliferations.map((proliferation: DataProliferation) => {
                    const isExpanded = expandedAssets.has(proliferation.asset_id);
                    const statusConfig =
                      STATUS_CONFIG[proliferation.status] ?? STATUS_CONFIG.contained;
                    const StatusIcon = statusConfig.icon;
                    const classificationColor =
                      CLASSIFICATION_COLORS[proliferation.classification] ??
                      'bg-muted text-muted-foreground';

                    return (
                      <div key={proliferation.asset_id}>
                        {/* Asset Row */}
                        <button
                          type="button"
                          className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left transition-colors hover:bg-muted/50"
                          onClick={() => toggleExpanded(proliferation.asset_id)}
                        >
                          <div className="min-w-0 flex-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <p className="text-sm font-medium">{proliferation.asset_name}</p>
                              <span
                                className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${classificationColor}`}
                              >
                                {proliferation.classification.replace(/_/g, ' ')}
                              </span>
                            </div>
                            <div className="mt-1.5 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                              <span className="tabular-nums">
                                {proliferation.total_copies} total{' '}
                                {proliferation.total_copies === 1 ? 'copy' : 'copies'}
                              </span>
                              <span className="tabular-nums">
                                {proliferation.authorized_copies} authorized
                              </span>
                              {proliferation.unauthorized_copies > 0 && (
                                <span className="tabular-nums font-medium text-red-600">
                                  {proliferation.unauthorized_copies} unauthorized
                                </span>
                              )}
                            </div>
                          </div>

                          <div className="flex shrink-0 items-center gap-3">
                            <span
                              className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${statusConfig.color}`}
                            >
                              <StatusIcon className="h-3 w-3" />
                              {statusConfig.label}
                            </span>
                            {proliferation.spread_events.length > 0 && (
                              <span className="text-muted-foreground">
                                {isExpanded ? (
                                  <ChevronUp className="h-4 w-4" />
                                ) : (
                                  <ChevronDown className="h-4 w-4" />
                                )}
                              </span>
                            )}
                          </div>
                        </button>

                        {/* Expanded Spread Events */}
                        {isExpanded && proliferation.spread_events.length > 0 && (
                          <div className="border-t bg-muted/20 px-5 py-3">
                            <p className="mb-3 text-xs font-medium text-muted-foreground">
                              Spread Events ({proliferation.spread_events.length})
                            </p>
                            <div className="space-y-2">
                              {proliferation.spread_events.map(
                                (event: SpreadEvent, idx: number) => (
                                  <div
                                    key={`${event.target_asset_id}-${idx}`}
                                    className="flex items-start justify-between gap-4 rounded-lg border bg-background p-3"
                                  >
                                    <div className="min-w-0 space-y-1">
                                      <div className="flex flex-wrap items-center gap-2">
                                        <p className="text-sm font-medium">
                                          {event.target_asset_name}
                                        </p>
                                        <Badge variant="outline" className="text-xs capitalize">
                                          {event.edge_type.replace(/_/g, ' ')}
                                        </Badge>
                                        {event.classification_changed && (
                                          <Badge
                                            variant="secondary"
                                            className="text-xs"
                                          >
                                            Classification Changed
                                          </Badge>
                                        )}
                                      </div>
                                      <p className="text-xs text-muted-foreground">
                                        Detected {formatDate(event.detected_at)}
                                      </p>
                                    </div>
                                    <span
                                      className={`inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                                        event.authorized
                                          ? 'bg-green-100 text-green-700'
                                          : 'bg-red-100 text-red-700'
                                      }`}
                                    >
                                      {event.authorized ? (
                                        <>
                                          <CheckCircle2 className="h-3 w-3" />
                                          Authorized
                                        </>
                                      ) : (
                                        <>
                                          <AlertTriangle className="h-3 w-3" />
                                          Unauthorized
                                        </>
                                      )}
                                    </span>
                                  </div>
                                )
                              )}
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
