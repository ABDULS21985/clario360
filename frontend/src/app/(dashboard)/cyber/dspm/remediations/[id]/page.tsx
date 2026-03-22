'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  CheckCircle2,
  Circle,
  Clock,
  Loader2,
  RefreshCw,
  RotateCcw,
  ShieldCheck,
  XCircle,
  AlertTriangle,
  User,
  Bot,
  Settings2,
  CalendarClock,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type { DSPMRemediation, DSPMRemediationHistory, CyberSeverity } from '@/types/cyber';

const STATUS_COLORS: Record<string, string> = {
  open: 'bg-blue-100 text-blue-700',
  in_progress: 'bg-amber-100 text-amber-800',
  awaiting_approval: 'bg-purple-100 text-purple-700',
  completed: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  cancelled: 'bg-gray-100 text-gray-600',
  rolled_back: 'bg-orange-100 text-orange-700',
  exception_granted: 'bg-teal-100 text-teal-700',
};

const STEP_ICONS: Record<string, typeof CheckCircle2> = {
  completed: CheckCircle2,
  running: Loader2,
  failed: XCircle,
  pending: Circle,
  skipped: Circle,
};

const STEP_COLORS: Record<string, string> = {
  completed: 'text-green-500',
  running: 'text-amber-500 animate-spin',
  failed: 'text-red-500',
  pending: 'text-muted-foreground',
  skipped: 'text-gray-400',
};

const ACTOR_ICONS: Record<string, typeof User> = {
  user: User,
  system: Bot,
  policy_engine: Settings2,
  scheduler: CalendarClock,
};

export default function RemediationDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params?.id ?? '';

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['cyber-dspm-remediation', id],
    queryFn: () => apiGet<{ data: DSPMRemediation }>(API_ENDPOINTS.CYBER_DSPM_REMEDIATIONS + '/' + id),
  });

  const { data: historyData, isLoading: historyLoading } = useQuery({
    queryKey: ['cyber-dspm-remediation-history', id],
    queryFn: () => apiGet<{ data: DSPMRemediationHistory[] }>(API_ENDPOINTS.CYBER_DSPM_REMEDIATIONS + '/' + id + '/history'),
  });

  const remediation = data?.data;
  const history = historyData?.data ?? [];

  async function handleApprove() {
    try {
      await apiPost(API_ENDPOINTS.CYBER_DSPM_REMEDIATIONS + '/' + id + '/approve');
      toast.success('Remediation approved');
      await refetch();
    } catch {
      toast.error('Failed to approve remediation');
    }
  }

  async function handleCancel() {
    const reason = window.prompt('Provide a reason for cancelling this remediation:');
    if (!reason) return;
    try {
      await apiPost(API_ENDPOINTS.CYBER_DSPM_REMEDIATIONS + '/' + id + '/cancel', { reason });
      toast.success('Remediation cancelled');
      await refetch();
    } catch {
      toast.error('Failed to cancel remediation');
    }
  }

  async function handleRollback() {
    const reason = window.prompt('Provide a reason for rolling back this remediation:');
    if (!reason) return;
    try {
      await apiPost(API_ENDPOINTS.CYBER_DSPM_REMEDIATIONS + '/' + id + '/rollback', { reason });
      toast.success('Rollback initiated');
      await refetch();
    } catch {
      toast.error('Failed to initiate rollback');
    }
  }

  function formatSlaStatus(r: DSPMRemediation): { text: string; color: string } {
    if (r.sla_breached) return { text: 'SLA Breached', color: 'text-red-600' };
    if (!r.sla_due_at) return { text: 'No SLA', color: 'text-muted-foreground' };
    const now = new Date();
    const due = new Date(r.sla_due_at);
    const diffMs = due.getTime() - now.getTime();
    if (diffMs <= 0) return { text: 'SLA Breached', color: 'text-red-600' };
    const hours = Math.floor(diffMs / (1000 * 60 * 60));
    if (hours >= 24) {
      const days = Math.floor(hours / 24);
      return { text: `${days}d ${hours % 24}h remaining`, color: days <= 1 ? 'text-amber-600' : 'text-green-600' };
    }
    return { text: `${hours}h remaining`, color: hours <= 4 ? 'text-amber-600' : 'text-green-600' };
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {isLoading ? (
          <>
            <div className="h-8 w-64 animate-pulse rounded bg-muted" />
            <LoadingSkeleton variant="card" count={3} />
          </>
        ) : error || !remediation ? (
          <ErrorState message="Failed to load remediation details" onRetry={() => void refetch()} />
        ) : (
          <>
            <PageHeader
              title={
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => router.push('/cyber/dspm/remediations')}
                    className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </button>
                  <span className="truncate">{remediation.title}</span>
                </div>
              }
              description={
                <div className="flex flex-wrap items-center gap-3 pl-11">
                  <SeverityIndicator severity={remediation.severity} size="sm" />
                  <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${STATUS_COLORS[remediation.status] ?? 'bg-muted text-muted-foreground'}`}>
                    {remediation.status.replace(/_/g, ' ')}
                  </span>
                  {(() => {
                    const sla = formatSlaStatus(remediation);
                    return (
                      <span className={`flex items-center gap-1 text-xs font-medium ${sla.color}`}>
                        <Clock className="h-3 w-3" />
                        {sla.text}
                      </span>
                    );
                  })()}
                </div>
              }
              actions={
                <div className="flex items-center gap-2">
                  {remediation.status === 'awaiting_approval' && (
                    <Button size="sm" onClick={handleApprove}>
                      <ShieldCheck className="mr-1.5 h-4 w-4" />
                      Approve
                    </Button>
                  )}
                  {remediation.rollback_available && !remediation.rolled_back && remediation.status === 'completed' && (
                    <Button variant="outline" size="sm" onClick={handleRollback}>
                      <RotateCcw className="mr-1.5 h-4 w-4" />
                      Rollback
                    </Button>
                  )}
                  {['open', 'in_progress', 'awaiting_approval'].includes(remediation.status) && (
                    <Button variant="destructive" size="sm" onClick={handleCancel}>
                      <XCircle className="mr-1.5 h-4 w-4" />
                      Cancel
                    </Button>
                  )}
                  <Button variant="outline" size="sm" onClick={() => void refetch()}>
                    <RefreshCw className="mr-1.5 h-4 w-4" />
                    Refresh
                  </Button>
                </div>
              }
            />

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
              <Card>
                <CardContent className="p-4 text-center">
                  <p className="text-xs text-muted-foreground">Finding Type</p>
                  <p className="mt-1 text-sm font-medium capitalize">{remediation.finding_type.replace(/_/g, ' ')}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <p className="text-xs text-muted-foreground">Asset</p>
                  <p className="mt-1 text-sm font-medium truncate">{remediation.data_asset_name ?? '--'}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <p className="text-xs text-muted-foreground">Assigned To</p>
                  <p className="mt-1 text-sm font-medium">{remediation.assigned_to ?? 'Unassigned'}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <p className="text-xs text-muted-foreground">Risk Before</p>
                  <p className="mt-1 text-lg font-bold tabular-nums text-red-600">{remediation.risk_score_before?.toFixed(0) ?? '--'}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <p className="text-xs text-muted-foreground">Risk After</p>
                  <p className="mt-1 text-lg font-bold tabular-nums text-green-600">{remediation.risk_score_after?.toFixed(0) ?? '--'}</p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <p className="text-xs text-muted-foreground">Reduction</p>
                  <p className="mt-1 text-lg font-bold tabular-nums text-blue-600">{remediation.risk_reduction?.toFixed(1) ?? '--'}</p>
                </CardContent>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Remediation Steps</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="relative space-y-0">
                  {(remediation.steps ?? []).map((step, idx) => {
                    const Icon = STEP_ICONS[step.status] ?? Circle;
                    const iconColor = STEP_COLORS[step.status] ?? 'text-muted-foreground';
                    const isLast = idx === remediation.steps.length - 1;
                    return (
                      <div key={step.step_id} className="relative flex gap-4 pb-6">
                        {!isLast && (
                          <div className="absolute left-[11px] top-6 h-full w-px bg-border" />
                        )}
                        <div className="relative z-10 mt-0.5 flex-shrink-0">
                          <Icon className={`h-6 w-6 ${iconColor}`} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between gap-2">
                            <p className="text-sm font-medium">
                              <span className="text-muted-foreground mr-2">Step {step.order}</span>
                              {step.action.replace(/_/g, ' ')}
                            </p>
                            <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${
                              step.status === 'completed' ? 'bg-green-100 text-green-700'
                              : step.status === 'running' ? 'bg-amber-100 text-amber-800'
                              : step.status === 'failed' ? 'bg-red-100 text-red-700'
                              : 'bg-muted text-muted-foreground'
                            }`}>
                              {step.status}
                            </span>
                          </div>
                          <p className="mt-0.5 text-xs text-muted-foreground">{step.description}</p>
                          {step.started_at && (
                            <p className="mt-1 text-xs text-muted-foreground">
                              Started: {new Date(step.started_at).toLocaleString()}
                              {step.completed_at && ` | Completed: ${new Date(step.completed_at).toLocaleString()}`}
                            </p>
                          )}
                          {step.error && (
                            <p className="mt-1 flex items-center gap-1 text-xs text-red-600">
                              <AlertTriangle className="h-3 w-3" />
                              {step.error}
                            </p>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Audit History</CardTitle>
              </CardHeader>
              <CardContent>
                {historyLoading ? (
                  <LoadingSkeleton variant="list-item" count={4} />
                ) : history.length === 0 ? (
                  <p className="py-4 text-center text-sm text-muted-foreground">No history entries yet</p>
                ) : (
                  <div className="space-y-4">
                    {history.map((entry) => {
                      const ActorIcon = ACTOR_ICONS[entry.actor_type] ?? User;
                      return (
                        <div key={entry.id} className="flex gap-3 rounded-lg border p-3">
                          <div className="mt-0.5 flex-shrink-0">
                            <ActorIcon className="h-4 w-4 text-muted-foreground" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center justify-between">
                              <p className="text-sm font-medium capitalize">{entry.action.replace(/_/g, ' ')}</p>
                              <span className="text-xs text-muted-foreground">
                                {new Date(entry.created_at).toLocaleString()}
                              </span>
                            </div>
                            <p className="mt-0.5 text-xs text-muted-foreground">
                              by {entry.actor_type.replace(/_/g, ' ')}
                              {entry.actor_id ? ` (${entry.actor_id.slice(0, 8)}...)` : ''}
                            </p>
                            {entry.details && Object.keys(entry.details).length > 0 && (
                              <div className="mt-2 rounded bg-muted/40 p-2 text-xs font-mono text-muted-foreground">
                                {Object.entries(entry.details).map(([key, val]) => (
                                  <div key={key}>
                                    <span className="font-medium">{key}:</span> {String(val)}
                                  </div>
                                ))}
                              </div>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </CardContent>
            </Card>

            {(remediation.compliance_tags ?? []).length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Compliance Tags</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-2">
                    {(remediation.compliance_tags ?? []).map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">{tag}</Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
