'use client';

import { useRouter } from 'next/navigation';
import { type ColumnDef, type Row } from '@tanstack/react-table';
import {
  AlertTriangle,
  CheckCircle2,
  Clock,
  Flame,
  Loader2,
  ShieldAlert,
  Timer,
  Wrench,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { DSPMRemediation, DSPMRemediationStats, CyberSeverity, DSPMRemediationStatus, DSPMFindingType } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';

const SEVERITY_COLORS: Record<CyberSeverity, string> = {
  critical: 'bg-red-100 text-red-700',
  high: 'bg-orange-100 text-orange-700',
  medium: 'bg-amber-100 text-amber-800',
  low: 'bg-blue-100 text-blue-700',
  info: 'bg-gray-100 text-gray-600',
};

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

function formatTimeRemaining(slaDueAt: string | undefined, slaBreached: boolean): string {
  if (slaBreached) return 'Breached';
  if (!slaDueAt) return '--';
  const now = new Date();
  const due = new Date(slaDueAt);
  const diffMs = due.getTime() - now.getTime();
  if (diffMs <= 0) return 'Breached';
  const hours = Math.floor(diffMs / (1000 * 60 * 60));
  if (hours >= 24) {
    const days = Math.floor(hours / 24);
    return `${days}d ${hours % 24}h`;
  }
  return `${hours}h`;
}

const remediationColumns: ColumnDef<DSPMRemediation>[] = [
  {
    id: 'title',
    accessorKey: 'title',
    header: 'Title',
    cell: ({ row }: { row: Row<DSPMRemediation> }) => {
      const r = row.original;
      return (
        <div>
          <p className="text-sm font-medium">{r.title}</p>
          <Badge variant="outline" className="mt-0.5 text-xs capitalize">
            {r.finding_type.replace(/_/g, ' ')}
          </Badge>
        </div>
      );
    },
    enableSorting: true,
  },
  {
    id: 'severity',
    accessorKey: 'severity',
    header: 'Severity',
    cell: ({ row }: { row: Row<DSPMRemediation> }) => {
      const sev = row.original.severity;
      return (
        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[sev] ?? 'bg-muted text-muted-foreground'}`}>
          {sev}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: 'data_asset_name',
    accessorKey: 'data_asset_name',
    header: 'Asset',
    cell: ({ row }: { row: Row<DSPMRemediation> }) => (
      <span className="text-sm">{row.original.data_asset_name ?? '--'}</span>
    ),
    enableSorting: true,
  },
  {
    id: 'assigned_to',
    accessorKey: 'assigned_to',
    header: 'Assignee',
    cell: ({ row }: { row: Row<DSPMRemediation> }) => (
      <span className="text-sm text-muted-foreground">{row.original.assigned_to ?? 'Unassigned'}</span>
    ),
  },
  {
    id: 'status',
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }: { row: Row<DSPMRemediation> }) => {
      const status = row.original.status;
      return (
        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${STATUS_COLORS[status] ?? 'bg-muted text-muted-foreground'}`}>
          {status.replace(/_/g, ' ')}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: 'sla',
    header: 'SLA',
    cell: ({ row }: { row: Row<DSPMRemediation> }) => {
      const r = row.original;
      const display = formatTimeRemaining(r.sla_due_at, r.sla_breached);
      const isBreached = display === 'Breached';
      return (
        <span className={`text-xs font-medium ${isBreached ? 'text-red-600' : 'text-muted-foreground'}`}>
          {isBreached ? (
            <span className="inline-flex items-center gap-1">
              <AlertTriangle className="h-3 w-3" />
              Breached
            </span>
          ) : display}
        </span>
      );
    },
  },
  {
    id: 'steps_progress',
    header: 'Steps',
    cell: ({ row }: { row: Row<DSPMRemediation> }) => {
      const r = row.original;
      const progress = r.total_steps > 0 ? Math.round((r.current_step / r.total_steps) * 100) : 0;
      return (
        <div className="flex items-center gap-2">
          <Progress value={progress} className="h-1.5 w-16" />
          <span className="text-xs tabular-nums text-muted-foreground">
            {r.current_step}/{r.total_steps}
          </span>
        </div>
      );
    },
  },
];

export default function RemediationsPage() {
  const router = useRouter();

  const {
    data: statsEnvelope,
    isLoading: statsLoading,
    error: statsError,
    mutate: refetchStats,
  } = useRealtimeData<{ data: DSPMRemediationStats }>(API_ENDPOINTS.CYBER_DSPM_REMEDIATION_STATS, {
    pollInterval: 60000,
  });

  const { tableProps, refetch } = useDataTable<DSPMRemediation>({
    queryKey: 'cyber-dspm-remediations',
    fetchFn: (params: FetchParams) => {
      const { filters, ...rest } = params;
      return apiGet<PaginatedResponse<DSPMRemediation>>(API_ENDPOINTS.CYBER_DSPM_REMEDIATIONS, { ...rest, ...filters } as Record<string, unknown>);
    },
    defaultSort: { column: 'severity', direction: 'desc' },
  });

  const stats = statsEnvelope?.data;

  const kpis = [
    { label: 'Open Remediations', value: stats?.total_open ?? 0, icon: Wrench, color: 'text-blue-600' },
    { label: 'Critical Open', value: stats?.total_critical_open ?? 0, icon: Flame, color: 'text-red-600' },
    { label: 'In Progress', value: stats?.total_in_progress ?? 0, icon: Loader2, color: 'text-amber-600' },
    { label: 'Completed (7d)', value: stats?.completed_last_7_days ?? 0, icon: CheckCircle2, color: 'text-green-600' },
    { label: 'SLA Breaches', value: stats?.sla_breaches ?? 0, icon: AlertTriangle, color: 'text-red-600' },
    { label: 'Avg Resolution', value: `${(stats?.avg_resolution_hours ?? 0).toFixed(1)}h`, icon: Timer, color: 'text-purple-600' },
  ];

  const filters = [
    {
      key: 'status',
      label: 'Status',
      type: 'multi-select' as const,
      options: ['open', 'in_progress', 'awaiting_approval', 'completed', 'failed', 'cancelled', 'rolled_back', 'exception_granted'].map((s) => ({
        label: s.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase()),
        value: s,
      })),
    },
    {
      key: 'severity',
      label: 'Severity',
      type: 'multi-select' as const,
      options: ['critical', 'high', 'medium', 'low'].map((s) => ({
        label: s.charAt(0).toUpperCase() + s.slice(1),
        value: s,
      })),
    },
    {
      key: 'finding_type',
      label: 'Finding Type',
      type: 'multi-select' as const,
      options: [
        'posture_gap', 'overprivileged_access', 'stale_access', 'classification_drift',
        'shadow_copy', 'policy_violation', 'encryption_missing', 'exposure_risk',
        'pii_unprotected', 'retention_expired', 'blast_radius_excessive',
      ].map((t) => ({
        label: t.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase()),
        value: t,
      })),
    },
  ];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Remediations"
          description="Track and manage automated remediation workflows for data security findings"
        />

        {statsLoading ? (
          <LoadingSkeleton variant="card" count={3} />
        ) : statsError ? (
          <ErrorState message="Failed to load remediation stats" onRetry={() => void refetchStats()} />
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
            {kpis.map((kpi) => {
              const Icon = kpi.icon;
              return (
                <Card key={kpi.label}>
                  <CardContent className="flex flex-col items-center p-4 text-center">
                    <Icon className={`mb-2 h-5 w-5 ${kpi.color}`} />
                    <p className="text-xs text-muted-foreground">{kpi.label}</p>
                    <p className="text-xl font-bold tabular-nums">
                      {typeof kpi.value === 'number' ? kpi.value.toLocaleString() : kpi.value}
                    </p>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}

        {stats && (
          <Card>
            <CardContent className="p-5">
              <h3 className="mb-4 text-sm font-semibold">Risk Reduction Summary</h3>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="rounded-lg border bg-muted/20 p-4 text-center">
                  <p className="text-xs text-muted-foreground">Total Risk Reduction</p>
                  <p className="text-2xl font-bold tabular-nums text-green-600">{stats.total_risk_reduction.toFixed(1)}</p>
                </div>
                <div className="rounded-lg border bg-muted/20 p-4">
                  <p className="mb-2 text-xs font-medium text-muted-foreground">By Severity</p>
                  <div className="space-y-1">
                    {Object.entries(stats.by_severity ?? {}).map(([sev, count]) => (
                      <div key={sev} className="flex items-center justify-between text-xs">
                        <span className="capitalize">{sev}</span>
                        <span className="font-medium tabular-nums">{count}</span>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="rounded-lg border bg-muted/20 p-4">
                  <p className="mb-2 text-xs font-medium text-muted-foreground">By Status</p>
                  <div className="space-y-1">
                    {Object.entries(stats.by_status ?? {}).map(([status, count]) => (
                      <div key={status} className="flex items-center justify-between text-xs">
                        <span className="capitalize">{status.replace(/_/g, ' ')}</span>
                        <span className="font-medium tabular-nums">{count}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        <div className="rounded-xl border bg-card">
          <div className="border-b px-5 py-4">
            <h3 className="text-sm font-semibold">Remediation Queue</h3>
            <p className="text-xs text-muted-foreground">Active and recent remediation workflows</p>
          </div>
          <div className="p-5">
            {tableProps.isLoading ? (
              <LoadingSkeleton variant="table-row" count={6} />
            ) : tableProps.error ? (
              <ErrorState message="Failed to load remediations" onRetry={refetch} />
            ) : (
              <DataTable
                {...tableProps}
                columns={remediationColumns}
                filters={filters}
                onSortChange={() => undefined}
                searchPlaceholder="Search remediations..."
                onRowClick={(row) => router.push(`/cyber/dspm/remediations/${row.id}`)}
                emptyState={{
                  icon: ShieldAlert,
                  title: 'No remediations found',
                  description: 'No remediation workflows have been created yet.',
                }}
              />
            )}
          </div>
        </div>
      </div>
    </PermissionRedirect>
  );
}
