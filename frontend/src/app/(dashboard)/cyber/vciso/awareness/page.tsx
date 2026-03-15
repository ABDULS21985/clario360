'use client';

import { useState, useMemo } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import {
  BookOpen,
  Eye,
  Edit,
  Plus,
  ShieldAlert,
  Users,
  Key,
  Lock,
  AlertTriangle,
  Wrench,
  CheckCircle,
} from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator, type Severity } from '@/components/shared/severity-indicator';
import { GaugeChart } from '@/components/shared/charts/gauge-chart';
import { PieChart } from '@/components/shared/charts/pie-chart';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { awarenessStatusConfig, iamFindingStatusConfig } from '@/lib/status-configs';
import { formatDate, truncate, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig, RowAction } from '@/types/table';
import type {
  VCISOAwarenessProgram,
  VCISOIAMFinding,
  VCISOIAMSummary,
} from '@/types/cyber';

import { AwarenessFormDialog } from './_components/awareness-form-dialog';
import { AwarenessDetailPanel } from './_components/awareness-detail-panel';
import { IAMFindingDetailPanel } from './_components/iam-finding-detail-panel';

// ── Constants ────────────────────────────────────────────────────────────────

const IAM_TYPE_LABELS: Record<string, string> = {
  mfa_gap: 'MFA Gaps',
  orphaned_account: 'Orphaned Accounts',
  privileged_access: 'Privileged Access',
  sod_violation: 'SoD Violations',
  stale_access: 'Stale Access',
  excessive_permissions: 'Excessive Permissions',
};

const IAM_TYPE_COLORS: Record<string, string> = {
  mfa_gap: '#ef4444',
  orphaned_account: '#f59e0b',
  privileged_access: '#8b5cf6',
  sod_violation: '#ec4899',
  stale_access: '#f97316',
  excessive_permissions: '#06b6d4',
};

const AWARENESS_TYPE_BADGE_CLASSES: Record<string, string> = {
  training: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  phishing_simulation: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300',
  policy_attestation: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
};

const AWARENESS_TYPE_LABELS: Record<string, string> = {
  training: 'Training',
  phishing_simulation: 'Phishing Simulation',
  policy_attestation: 'Policy Attestation',
};

const IAM_FINDING_TYPE_BADGE_CLASSES: Record<string, string> = {
  mfa_gap: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  orphaned_account: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300',
  privileged_access: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  sod_violation: 'bg-pink-100 text-pink-800 dark:bg-pink-900/30 dark:text-pink-300',
  stale_access: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
  excessive_permissions: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300',
};

// ── Filters ──────────────────────────────────────────────────────────────────

const AWARENESS_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'select',
    options: [
      { label: 'Training', value: 'training' },
      { label: 'Phishing Simulation', value: 'phishing_simulation' },
      { label: 'Policy Attestation', value: 'policy_attestation' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Scheduled', value: 'scheduled' },
      { label: 'Active', value: 'active' },
      { label: 'Completed', value: 'completed' },
    ],
  },
];

const IAM_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'select',
    options: [
      { label: 'MFA Gap', value: 'mfa_gap' },
      { label: 'Orphaned Account', value: 'orphaned_account' },
      { label: 'Privileged Access', value: 'privileged_access' },
      { label: 'SoD Violation', value: 'sod_violation' },
      { label: 'Stale Access', value: 'stale_access' },
      { label: 'Excessive Permissions', value: 'excessive_permissions' },
    ],
  },
  {
    key: 'severity',
    label: 'Severity',
    type: 'select',
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
      { label: 'Info', value: 'info' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Open', value: 'open' },
      { label: 'In Progress', value: 'in_progress' },
      { label: 'Resolved', value: 'resolved' },
      { label: 'Accepted', value: 'accepted' },
    ],
  },
];

// ── Helpers ──────────────────────────────────────────────────────────────────

function completionColor(rate: number): string {
  const pct = rate * 100;
  if (pct >= 80) return 'text-green-600';
  if (pct >= 60) return 'text-amber-600';
  return 'text-red-600';
}

function passRateColor(rate: number): string {
  const pct = rate * 100;
  if (pct >= 90) return 'text-green-600';
  if (pct >= 70) return 'text-amber-600';
  return 'text-red-600';
}

// ── Columns ──────────────────────────────────────────────────────────────────

function getAwarenessColumns(): ColumnDef<VCISOAwarenessProgram>[] {
  return [
    {
      accessorKey: 'name',
      header: 'Name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-foreground">{row.original.name}</span>
      ),
    },
    {
      accessorKey: 'type',
      header: 'Type',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge
          variant="secondary"
          className={cn(
            'text-xs',
            AWARENESS_TYPE_BADGE_CLASSES[row.original.type] ?? '',
          )}
        >
          {AWARENESS_TYPE_LABELS[row.original.type] ?? titleCase(row.original.type)}
        </Badge>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={awarenessStatusConfig} />
      ),
    },
    {
      accessorKey: 'total_users',
      header: 'Total Users',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.total_users.toLocaleString()}</span>
      ),
    },
    {
      accessorKey: 'completion_rate',
      header: 'Completion Rate',
      enableSorting: true,
      cell: ({ row }) => {
        const pct = Math.round(row.original.completion_rate * 100);
        return (
          <div className="flex items-center gap-2 min-w-[120px]">
            <div className="h-2 flex-1 rounded-full bg-muted overflow-hidden">
              <div
                className={cn(
                  'h-full rounded-full transition-all',
                  pct >= 80
                    ? 'bg-green-500'
                    : pct >= 60
                      ? 'bg-amber-500'
                      : 'bg-red-500',
                )}
                style={{ width: `${pct}%` }}
              />
            </div>
            <span className={cn('text-xs font-medium tabular-nums', completionColor(row.original.completion_rate))}>
              {pct}%
            </span>
          </div>
        );
      },
    },
    {
      accessorKey: 'pass_rate',
      header: 'Pass Rate',
      enableSorting: true,
      cell: ({ row }) => {
        const pct = Math.round(row.original.pass_rate * 100);
        return (
          <span className={cn('text-sm font-medium', passRateColor(row.original.pass_rate))}>
            {pct}%
          </span>
        );
      },
    },
    {
      accessorKey: 'start_date',
      header: 'Start Date',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(row.original.start_date)}
        </span>
      ),
    },
    {
      accessorKey: 'end_date',
      header: 'End Date',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(row.original.end_date)}
        </span>
      ),
    },
  ];
}

function getIAMFindingColumns(): ColumnDef<VCISOIAMFinding>[] {
  return [
    {
      accessorKey: 'title',
      header: 'Title',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-foreground">{row.original.title}</span>
      ),
    },
    {
      accessorKey: 'type',
      header: 'Type',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge
          variant="secondary"
          className={cn(
            'text-xs',
            IAM_FINDING_TYPE_BADGE_CLASSES[row.original.type] ?? '',
          )}
        >
          {IAM_TYPE_LABELS[row.original.type] ?? titleCase(row.original.type)}
        </Badge>
      ),
    },
    {
      accessorKey: 'severity',
      header: 'Severity',
      enableSorting: true,
      cell: ({ row }) => (
        <SeverityIndicator severity={row.original.severity as Severity} />
      ),
    },
    {
      accessorKey: 'affected_users',
      header: 'Affected Users',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.affected_users.toLocaleString()}</span>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={iamFindingStatusConfig} />
      ),
    },
    {
      accessorKey: 'remediation',
      header: 'Remediation',
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground max-w-[120px] sm:max-w-[200px] truncate block">
          {row.original.remediation ? truncate(row.original.remediation, 60) : '--'}
        </span>
      ),
    },
    {
      accessorKey: 'discovered_at',
      header: 'Discovered At',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(row.original.discovered_at)}
        </span>
      ),
    },
  ];
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function AwarenessIAMPage() {
  // Awareness state
  const [selectedProgram, setSelectedProgram] = useState<VCISOAwarenessProgram | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editProgram, setEditProgram] = useState<VCISOAwarenessProgram | null>(null);

  // IAM state
  const [selectedFinding, setSelectedFinding] = useState<VCISOIAMFinding | null>(null);

  // ── IAM Summary ────────────────────────────────────────────
  const {
    data: iamSummaryEnvelope,
    isLoading: iamSummaryLoading,
    error: iamSummaryError,
    mutate: refetchIAMSummary,
  } = useRealtimeData<{ data: VCISOIAMSummary }>(API_ENDPOINTS.CYBER_VCISO_IAM_SUMMARY, {
    wsTopics: ['vciso.iam'],
  });
  const iamSummary = iamSummaryEnvelope?.data;

  // ── Awareness Table ────────────────────────────────────────
  const {
    tableProps: awarenessTableProps,
    refetch: refetchAwareness,
  } = useDataTable<VCISOAwarenessProgram>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOAwarenessProgram>>(
        API_ENDPOINTS.CYBER_VCISO_AWARENESS,
        params as unknown as Record<string, unknown>,
      ),
    queryKey: 'vciso-awareness',
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['vciso.awareness'],
  });

  // ── IAM Findings Table ─────────────────────────────────────
  const {
    tableProps: iamTableProps,
    refetch: refetchIAM,
  } = useDataTable<VCISOIAMFinding>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOIAMFinding>>(
        API_ENDPOINTS.CYBER_VCISO_IAM_FINDINGS,
        params as unknown as Record<string, unknown>,
      ),
    queryKey: 'vciso-iam-findings',
    defaultSort: { column: 'discovered_at', direction: 'desc' },
    wsTopics: ['vciso.iam'],
  });

  // ── IAM Mutations ──────────────────────────────────────────
  const remediateMutation = useApiMutation<VCISOIAMFinding, Record<string, unknown>>(
    'put',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_IAM_FINDINGS}/${(variables as Record<string, string>).id}`,
    {
      successMessage: 'Finding marked as in progress for remediation',
      invalidateKeys: ['vciso-iam-findings', API_ENDPOINTS.CYBER_VCISO_IAM_SUMMARY],
      onSuccess: () => {
        refetchIAM();
        void refetchIAMSummary();
      },
    },
  );

  const acceptMutation = useApiMutation<VCISOIAMFinding, Record<string, unknown>>(
    'put',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_IAM_FINDINGS}/${(variables as Record<string, string>).id}`,
    {
      successMessage: 'Finding accepted',
      invalidateKeys: ['vciso-iam-findings', API_ENDPOINTS.CYBER_VCISO_IAM_SUMMARY],
      onSuccess: () => {
        refetchIAM();
        void refetchIAMSummary();
      },
    },
  );

  // ── Columns ────────────────────────────────────────────────
  const awarenessColumns = useMemo(() => getAwarenessColumns(), []);
  const iamColumns = useMemo(() => getIAMFindingColumns(), []);

  // ── Awareness Row Actions ──────────────────────────────────
  const awarenessRowActions: RowAction<VCISOAwarenessProgram>[] = [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (row) => setSelectedProgram(row),
    },
    {
      label: 'Edit',
      icon: Edit,
      onClick: (row) => {
        setEditProgram(row);
        setShowCreateDialog(true);
      },
    },
  ];

  // ── IAM Row Actions ────────────────────────────────────────
  const iamRowActions: RowAction<VCISOIAMFinding>[] = [
    {
      label: 'View',
      icon: Eye,
      onClick: (row) => setSelectedFinding(row),
    },
    {
      label: 'Remediate',
      icon: Wrench,
      onClick: (row) => remediateMutation.mutate({ id: row.id, status: 'in_progress' }),
      hidden: (row) => row.status === 'resolved' || row.status === 'in_progress',
    },
    {
      label: 'Accept',
      icon: CheckCircle,
      onClick: (row) => acceptMutation.mutate({ id: row.id, status: 'accepted' }),
      hidden: (row) => row.status === 'accepted' || row.status === 'resolved',
    },
  ];

  // ── IAM PieChart data ──────────────────────────────────────
  const iamPieData = useMemo(() => {
    if (!iamSummary?.by_type) return [];
    return Object.entries(iamSummary.by_type)
      .filter(([, count]) => count > 0)
      .map(([type, count]) => ({
        name: IAM_TYPE_LABELS[type] ?? titleCase(type),
        value: count,
        color: IAM_TYPE_COLORS[type] ?? '#94a3b8',
      }));
  }, [iamSummary]);

  const handleRefreshAll = () => {
    refetchAwareness();
    refetchIAM();
    void refetchIAMSummary();
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Awareness & IAM"
          description="Track security awareness programs and manage identity and access governance findings."
          actions={
            <Button onClick={() => { setEditProgram(null); setShowCreateDialog(true); }}>
              <Plus className="mr-2 h-4 w-4" />
              Create Program
            </Button>
          }
        />

        <Tabs defaultValue="awareness" className="space-y-4">
          <TabsList>
            <TabsTrigger value="awareness">Security Awareness</TabsTrigger>
            <TabsTrigger value="iam">Identity & Access Governance</TabsTrigger>
          </TabsList>

          {/* ── Security Awareness Tab ───────────────────────────── */}
          <TabsContent value="awareness" className="space-y-4">
            <DataTable
              columns={awarenessColumns}
              filters={AWARENESS_FILTERS}
              rowActions={awarenessRowActions}
              searchPlaceholder="Search awareness programs..."
              emptyState={{
                icon: BookOpen,
                title: 'No awareness programs',
                description: 'Create a new security awareness program to get started.',
                action: {
                  label: 'Create Program',
                  onClick: () => { setEditProgram(null); setShowCreateDialog(true); },
                  icon: Plus,
                },
              }}
              onRowClick={(row) => setSelectedProgram(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...awarenessTableProps}
            />
          </TabsContent>

          {/* ── Identity & Access Governance Tab ─────────────────── */}
          <TabsContent value="iam" className="space-y-6">
            {/* KPI Row */}
            {iamSummaryError ? (
              <ErrorState
                message="Failed to load IAM summary data"
                onRetry={() => void refetchIAMSummary()}
              />
            ) : iamSummaryLoading ? (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                {[1, 2, 3, 4].map((i) => (
                  <LoadingSkeleton key={i} variant="card" />
                ))}
              </div>
            ) : (
              <>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                  {/* MFA Coverage Gauge */}
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-semibold flex items-center gap-2">
                        <Lock className="h-4 w-4 text-muted-foreground" />
                        MFA Coverage
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="flex justify-center">
                      <GaugeChart
                        value={iamSummary?.mfa_coverage_percent ?? 0}
                        max={100}
                        thresholds={{ good: 80, warning: 60 }}
                        label="Coverage"
                        size={160}
                        format="percentage"
                      />
                    </CardContent>
                  </Card>

                  <KpiCard
                    title="Privileged Accounts"
                    value={iamSummary?.privileged_accounts ?? 0}
                    icon={Key}
                    iconColor="text-purple-600"
                    description="Accounts with elevated access"
                  />

                  <KpiCard
                    title="Orphaned Accounts"
                    value={iamSummary?.orphaned_accounts ?? 0}
                    icon={Users}
                    iconColor="text-amber-600"
                    description="Accounts without active owners"
                    className={
                      (iamSummary?.orphaned_accounts ?? 0) > 0
                        ? 'border-amber-200'
                        : ''
                    }
                  />

                  <KpiCard
                    title="Stale Access"
                    value={iamSummary?.stale_access_count ?? 0}
                    icon={AlertTriangle}
                    iconColor="text-red-600"
                    description="Unused access permissions"
                    className={
                      (iamSummary?.stale_access_count ?? 0) > 0
                        ? 'border-red-200'
                        : ''
                    }
                  />
                </div>

                {/* Findings by Type Chart */}
                {iamPieData.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base flex items-center gap-2">
                        <ShieldAlert className="h-5 w-5 text-muted-foreground" />
                        Findings by Type
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <PieChart
                        data={iamPieData}
                        innerRadius={50}
                        outerRadius={90}
                        height={240}
                        showLegend
                        centerValue={String(iamSummary?.total_findings ?? 0)}
                        centerLabel="Total"
                      />
                    </CardContent>
                  </Card>
                )}
              </>
            )}

            {/* IAM Findings Table */}
            <DataTable
              columns={iamColumns}
              filters={IAM_FILTERS}
              rowActions={iamRowActions}
              searchPlaceholder="Search IAM findings..."
              emptyState={{
                icon: ShieldAlert,
                title: 'No IAM findings',
                description: 'No identity and access management findings have been detected.',
              }}
              onRowClick={(row) => setSelectedFinding(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...iamTableProps}
            />
          </TabsContent>
        </Tabs>
      </div>

      {/* ── Awareness Detail Panel ───────────────────────────── */}
      {selectedProgram && (
        <AwarenessDetailPanel
          open={!!selectedProgram}
          onOpenChange={(o) => {
            if (!o) setSelectedProgram(null);
          }}
          program={selectedProgram}
        />
      )}

      {/* ── Create/Edit Program Dialog ───────────────────────── */}
      <AwarenessFormDialog
        open={showCreateDialog}
        onOpenChange={(o) => {
          setShowCreateDialog(o);
          if (!o) setEditProgram(null);
        }}
        onCreated={handleRefreshAll}
        program={editProgram}
      />

      {/* ── IAM Finding Detail Panel ─────────────────────────── */}
      {selectedFinding && (
        <IAMFindingDetailPanel
          open={!!selectedFinding}
          onOpenChange={(o) => {
            if (!o) setSelectedFinding(null);
          }}
          finding={selectedFinding}
        />
      )}
    </PermissionRedirect>
  );
}
