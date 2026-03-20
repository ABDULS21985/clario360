'use client';

import { useQuery } from '@tanstack/react-query';
import {
  Brain,
  AlertTriangle,
  ShieldAlert,
  Shield,
  Lock,
  CheckCircle2,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { AISecurityDashboard, AIDataUsage } from '@/types/cyber';

const RISK_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-700',
  high: 'bg-orange-100 text-orange-700',
  medium: 'bg-amber-100 text-amber-800',
  low: 'bg-blue-100 text-blue-700',
};

const STATUS_COLORS: Record<string, string> = {
  active: 'bg-green-100 text-green-700',
  inactive: 'bg-gray-100 text-gray-600',
  blocked: 'bg-red-100 text-red-700',
  under_review: 'bg-amber-100 text-amber-800',
};

function formatLabel(value: string): string {
  return value.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function AISecurityPage() {
  const {
    data: dashboardEnvelope,
    isLoading: dashLoading,
    error: dashError,
    refetch: refetchDash,
  } = useQuery({
    queryKey: ['dspm-ai-dashboard'],
    queryFn: () => apiGet<{ data: AISecurityDashboard }>(API_ENDPOINTS.CYBER_DSPM_AI_DASHBOARD),
    staleTime: 120000,
  });

  const {
    data: rankingEnvelope,
    isLoading: rankingLoading,
    error: rankingError,
    refetch: refetchRanking,
  } = useQuery({
    queryKey: ['dspm-ai-risk-ranking'],
    queryFn: () => apiGet<{ data: AIDataUsage[] }>(API_ENDPOINTS.CYBER_DSPM_AI_RISK_RANKING),
    staleTime: 120000,
  });

  const isLoading = dashLoading || rankingLoading;
  const error = dashError || rankingError;
  const dashboard = dashboardEnvelope?.data;
  const riskyUsages = rankingEnvelope?.data ?? [];

  const kpis = [
    {
      label: 'Total AI Data Usages',
      value: dashboard?.total_ai_data_usages ?? 0,
      icon: Brain,
      color: 'text-blue-600',
    },
    {
      label: 'High Risk Count',
      value: dashboard?.high_risk_count ?? 0,
      icon: AlertTriangle,
      color: 'text-orange-600',
    },
    {
      label: 'PII in AI Count',
      value: dashboard?.pii_in_ai_count ?? 0,
      icon: ShieldAlert,
      color: 'text-red-600',
    },
    {
      label: 'Consent Gap Count',
      value: dashboard?.consent_gap_count ?? 0,
      icon: Shield,
      color: 'text-amber-600',
    },
  ];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="AI Data Security"
          description="Monitor AI data usage risks, PII exposure, consent gaps, and anonymization posture across your AI pipelines"
        />

        {isLoading ? (
          <LoadingSkeleton variant="card" count={4} />
        ) : error ? (
          <ErrorState
            message="Failed to load AI security data"
            onRetry={() => {
              void refetchDash();
              void refetchRanking();
            }}
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
                      <div className={`rounded-lg bg-muted p-2.5 ${kpi.color}`}>
                        <Icon className="h-5 w-5" />
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">{kpi.label}</p>
                        <p className="text-2xl font-bold tabular-nums">{kpi.value.toLocaleString()}</p>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              {/* Risk Distribution */}
              <div className="rounded-xl border bg-card p-5">
                <h3 className="mb-4 text-sm font-semibold">Risk Distribution</h3>
                {dashboard?.risk_distribution && Object.keys(dashboard.risk_distribution).length > 0 ? (
                  <div className="space-y-3">
                    {(['critical', 'high', 'medium', 'low'] as const).map((level) => {
                      const count = dashboard.risk_distribution[level] ?? 0;
                      const total = dashboard.total_ai_data_usages || 1;
                      const pct = Math.round((count / total) * 100);
                      return (
                        <div key={level}>
                          <div className="mb-1 flex items-center justify-between text-xs">
                            <span className="flex items-center gap-2">
                              <Badge variant="secondary" className={`${RISK_COLORS[level]} capitalize`}>
                                {level}
                              </Badge>
                            </span>
                            <span className="font-medium tabular-nums">
                              {count} ({pct}%)
                            </span>
                          </div>
                          <div className="h-2 overflow-hidden rounded-full bg-muted">
                            <div
                              className={`h-full rounded-full transition-all ${
                                level === 'critical'
                                  ? 'bg-red-500'
                                  : level === 'high'
                                    ? 'bg-orange-500'
                                    : level === 'medium'
                                      ? 'bg-amber-500'
                                      : 'bg-blue-500'
                              }`}
                              style={{ width: `${pct}%` }}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No risk data available.</p>
                )}
              </div>

              {/* Usage Type Distribution */}
              <div className="rounded-xl border bg-card p-5">
                <h3 className="mb-4 text-sm font-semibold">Usage Type Distribution</h3>
                {dashboard?.usage_type_distribution && Object.keys(dashboard.usage_type_distribution).length > 0 ? (
                  <div className="space-y-3">
                    {Object.entries(dashboard.usage_type_distribution)
                      .sort(([, a], [, b]) => b - a)
                      .map(([usageType, count]) => {
                        const total = dashboard.total_ai_data_usages || 1;
                        const pct = Math.round((count / total) * 100);
                        return (
                          <div key={usageType}>
                            <div className="mb-1 flex items-center justify-between text-xs">
                              <span className="text-muted-foreground">{formatLabel(usageType)}</span>
                              <span className="font-medium tabular-nums">
                                {count} ({pct}%)
                              </span>
                            </div>
                            <div className="h-2 overflow-hidden rounded-full bg-muted">
                              <div
                                className="h-full rounded-full bg-primary/70 transition-all"
                                style={{ width: `${pct}%` }}
                              />
                            </div>
                          </div>
                        );
                      })}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No usage type data available.</p>
                )}
              </div>
            </div>

            {/* Top Risky AI Data Usages Table */}
            <div className="rounded-xl border bg-card">
              <div className="border-b px-5 py-4">
                <h3 className="text-sm font-semibold">Top Risky AI Data Usages</h3>
                <p className="text-xs text-muted-foreground">
                  AI data usages ranked by risk score, showing PII exposure and consent status
                </p>
              </div>
              <div className="p-5">
                {riskyUsages.length === 0 && (!dashboard?.top_risky_usages || dashboard.top_risky_usages.length === 0) ? (
                  <div className="flex flex-col items-center justify-center py-12 text-center">
                    <CheckCircle2 className="mb-3 h-8 w-8 text-green-500" />
                    <p className="text-sm font-medium">No Risky AI Data Usages</p>
                    <p className="text-xs text-muted-foreground">
                      All AI data usages are within acceptable risk thresholds.
                    </p>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Asset Name</TableHead>
                          <TableHead>Usage Type</TableHead>
                          <TableHead>Risk Level</TableHead>
                          <TableHead className="text-right">Risk Score</TableHead>
                          <TableHead>PII Types</TableHead>
                          <TableHead>Consent</TableHead>
                          <TableHead>Anonymization Level</TableHead>
                          <TableHead>Status</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(riskyUsages.length > 0 ? riskyUsages : dashboard?.top_risky_usages ?? []).map(
                          (usage: AIDataUsage) => (
                            <TableRow key={usage.id}>
                              <TableCell className="font-medium">
                                {usage.data_asset_name ?? usage.data_asset_id}
                              </TableCell>
                              <TableCell>
                                <Badge variant="outline" className="text-xs capitalize">
                                  {formatLabel(usage.usage_type)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant="secondary"
                                  className={`capitalize ${RISK_COLORS[usage.ai_risk_level] ?? 'bg-muted text-muted-foreground'}`}
                                >
                                  {usage.ai_risk_level}
                                </Badge>
                              </TableCell>
                              <TableCell className="text-right tabular-nums font-medium">
                                {usage.ai_risk_score}
                              </TableCell>
                              <TableCell>
                                {usage.pii_types.length > 0 ? (
                                  <div className="flex flex-wrap gap-1">
                                    {usage.pii_types.slice(0, 3).map((pii) => (
                                      <Badge key={pii} variant="outline" className="text-xs">
                                        {formatLabel(pii)}
                                      </Badge>
                                    ))}
                                    {usage.pii_types.length > 3 && (
                                      <Badge variant="outline" className="text-xs text-muted-foreground">
                                        +{usage.pii_types.length - 3}
                                      </Badge>
                                    )}
                                  </div>
                                ) : (
                                  <span className="text-xs text-muted-foreground">None</span>
                                )}
                              </TableCell>
                              <TableCell>
                                {usage.consent_verified ? (
                                  <div className="flex items-center gap-1 text-green-600">
                                    <Lock className="h-3.5 w-3.5" />
                                    <span className="text-xs font-medium">Verified</span>
                                  </div>
                                ) : (
                                  <div className="flex items-center gap-1 text-red-600">
                                    <AlertTriangle className="h-3.5 w-3.5" />
                                    <span className="text-xs font-medium">Gap</span>
                                  </div>
                                )}
                              </TableCell>
                              <TableCell>
                                <span className="text-xs capitalize">
                                  {usage.anonymization_level
                                    ? formatLabel(usage.anonymization_level)
                                    : 'N/A'}
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant="secondary"
                                  className={`capitalize ${STATUS_COLORS[usage.status] ?? 'bg-muted text-muted-foreground'}`}
                                >
                                  {formatLabel(usage.status)}
                                </Badge>
                              </TableCell>
                            </TableRow>
                          ),
                        )}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
