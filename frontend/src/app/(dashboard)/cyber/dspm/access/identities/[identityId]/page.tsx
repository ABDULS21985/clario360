'use client';

import { useRouter, useParams } from 'next/navigation';
import { format } from 'date-fns';
import { useState } from 'react';
import { ArrowLeft, CheckCircle, ChevronLeft, ChevronRight, Clock, XCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import type {
  IdentityProfile,
  AccessMapping,
  AccessAuditEntry,
  BlastRadius,
  AccessRecommendation,
  DataClassification,
} from '@/types/cyber';
import type { PaginationMeta } from '@/types/api';

// ─── Helpers ─────────────────────────────────────────────────────────────────

function scoreColor(score: number): string {
  if (score >= 75) return 'bg-red-500';
  if (score >= 50) return 'bg-orange-500';
  if (score >= 25) return 'bg-amber-500';
  return 'bg-green-500';
}

function scoreBorderColor(score: number): string {
  if (score >= 75) return 'border-red-300';
  if (score >= 50) return 'border-orange-300';
  if (score >= 25) return 'border-amber-300';
  return 'border-green-300';
}

function scoreTextColor(score: number): string {
  if (score >= 75) return 'text-red-700';
  if (score >= 50) return 'text-orange-700';
  if (score >= 25) return 'text-amber-700';
  return 'text-green-700';
}

const STATUS_VARIANT: Record<string, string> = {
  active: 'bg-green-100 text-green-800 border-green-200',
  inactive: 'bg-gray-100 text-gray-800 border-gray-200',
  under_review: 'bg-amber-100 text-amber-800 border-amber-200',
  remediated: 'bg-blue-100 text-blue-800 border-blue-200',
};

const CLASSIFICATION_COLORS: Record<DataClassification, string> = {
  public: 'bg-gray-100 text-gray-800 border-gray-200',
  internal: 'bg-blue-100 text-blue-800 border-blue-200',
  confidential: 'bg-amber-100 text-amber-800 border-amber-200',
  restricted: 'bg-red-100 text-red-800 border-red-200',
};

const RECOMMENDATION_TYPE_COLORS: Record<string, string> = {
  revoke: 'bg-red-100 text-red-800 border-red-200',
  downgrade: 'bg-amber-100 text-amber-800 border-amber-200',
  time_bound: 'bg-blue-100 text-blue-800 border-blue-200',
  review: 'bg-gray-100 text-gray-800 border-gray-200',
};

function statusLabel(status: string): string {
  return status
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function identityTypeLabel(type: string): string {
  return type
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

// ─── Page ────────────────────────────────────────────────────────────────────

export default function DspmAccessIdentityDetailPage() {
  const router = useRouter();
  const params = useParams<{ identityId: string }>();
  const identityId = params?.identityId ?? '';

  const {
    data: profileEnvelope,
    isLoading: profileLoading,
    error: profileError,
    mutate: refetchProfile,
  } = useRealtimeData<{ data: IdentityProfile }>(
    `${API_ENDPOINTS.CYBER_DSPM_ACCESS_IDENTITIES}/${identityId}`,
    { pollInterval: 60000 },
  );

  const {
    data: mappingsEnvelope,
    isLoading: mappingsLoading,
    error: mappingsError,
    mutate: refetchMappings,
  } = useRealtimeData<{ data: AccessMapping[] }>(
    `${API_ENDPOINTS.CYBER_DSPM_ACCESS_IDENTITIES}/${identityId}/mappings`,
    { pollInterval: 60000 },
  );

  const {
    data: blastEnvelope,
    isLoading: blastLoading,
    error: blastError,
    mutate: refetchBlast,
  } = useRealtimeData<{ data: BlastRadius }>(
    `${API_ENDPOINTS.CYBER_DSPM_ACCESS_IDENTITIES}/${identityId}/blast-radius`,
    { pollInterval: 60000 },
  );

  const {
    data: recsEnvelope,
    isLoading: recsLoading,
    error: recsError,
    mutate: refetchRecs,
  } = useRealtimeData<{ data: AccessRecommendation[] }>(
    `${API_ENDPOINTS.CYBER_DSPM_ACCESS_IDENTITIES}/${identityId}/recommendations`,
    { pollInterval: 60000 },
  );

  const [auditPage, setAuditPage] = useState(1);
  const {
    data: auditEnvelope,
    isLoading: auditLoading,
    error: auditError,
    mutate: refetchAudit,
  } = useRealtimeData<{ data: AccessAuditEntry[]; meta: PaginationMeta }>(
    `${API_ENDPOINTS.CYBER_DSPM_ACCESS_IDENTITIES}/${identityId}/audit?page=${auditPage}&per_page=25`,
    { pollInterval: 60000 },
  );

  const auditEntries = auditEnvelope?.data ?? [];
  const auditMeta = auditEnvelope?.meta;

  const profile = profileEnvelope?.data;
  const mappings = mappingsEnvelope?.data ?? [];
  const blast = blastEnvelope?.data;
  const recommendations = recsEnvelope?.data ?? [];

  if (profileLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <div className="space-y-6">
          <LoadingSkeleton variant="card" />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (profileError || !profile) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState
          message="Failed to load identity profile"
          onRetry={() => void refetchProfile()}
        />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {/* Header */}
        <PageHeader
          title={profile.identity_name}
          description={`${identityTypeLabel(profile.identity_type)} \u00b7 ${profile.identity_source}`}
          actions={
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push('/cyber/dspm/access/identities')}
            >
              <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
              Back
            </Button>
          }
        />

        {/* Score Cards + Status */}
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <Card className={`border-2 ${scoreBorderColor(profile.access_risk_score)}`}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                Risk Score
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-end gap-3">
                <span className={`text-4xl font-bold ${scoreTextColor(profile.access_risk_score)}`}>
                  {profile.access_risk_score}
                </span>
                <span className="mb-1 text-sm text-muted-foreground">/ 100</span>
              </div>
              <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted">
                <div
                  className={`h-full rounded-full ${scoreColor(profile.access_risk_score)}`}
                  style={{ width: `${Math.min(profile.access_risk_score, 100)}%` }}
                />
              </div>
            </CardContent>
          </Card>

          <Card className={`border-2 ${scoreBorderColor(profile.blast_radius_score)}`}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                Blast Radius Score
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-end gap-3">
                <span className={`text-4xl font-bold ${scoreTextColor(profile.blast_radius_score)}`}>
                  {profile.blast_radius_score}
                </span>
                <span className="mb-1 text-sm text-muted-foreground">/ 100</span>
              </div>
              <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted">
                <div
                  className={`h-full rounded-full ${scoreColor(profile.blast_radius_score)}`}
                  style={{ width: `${Math.min(profile.blast_radius_score, 100)}%` }}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                Status
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col gap-3">
                <Badge
                  variant="outline"
                  className={`w-fit text-sm ${STATUS_VARIANT[profile.status] ?? 'bg-gray-100 text-gray-800'}`}
                >
                  {statusLabel(profile.status)}
                </Badge>
                <div className="space-y-1 text-xs text-muted-foreground">
                  <p>
                    {profile.total_assets_accessible} assets accessible
                  </p>
                  <p>
                    {profile.overprivileged_count} overprivileged
                    {' \u00b7 '}
                    {profile.stale_permission_count} stale
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Tabs */}
        <Tabs defaultValue="access-map" className="space-y-4">
          <TabsList>
            <TabsTrigger value="access-map">Access Map</TabsTrigger>
            <TabsTrigger value="blast-radius">Blast Radius</TabsTrigger>
            <TabsTrigger value="recommendations">Recommendations</TabsTrigger>
            <TabsTrigger value="audit-trail">Audit Trail</TabsTrigger>
          </TabsList>

          {/* ── Access Map Tab ──────────────────────────────────────────────── */}
          <TabsContent value="access-map" className="space-y-4">
            {mappingsLoading ? (
              <LoadingSkeleton variant="table-row" />
            ) : mappingsError ? (
              <ErrorState
                message="Failed to load access mappings"
                onRetry={() => void refetchMappings()}
              />
            ) : mappings.length === 0 ? (
              <Card>
                <CardContent className="py-8 text-center text-sm text-muted-foreground">
                  No access mappings found for this identity.
                </CardContent>
              </Card>
            ) : (
              <div className="overflow-x-auto rounded-xl border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left font-medium">Data Asset</th>
                      <th className="px-4 py-3 text-left font-medium">Classification</th>
                      <th className="px-4 py-3 text-left font-medium">Permission</th>
                      <th className="px-4 py-3 text-left font-medium">Source</th>
                      <th className="px-4 py-3 text-left font-medium">Stale</th>
                      <th className="px-4 py-3 text-right font-medium">Usage (90d)</th>
                      <th className="px-4 py-3 text-left font-medium">Last Used</th>
                      <th className="px-4 py-3 text-right font-medium">Risk Score</th>
                    </tr>
                  </thead>
                  <tbody>
                    {mappings.map((m) => (
                      <tr key={m.id} className="border-b last:border-b-0 hover:bg-muted/30">
                        <td className="px-4 py-3 font-medium">{m.data_asset_name}</td>
                        <td className="px-4 py-3">
                          <Badge
                            variant="outline"
                            className={CLASSIFICATION_COLORS[m.data_classification] ?? ''}
                          >
                            {statusLabel(m.data_classification)}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <span className="rounded bg-muted px-2 py-0.5 text-xs font-mono">
                            {m.permission_type}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">
                          {statusLabel(m.permission_source)}
                        </td>
                        <td className="px-4 py-3">
                          {m.is_stale ? (
                            <Badge variant="destructive" className="text-[10px]">
                              Stale
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="text-[10px]">
                              Active
                            </Badge>
                          )}
                        </td>
                        <td className="px-4 py-3 text-right tabular-nums">
                          {m.usage_count_90d}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">
                          {m.last_used_at
                            ? format(new Date(m.last_used_at), 'MMM d, yyyy')
                            : 'Never'}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <span className={`text-xs font-semibold ${scoreTextColor(m.access_risk_score)}`}>
                            {m.access_risk_score}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </TabsContent>

          {/* ── Blast Radius Tab ────────────────────────────────────────────── */}
          <TabsContent value="blast-radius" className="space-y-4">
            {blastLoading ? (
              <LoadingSkeleton variant="card" />
            ) : blastError ? (
              <ErrorState
                message="Failed to load blast radius data"
                onRetry={() => void refetchBlast()}
              />
            ) : !blast ? (
              <Card>
                <CardContent className="py-8 text-center text-sm text-muted-foreground">
                  No blast radius data available.
                </CardContent>
              </Card>
            ) : (
              <>
                {/* Summary Cards */}
                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        Total Assets Exposed
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <span className="text-3xl font-bold">{blast.total_assets_exposed}</span>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        Sensitive Assets
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <span className="text-3xl font-bold text-amber-600">
                        {blast.sensitive_assets}
                      </span>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">
                        Weighted Score
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <span className={`text-3xl font-bold ${scoreTextColor(blast.weighted_score)}`}>
                        {blast.weighted_score.toFixed(1)}
                      </span>
                    </CardContent>
                  </Card>
                </div>

                {/* Top Risky Assets */}
                {blast.top_risky_assets.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-sm font-semibold">
                        Top Risky Assets
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        {blast.top_risky_assets.map((asset) => (
                          <div
                            key={`${asset.data_asset_id}-${asset.permission_type}`}
                            className="flex items-center justify-between rounded-lg border p-3"
                          >
                            <div className="space-y-1">
                              <p className="text-sm font-medium">{asset.data_asset_name}</p>
                              <div className="flex items-center gap-2">
                                <Badge
                                  variant="outline"
                                  className={CLASSIFICATION_COLORS[asset.data_classification] ?? ''}
                                >
                                  {statusLabel(asset.data_classification)}
                                </Badge>
                                <span className="rounded bg-muted px-2 py-0.5 text-xs font-mono">
                                  {asset.permission_type}
                                </span>
                              </div>
                            </div>
                            <div className="text-right">
                              <p className={`text-lg font-bold ${scoreTextColor(asset.weighted_score)}`}>
                                {asset.weighted_score.toFixed(1)}
                              </p>
                              <p className="text-[10px] text-muted-foreground">weighted score</p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                )}

                {/* Escalation Paths */}
                {blast.escalation_paths.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-sm font-semibold">
                        Escalation Paths
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        {blast.escalation_paths.map((path, idx) => (
                          <div
                            key={`${path.asset_id}-${path.pattern}-${idx}`}
                            className="rounded-lg border p-3"
                          >
                            <div className="flex items-start justify-between gap-4">
                              <div className="space-y-1">
                                <p className="text-sm font-medium">{path.pattern}</p>
                                <p className="text-xs text-muted-foreground">
                                  {path.from_permission} &rarr; {path.to_permission} on{' '}
                                  <span className="font-medium">{path.asset_name}</span>
                                </p>
                                {path.mitre_technique && (
                                  <p className="text-[10px] text-muted-foreground">
                                    MITRE: {path.mitre_technique}
                                  </p>
                                )}
                              </div>
                              <Badge
                                variant="outline"
                                className={
                                  path.severity === 'critical'
                                    ? 'bg-red-100 text-red-800 border-red-200'
                                    : path.severity === 'high'
                                      ? 'bg-orange-100 text-orange-800 border-orange-200'
                                      : 'bg-amber-100 text-amber-800 border-amber-200'
                                }
                              >
                                {statusLabel(path.severity)}
                              </Badge>
                            </div>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                )}
              </>
            )}
          </TabsContent>

          {/* ── Recommendations Tab ─────────────────────────────────────────── */}
          <TabsContent value="recommendations" className="space-y-4">
            {recsLoading ? (
              <LoadingSkeleton variant="card" />
            ) : recsError ? (
              <ErrorState
                message="Failed to load recommendations"
                onRetry={() => void refetchRecs()}
              />
            ) : recommendations.length === 0 ? (
              <Card>
                <CardContent className="py-8 text-center text-sm text-muted-foreground">
                  No recommendations at this time.
                </CardContent>
              </Card>
            ) : (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                {recommendations.map((rec, idx) => (
                  <Card key={`${rec.permission_id}-${rec.type}-${idx}`}>
                    <CardHeader className="pb-2">
                      <div className="flex items-center justify-between">
                        <Badge
                          variant="outline"
                          className={RECOMMENDATION_TYPE_COLORS[rec.type] ?? 'bg-gray-100 text-gray-800'}
                        >
                          {statusLabel(rec.type)}
                        </Badge>
                        <span className={`text-xs font-semibold ${scoreTextColor(rec.risk_reduction_estimate)}`}>
                          -{rec.risk_reduction_estimate} risk
                        </span>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-2">
                      <div className="space-y-1">
                        <p className="text-sm font-medium">{rec.asset_name}</p>
                        <div className="flex items-center gap-2">
                          <span className="rounded bg-muted px-2 py-0.5 text-xs font-mono">
                            {rec.permission_type}
                          </span>
                          <Badge
                            variant="outline"
                            className={CLASSIFICATION_COLORS[rec.data_classification] ?? ''}
                          >
                            {statusLabel(rec.data_classification)}
                          </Badge>
                        </div>
                      </div>
                      <p className="text-xs text-muted-foreground">{rec.reason}</p>
                      <p className="text-xs">
                        <span className="font-medium">Impact:</span>{' '}
                        <span className="text-muted-foreground">{rec.impact}</span>
                      </p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </TabsContent>

          {/* ── Audit Trail Tab ─────────────────────────────────────────────── */}
          <TabsContent value="audit-trail" className="space-y-4">
            {auditLoading ? (
              <LoadingSkeleton variant="table-row" count={5} />
            ) : auditError ? (
              <ErrorState
                message="Failed to load audit trail"
                onRetry={() => void refetchAudit()}
              />
            ) : auditEntries.length === 0 ? (
              <Card>
                <CardContent className="flex flex-col items-center justify-center py-12">
                  <Clock className="mb-3 h-10 w-10 text-muted-foreground/50" />
                  <p className="text-sm font-medium text-muted-foreground">No audit events</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    No access audit events have been recorded for this identity.
                  </p>
                </CardContent>
              </Card>
            ) : (
              <>
                <div className="overflow-x-auto rounded-xl border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-4 py-3 text-left font-medium">Action</th>
                        <th className="px-4 py-3 text-left font-medium">Table</th>
                        <th className="px-4 py-3 text-left font-medium">Database</th>
                        <th className="px-4 py-3 text-left font-medium">Source IP</th>
                        <th className="px-4 py-3 text-right font-medium">Rows</th>
                        <th className="px-4 py-3 text-right font-medium">Duration</th>
                        <th className="px-4 py-3 text-center font-medium">Status</th>
                        <th className="px-4 py-3 text-left font-medium">Time</th>
                      </tr>
                    </thead>
                    <tbody>
                      {auditEntries.map((entry) => (
                        <tr key={entry.id} className="border-b last:border-b-0 hover:bg-muted/30">
                          <td className="px-4 py-3">
                            <span className="rounded bg-muted px-2 py-0.5 text-xs font-mono uppercase">
                              {entry.action}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground">
                            {entry.table_name || '--'}
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground">
                            {entry.database_name || '--'}
                          </td>
                          <td className="px-4 py-3 text-xs font-mono text-muted-foreground">
                            {entry.source_ip || '--'}
                          </td>
                          <td className="px-4 py-3 text-right text-xs tabular-nums text-muted-foreground">
                            {entry.rows_affected ?? '--'}
                          </td>
                          <td className="px-4 py-3 text-right text-xs tabular-nums text-muted-foreground">
                            {entry.duration_ms != null ? `${entry.duration_ms}ms` : '--'}
                          </td>
                          <td className="px-4 py-3 text-center">
                            {entry.success ? (
                              <CheckCircle className="mx-auto h-4 w-4 text-green-500" />
                            ) : (
                              <XCircle className="mx-auto h-4 w-4 text-red-500" />
                            )}
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground">
                            {format(new Date(entry.event_timestamp), 'MMM d, yyyy HH:mm')}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                {auditMeta && auditMeta.total_pages > 1 && (
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>
                      Page {auditMeta.page} of {auditMeta.total_pages} ({auditMeta.total} events)
                    </span>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={auditPage <= 1}
                        onClick={() => setAuditPage((p) => Math.max(1, p - 1))}
                      >
                        <ChevronLeft className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={auditPage >= auditMeta.total_pages}
                        onClick={() => setAuditPage((p) => p + 1)}
                      >
                        <ChevronRight className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </PermissionRedirect>
  );
}
