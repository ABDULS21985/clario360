'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, Wrench, CheckCircle, Clock, PlayCircle, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { getRemediationColumns } from './_components/remediation-columns';
import { RemediationCreateDialog } from './_components/remediation-create-dialog';
import { RemediationApproveDialog } from './_components/remediation-approve-dialog';
import type { RemediationAction, RemediationStats } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';

const STATUS_FILTERS = [
  'draft', 'pending_approval', 'approved', 'rejected', 'dry_run_running',
  'dry_run_completed', 'executing', 'executed', 'verified', 'closed',
];

export default function CyberRemediationPage() {
  const router = useRouter();
  const [createOpen, setCreateOpen] = useState(false);
  const [approveAction, setApproveAction] = useState<RemediationAction | null>(null);
  const [approveMode, setApproveMode] = useState<'approve' | 'reject'>('approve');

  const { data: statsEnvelope, mutate: refetchStats } = useRealtimeData<{ data: RemediationStats }>(
    API_ENDPOINTS.CYBER_REMEDIATION_STATS,
    { pollInterval: 60000 },
  );
  const stats = statsEnvelope?.data;

  const { tableProps, refetch } = useDataTable<RemediationAction>({
    queryKey: 'cyber-remediation',
    fetchFn: (params: FetchParams) =>
      apiGet<PaginatedResponse<RemediationAction>>(API_ENDPOINTS.CYBER_REMEDIATION, params as unknown as Record<string, unknown>),
    wsTopics: ['remediation.created', 'remediation.status_changed'],
    defaultSort: { column: 'created_at', direction: 'desc' },
  });

  const kpis = [
    {
      label: 'Pending Approval',
      value: stats?.pending_approval ?? 0,
      icon: Clock,
      color: 'text-amber-600',
      bg: 'bg-amber-50 dark:bg-amber-950/20',
    },
    {
      label: 'Execution Pending',
      value: stats?.execution_pending ?? 0,
      icon: PlayCircle,
      color: 'text-blue-600',
      bg: 'bg-blue-50 dark:bg-blue-950/20',
    },
    {
      label: 'Total Actions',
      value: stats?.total ?? 0,
      icon: Wrench,
      color: 'text-muted-foreground',
      bg: 'bg-muted/30',
    },
    {
      label: 'Verified & Closed',
      value: (stats?.by_status?.['verified'] ?? 0) + (stats?.by_status?.['closed'] ?? 0),
      icon: CheckCircle,
      color: 'text-green-600',
      bg: 'bg-green-50 dark:bg-green-950/20',
    },
  ];

  const columns = getRemediationColumns({
    onApprove: (action) => {
      setApproveAction(action);
      setApproveMode('approve');
    },
    onExecute: (action) => {
      router.push(`/cyber/remediation/${action.id}`);
    },
  });

  const filters = [
    {
      id: 'status',
      label: 'Status',
      type: 'multi-select' as const,
      options: STATUS_FILTERS.map((s) => ({ label: s.replace(/_/g, ' '), value: s })),
    },
    {
      id: 'severity',
      label: 'Severity',
      type: 'multi-select' as const,
      options: ['critical', 'high', 'medium', 'low'].map((s) => ({ label: s, value: s })),
    },
    {
      id: 'type',
      label: 'Type',
      type: 'multi-select' as const,
      options: ['patch', 'config_change', 'firewall_rule', 'user_action', 'script', 'manual'].map((t) => ({
        label: t.replace(/_/g, ' '),
        value: t,
      })),
    },
  ];

  const handleSuccess = () => {
    refetch();
    void refetchStats();
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Remediation"
          description="Track and orchestrate security remediation actions through their full lifecycle"
          actions={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              New Action
            </Button>
          }
        />

        {/* KPI Summary */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {kpis.map(({ label, value, icon: Icon, color, bg }) => (
            <div key={label} className={`flex items-center gap-3 rounded-xl border p-4 ${bg}`}>
              <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border bg-background ${color}`}>
                <Icon className="h-5 w-5" />
              </div>
              <div>
                <p className={`text-2xl font-bold tabular-nums ${color}`}>{value}</p>
                <p className="text-xs text-muted-foreground">{label}</p>
              </div>
            </div>
          ))}
        </div>

        {/* Severity breakdown bar */}
        {stats && (
          <div className="rounded-xl border bg-card p-4">
            <p className="mb-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground">By Severity</p>
            <div className="flex items-center gap-4">
              {(['critical', 'high', 'medium', 'low'] as const).map((sev) => {
                const count = stats.by_severity?.[sev] ?? 0;
                const colors: Record<string, string> = {
                  critical: 'bg-red-500 text-red-700',
                  high: 'bg-orange-500 text-orange-700',
                  medium: 'bg-amber-500 text-amber-700',
                  low: 'bg-blue-400 text-blue-700',
                };
                return (
                  <div key={sev} className="flex items-center gap-2">
                    <span className={`h-2.5 w-2.5 rounded-full ${colors[sev]?.split(' ')[0]}`} />
                    <span className="capitalize text-sm font-medium">{sev}</span>
                    <span className={`text-sm font-bold ${colors[sev]?.split(' ')[1]}`}>{count}</span>
                  </div>
                );
              })}
              {(stats.by_status?.['rollback_pending'] ?? 0) + (stats.by_status?.['rollback_failed'] ?? 0) > 0 && (
                <div className="ml-auto flex items-center gap-1.5 rounded-full bg-orange-100 px-3 py-1 text-xs font-medium text-orange-700">
                  <AlertTriangle className="h-3.5 w-3.5" />
                  {(stats.by_status?.['rollback_pending'] ?? 0) + (stats.by_status?.['rollback_failed'] ?? 0)} rollback issue(s)
                </div>
              )}
            </div>
          </div>
        )}

        {/* Table */}
        {tableProps.isLoading ? (
          <LoadingSkeleton variant="table-row" count={8} />
        ) : tableProps.error ? (
          <ErrorState message="Failed to load remediation actions" onRetry={refetch} />
        ) : (
          <DataTable
            {...tableProps}
            columns={columns}
            filters={filters}
            onSortChange={() => undefined}
            searchPlaceholder="Search remediation actions…"
            emptyState={{
              icon: Wrench,
              title: 'No remediation actions',
              description: 'Create your first remediation action to start tracking security fixes.',
              action: { label: 'New Action', onClick: () => setCreateOpen(true) },
            }}
          />
        )}

        <RemediationCreateDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          onSuccess={handleSuccess}
        />

        {approveAction && (
          <RemediationApproveDialog
            open={!!approveAction}
            onOpenChange={(o) => { if (!o) setApproveAction(null); }}
            action={approveAction}
            mode={approveMode}
            onSuccess={handleSuccess}
          />
        )}
      </div>
    </PermissionRedirect>
  );
}
