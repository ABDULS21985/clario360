'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  ArrowLeft,
  PlayCircle,
  CheckCircle,
  XCircle,
  RotateCcw,
  ClipboardList,
  AlertTriangle,
  Clock,
  User,
} from 'lucide-react';
import { timeAgo } from '@/lib/utils';
import { RemediationLifecycleBadge } from '../_components/remediation-lifecycle-badge';
import { RemediationApproveDialog } from '../_components/remediation-approve-dialog';
import { DryRunResultsPanel } from '../_components/dry-run-results-panel';
import { RollbackDialog } from '../_components/rollback-dialog';
import { useApiMutation } from '@/hooks/use-api-mutation';
import type { RemediationAction, RemediationAuditEntry } from '@/types/cyber';

interface Props {
  params: { id: string };
}

export default function RemediationDetailPage({ params }: Props) {
  const { id } = params;
  const router = useRouter();

  const [approveOpen, setApproveOpen] = useState(false);
  const [approveMode, setApproveMode] = useState<'approve' | 'reject'>('approve');
  const [rollbackOpen, setRollbackOpen] = useState(false);

  const { data: envelope, isLoading, error, refetch } = useQuery({
    queryKey: [`cyber-remediation-${id}`],
    queryFn: () => apiGet<{ data: RemediationAction }>(`${API_ENDPOINTS.CYBER_REMEDIATION}/${id}`),
    refetchInterval: 15000,
  });

  const { data: auditEnvelope } = useQuery({
    queryKey: [`cyber-remediation-audit-${id}`],
    queryFn: () => apiGet<{ data: RemediationAuditEntry[] }>(`${API_ENDPOINTS.CYBER_REMEDIATION}/${id}/audit-trail`),
  });

  const action = envelope?.data;
  const auditTrail = auditEnvelope?.data ?? [];

  const { mutate: runDryRun, isPending: dryRunning } = useApiMutation<unknown, Record<string, never>>(
    'post',
    `${API_ENDPOINTS.CYBER_REMEDIATION}/${id}/dry-run`,
    { successMessage: 'Dry run started', invalidateKeys: [`cyber-remediation-${id}`], onSuccess: () => void refetch() },
  );

  const { mutate: execute, isPending: executing } = useApiMutation<unknown, Record<string, never>>(
    'post',
    `${API_ENDPOINTS.CYBER_REMEDIATION}/${id}/execute`,
    { successMessage: 'Execution started', invalidateKeys: [`cyber-remediation-${id}`], onSuccess: () => void refetch() },
  );

  const canApprove = action?.status === 'pending_approval';
  const canDryRun = action?.status === 'approved' || action?.status === 'dry_run_failed';
  const canExecute = action?.status === 'dry_run_completed' || action?.status === 'approved';
  const canRollback = ['executed', 'verified', 'verification_failed'].includes(action?.status ?? '');

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {isLoading ? (
          <>
            <div className="h-8 w-64 animate-pulse rounded bg-muted" />
            <LoadingSkeleton variant="card" />
          </>
        ) : error || !action ? (
          <ErrorState message="Failed to load remediation action" onRetry={() => refetch()} />
        ) : (
          <>
            <PageHeader
              title={
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => router.back()}
                    className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </button>
                  <span className="truncate">{action.title}</span>
                </div>
              }
              description={
                <div className="flex items-center gap-3 pl-11">
                  <RemediationLifecycleBadge status={action.status} />
                  <SeverityIndicator severity={action.severity} showLabel />
                  <span className="capitalize text-xs text-muted-foreground">{action.type.replace(/_/g, ' ')}</span>
                </div>
              }
              actions={
                <div className="flex items-center gap-2">
                  {canApprove && (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        className="text-destructive"
                        onClick={() => { setApproveMode('reject'); setApproveOpen(true); }}
                      >
                        <XCircle className="mr-1.5 h-3.5 w-3.5" /> Reject
                      </Button>
                      <Button
                        size="sm"
                        className="bg-green-600 hover:bg-green-700"
                        onClick={() => { setApproveMode('approve'); setApproveOpen(true); }}
                      >
                        <CheckCircle className="mr-1.5 h-3.5 w-3.5" /> Approve
                      </Button>
                    </>
                  )}
                  {canDryRun && (
                    <Button variant="outline" size="sm" onClick={() => runDryRun({} as Record<string, never>)} disabled={dryRunning}>
                      {dryRunning ? 'Running…' : <><ClipboardList className="mr-1.5 h-3.5 w-3.5" /> Dry Run</>}
                    </Button>
                  )}
                  {canExecute && (
                    <Button size="sm" onClick={() => execute({} as Record<string, never>)} disabled={executing}>
                      {executing ? 'Executing…' : <><PlayCircle className="mr-1.5 h-3.5 w-3.5" /> Execute</>}
                    </Button>
                  )}
                  {canRollback && (
                    <Button variant="outline" size="sm" className="text-orange-600" onClick={() => setRollbackOpen(true)}>
                      <RotateCcw className="mr-1.5 h-3.5 w-3.5" /> Rollback
                    </Button>
                  )}
                </div>
              }
            />

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              {/* Left: main content */}
              <div className="space-y-6 lg:col-span-2">
                {/* Description */}
                <div className="rounded-xl border bg-card p-5">
                  <h3 className="mb-2 text-sm font-semibold">Description</h3>
                  <p className="text-sm leading-relaxed text-muted-foreground">{action.description}</p>
                </div>

                {/* Plan Steps */}
                <div className="rounded-xl border bg-card p-5">
                  <h3 className="mb-4 text-sm font-semibold">Execution Plan</h3>
                  <div className="space-y-3">
                    {action.plan.steps.map((step, idx) => {
                      const stepResult = action.execution_result?.step_results.find(
                        (r) => r.step_number === step.number,
                      );
                      const statusIcon = stepResult
                        ? stepResult.status === 'success'
                          ? <CheckCircle className="h-4 w-4 text-green-500" />
                          : stepResult.status === 'failure'
                          ? <XCircle className="h-4 w-4 text-red-500" />
                          : <Clock className="h-4 w-4 text-muted-foreground" />
                        : null;

                      return (
                        <div
                          key={idx}
                          className={`flex gap-4 rounded-lg border p-4 transition-colors ${
                            stepResult?.status === 'success'
                              ? 'border-green-200 bg-green-50/50 dark:border-green-800 dark:bg-green-950/20'
                              : stepResult?.status === 'failure'
                              ? 'border-red-200 bg-red-50/50 dark:border-red-800 dark:bg-red-950/20'
                              : 'bg-muted/20'
                          }`}
                        >
                          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-background border text-xs font-bold">
                            {statusIcon ?? step.number}
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium">{step.action}</p>
                            {step.target && (
                              <p className="mt-0.5 font-mono text-xs text-muted-foreground">Target: {step.target}</p>
                            )}
                            {step.description && (
                              <p className="mt-1 text-xs text-muted-foreground">{step.description}</p>
                            )}
                            {stepResult?.output && (
                              <pre className="mt-2 rounded bg-background p-2 text-xs overflow-x-auto">{stepResult.output}</pre>
                            )}
                            {stepResult?.error && (
                              <p className="mt-1 text-xs text-red-600">{stepResult.error}</p>
                            )}
                          </div>
                          {stepResult && (
                            <span className="text-xs text-muted-foreground">{stepResult.duration_ms}ms</span>
                          )}
                        </div>
                      );
                    })}
                  </div>

                  {action.plan.reversible !== undefined && (
                    <div className="mt-4 flex items-center gap-4 border-t pt-4 text-xs text-muted-foreground">
                      <span className={action.plan.reversible ? 'text-green-600' : 'text-orange-600'}>
                        {action.plan.reversible ? '✓ Reversible' : '✗ Irreversible'}
                      </span>
                      {action.plan.requires_reboot && <span className="text-amber-600">⚠ Requires reboot</span>}
                      {action.plan.risk_level && <span>Risk: <strong className="capitalize">{action.plan.risk_level}</strong></span>}
                      {action.plan.estimated_downtime && <span>Downtime: <strong>{action.plan.estimated_downtime}</strong></span>}
                    </div>
                  )}
                </div>

                {/* Dry Run Results */}
                {action.dry_run_result && (
                  <DryRunResultsPanel result={action.dry_run_result} />
                )}

                {/* Execution Result */}
                {action.execution_result && (
                  <div className="rounded-xl border bg-card p-5">
                    <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
                      Execution Result
                      {action.execution_result.success ? (
                        <Badge className="bg-green-100 text-green-700">Success</Badge>
                      ) : (
                        <Badge variant="destructive">Failed</Badge>
                      )}
                    </h3>
                    <div className="grid grid-cols-3 gap-4 text-center">
                      <div className="rounded-lg border bg-muted/20 p-3">
                        <p className="text-2xl font-bold">{action.execution_result.steps_executed}</p>
                        <p className="text-xs text-muted-foreground">Steps Executed</p>
                      </div>
                      <div className="rounded-lg border bg-muted/20 p-3">
                        <p className="text-2xl font-bold">{action.execution_result.changes_applied.length}</p>
                        <p className="text-xs text-muted-foreground">Changes Applied</p>
                      </div>
                      <div className="rounded-lg border bg-muted/20 p-3">
                        <p className="text-2xl font-bold">{(action.execution_result.duration_ms / 1000).toFixed(1)}s</p>
                        <p className="text-xs text-muted-foreground">Duration</p>
                      </div>
                    </div>
                    {action.execution_result.changes_applied.length > 0 && (
                      <div className="mt-4">
                        <p className="mb-2 text-xs font-semibold text-muted-foreground">Applied Changes</p>
                        <div className="space-y-2">
                          {action.execution_result.changes_applied.map((change, i) => (
                            <div key={i} className="rounded-lg border bg-muted/10 p-3">
                              <div className="flex items-center gap-2">
                                <span className="text-xs font-medium capitalize">{change.change_type.replace(/_/g, ' ')}</span>
                                <span className="text-xs text-muted-foreground">on {change.asset_id}</span>
                              </div>
                              <p className="mt-0.5 text-xs text-muted-foreground">{change.description}</p>
                              {change.old_value && change.new_value && (
                                <div className="mt-2 grid grid-cols-2 gap-2">
                                  <div className="rounded bg-red-50 p-2 text-xs text-red-700 dark:bg-red-950/30 dark:text-red-400">
                                    Before: {change.old_value}
                                  </div>
                                  <div className="rounded bg-green-50 p-2 text-xs text-green-700 dark:bg-green-950/30 dark:text-green-400">
                                    After: {change.new_value}
                                  </div>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {/* Verification Result */}
                {action.verification_result && (
                  <div className="rounded-xl border bg-card p-5">
                    <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
                      Verification
                      {action.verification_result.verified ? (
                        <Badge className="bg-green-100 text-green-700">Passed</Badge>
                      ) : (
                        <Badge variant="destructive">Failed</Badge>
                      )}
                    </h3>
                    <div className="space-y-2">
                      {action.verification_result.checks.map((check, i) => (
                        <div key={i} className={`flex items-start gap-3 rounded-lg border p-3 ${check.passed ? 'border-green-200 bg-green-50/30' : 'border-red-200 bg-red-50/30'}`}>
                          {check.passed ? (
                            <CheckCircle className="mt-0.5 h-4 w-4 shrink-0 text-green-500" />
                          ) : (
                            <XCircle className="mt-0.5 h-4 w-4 shrink-0 text-red-500" />
                          )}
                          <div>
                            <p className="text-sm font-medium">{check.name}</p>
                            <p className="text-xs text-muted-foreground">Expected: {check.expected}</p>
                            <p className="text-xs text-muted-foreground">Actual: {check.actual}</p>
                            {check.notes && <p className="mt-1 text-xs text-muted-foreground">{check.notes}</p>}
                          </div>
                        </div>
                      ))}
                    </div>
                    {action.verification_result.failure_reason && (
                      <p className="mt-3 text-sm text-red-600">{action.verification_result.failure_reason}</p>
                    )}
                  </div>
                )}

                {/* Audit Trail */}
                {auditTrail.length > 0 && (
                  <div className="rounded-xl border bg-card p-5">
                    <h3 className="mb-4 text-sm font-semibold">Audit Trail</h3>
                    <div className="space-y-3">
                      {auditTrail.map((entry) => (
                        <div key={entry.id} className="flex items-start gap-3 border-b pb-3 last:border-0 last:pb-0">
                          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted">
                            <User className="h-3.5 w-3.5 text-muted-foreground" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium">{entry.actor_name ?? 'System'}</span>
                              <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">{entry.action.replace(/_/g, ' ')}</span>
                            </div>
                            <p className="mt-0.5 text-xs text-muted-foreground">{timeAgo(entry.created_at)}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* Right: metadata sidebar */}
              <div className="space-y-4">
                <div className="rounded-xl border bg-card p-4">
                  <h3 className="mb-3 text-sm font-semibold">Details</h3>
                  <dl className="space-y-2.5 text-sm">
                    {[
                      { label: 'Status', value: <RemediationLifecycleBadge status={action.status} /> },
                      { label: 'Severity', value: <SeverityIndicator severity={action.severity} showLabel /> },
                      { label: 'Type', value: <span className="capitalize">{action.type.replace(/_/g, ' ')}</span> },
                      { label: 'Execution Mode', value: <span className="capitalize">{action.execution_mode.replace(/_/g, ' ')}</span> },
                      { label: 'Created By', value: action.created_by_name ?? '—' },
                      { label: 'Created', value: timeAgo(action.created_at) },
                    ].map(({ label, value }) => (
                      <div key={label} className="flex items-center justify-between gap-2 border-b pb-2 last:border-0 last:pb-0">
                        <dt className="text-muted-foreground">{label}</dt>
                        <dd className="text-right">{value}</dd>
                      </div>
                    ))}
                  </dl>
                </div>

                {action.approved_by && (
                  <div className="rounded-xl border bg-card p-4">
                    <h3 className="mb-3 text-sm font-semibold text-green-600">Approval</h3>
                    <div className="space-y-1.5 text-sm">
                      <p className="text-muted-foreground">Approved by <strong>{action.approved_by}</strong></p>
                      {action.approved_at && <p className="text-xs text-muted-foreground">{timeAgo(action.approved_at)}</p>}
                    </div>
                  </div>
                )}

                {action.rejected_by && (
                  <div className="rounded-xl border border-red-200 bg-red-50/30 p-4 dark:border-red-800 dark:bg-red-950/10">
                    <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-red-600">
                      <XCircle className="h-4 w-4" /> Rejected
                    </h3>
                    <p className="text-sm text-muted-foreground">By <strong>{action.rejected_by}</strong></p>
                    {action.rejected_at && <p className="text-xs text-muted-foreground mt-1">{timeAgo(action.rejected_at)}</p>}
                  </div>
                )}

                {action.rollback_deadline && (
                  <div className="rounded-xl border border-orange-200 bg-orange-50/30 p-4 dark:border-orange-800 dark:bg-orange-950/10">
                    <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-orange-600">
                      <AlertTriangle className="h-4 w-4" /> Rollback Window
                    </h3>
                    <p className="text-xs text-muted-foreground">Expires: {new Date(action.rollback_deadline).toLocaleString()}</p>
                    {action.rollback_reason && <p className="mt-1 text-xs text-muted-foreground">{action.rollback_reason}</p>}
                  </div>
                )}

                {/* Tags */}
                {action.tags.length > 0 && (
                  <div className="rounded-xl border bg-card p-4">
                    <h3 className="mb-2 text-sm font-semibold">Tags</h3>
                    <div className="flex flex-wrap gap-1.5">
                      {action.tags.map((tag) => (
                        <Badge key={tag} variant="secondary" className="text-xs">{tag}</Badge>
                      ))}
                    </div>
                  </div>
                )}

                {/* Linked items */}
                {(action.alert_id || action.vulnerability_id) && (
                  <div className="rounded-xl border bg-card p-4">
                    <h3 className="mb-3 text-sm font-semibold">Linked Items</h3>
                    <div className="space-y-2 text-sm">
                      {action.alert_id && (
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Alert</span>
                          <a href={`/cyber/alerts/${action.alert_id}`} className="font-mono text-xs text-primary hover:underline">
                            {action.alert_id.slice(0, 8)}…
                          </a>
                        </div>
                      )}
                      {action.vulnerability_id && (
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Vulnerability</span>
                          <span className="font-mono text-xs">{action.vulnerability_id.slice(0, 8)}…</span>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>

            <RemediationApproveDialog
              open={approveOpen}
              onOpenChange={setApproveOpen}
              action={action}
              mode={approveMode}
              onSuccess={() => void refetch()}
            />
            <RollbackDialog
              open={rollbackOpen}
              onOpenChange={setRollbackOpen}
              action={action}
              onSuccess={() => void refetch()}
            />
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
