'use client';

import { useState, useMemo } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import {
  ShieldAlert,
  Plus,
  Eye,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Clock,
  Target,
  TrendingDown,
  Building2,
  BarChart3,
} from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs';
import { Separator } from '@/components/ui/separator';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { buildSuiteQueryParams } from '@/lib/suite-api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate } from '@/lib/format';
import { cn } from '@/lib/utils';
import { riskStatusConfig, riskTreatmentConfig } from '@/lib/status-configs';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig, RowAction } from '@/types/table';
import type {
  VCISORiskEntry,
  VCISORiskStats,
} from '@/types/cyber';

import { RiskDetailPanel } from './_components/risk-detail-panel';
import { RiskFormDialog } from './_components/risk-form-dialog';
import { RiskAcceptanceDialog } from './_components/risk-acceptance-dialog';

// ── Helpers ──────────────────────────────────────────────────────────────────

function titleCase(str: string): string {
  return str
    .replace(/[-_]/g, ' ')
    .split(' ')
    .map((w) => (w.length > 0 ? w.charAt(0).toUpperCase() + w.slice(1).toLowerCase() : w))
    .join(' ');
}

function residualScoreColor(score: number): string {
  if (score <= 30) return 'bg-green-100 text-green-700';
  if (score <= 60) return 'bg-amber-100 text-amber-700';
  return 'bg-red-100 text-red-700';
}

// ── Filters ──────────────────────────────────────────────────────────────────

const RISK_FILTERS: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Open', value: 'open' },
      { label: 'Mitigated', value: 'mitigated' },
      { label: 'Accepted', value: 'accepted' },
      { label: 'Closed', value: 'closed' },
    ],
  },
  {
    key: 'treatment',
    label: 'Treatment',
    type: 'select',
    options: [
      { label: 'Mitigate', value: 'mitigate' },
      { label: 'Transfer', value: 'transfer' },
      { label: 'Accept', value: 'accept' },
      { label: 'Avoid', value: 'avoid' },
    ],
  },
  {
    key: 'likelihood',
    label: 'Likelihood',
    type: 'select',
    options: [
      { label: 'Low', value: 'low' },
      { label: 'Medium', value: 'medium' },
      { label: 'High', value: 'high' },
      { label: 'Critical', value: 'critical' },
    ],
  },
  {
    key: 'impact',
    label: 'Impact',
    type: 'select',
    options: [
      { label: 'Low', value: 'low' },
      { label: 'Medium', value: 'medium' },
      { label: 'High', value: 'high' },
      { label: 'Critical', value: 'critical' },
    ],
  },
];

// ── Columns ──────────────────────────────────────────────────────────────────

function getRiskColumns(): ColumnDef<VCISORiskEntry>[] {
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
      accessorKey: 'category',
      header: 'Category',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline" className="text-xs">
          {row.original.category}
        </Badge>
      ),
    },
    {
      accessorKey: 'likelihood',
      header: 'Likelihood',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{titleCase(row.original.likelihood)}</span>
      ),
    },
    {
      accessorKey: 'impact',
      header: 'Impact',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{titleCase(row.original.impact)}</span>
      ),
    },
    {
      accessorKey: 'inherent_score',
      header: 'Inherent',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm font-medium">{row.original.inherent_score}</span>
      ),
    },
    {
      accessorKey: 'residual_score',
      header: 'Residual',
      enableSorting: true,
      cell: ({ row }) => (
        <span
          className={cn(
            'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold',
            residualScoreColor(row.original.residual_score),
          )}
        >
          {row.original.residual_score}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={riskStatusConfig} />
      ),
    },
    {
      accessorKey: 'treatment',
      header: 'Treatment',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.treatment} config={riskTreatmentConfig} />
      ),
    },
    {
      accessorKey: 'owner_name',
      header: 'Owner',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.owner_name || 'Unassigned'}</span>
      ),
    },
    {
      accessorKey: 'review_date',
      header: 'Review Date',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.review_date ? formatDate(row.original.review_date) : '--'}
        </span>
      ),
    },
  ];
}

function getAcceptanceColumns(): ColumnDef<VCISORiskEntry>[] {
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
      accessorKey: 'category',
      header: 'Category',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline" className="text-xs">
          {row.original.category}
        </Badge>
      ),
    },
    {
      accessorKey: 'residual_score',
      header: 'Residual Score',
      enableSorting: true,
      cell: ({ row }) => (
        <span
          className={cn(
            'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold',
            residualScoreColor(row.original.residual_score),
          )}
        >
          {row.original.residual_score}
        </span>
      ),
    },
    {
      accessorKey: 'acceptance_rationale',
      header: 'Rationale',
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground max-w-[160px] sm:max-w-[250px] truncate block">
          {row.original.acceptance_rationale || '--'}
        </span>
      ),
    },
    {
      accessorKey: 'acceptance_approved_by_name',
      header: 'Approved By',
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.acceptance_approved_by_name || '--'}
        </span>
      ),
    },
    {
      accessorKey: 'acceptance_expiry',
      header: 'Expiry',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.acceptance_expiry
            ? formatDate(row.original.acceptance_expiry)
            : 'No expiry'}
        </span>
      ),
    },
    {
      accessorKey: 'owner_name',
      header: 'Owner',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.owner_name || 'Unassigned'}</span>
      ),
    },
  ];
}

// ── Likelihood / Impact score mapping for heat matrix ────────────────────────

const LIKELIHOOD_LABELS = ['Low', 'Medium', 'High', 'Critical'];
const IMPACT_LABELS = ['Low', 'Medium', 'High', 'Critical'];
const LIKELIHOOD_VALUES = ['low', 'medium', 'high', 'critical'];
const IMPACT_VALUES = ['low', 'medium', 'high', 'critical'];

function getHeatColor(likelihoodIdx: number, impactIdx: number): string {
  const score = (likelihoodIdx + 1) * (impactIdx + 1);
  if (score <= 4) return 'bg-green-100 text-green-800 border-green-200';
  if (score <= 9) return 'bg-amber-100 text-amber-800 border-amber-200';
  if (score <= 16) return 'bg-orange-100 text-orange-800 border-orange-200';
  return 'bg-red-100 text-red-800 border-red-200';
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function RiskRegisterPage() {
  const [selectedRisk, setSelectedRisk] = useState<VCISORiskEntry | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [acceptTarget, setAcceptTarget] = useState<VCISORiskEntry | null>(null);
  const [closeTarget, setCloseTarget] = useState<VCISORiskEntry | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<VCISORiskEntry | null>(null);

  // ── Stats ───────────────────────────────────────────────
  const {
    data: statsEnvelope,
    isLoading: statsLoading,
  } = useRealtimeData<{ data: VCISORiskStats }>(API_ENDPOINTS.CYBER_VCISO_RISKS_STATS, {
    wsTopics: ['vciso.risks'],
  });
  const stats = statsEnvelope?.data;

  // ── Risk Register Table ─────────────────────────────────
  const {
    tableProps,
    refetch,
  } = useDataTable<VCISORiskEntry>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISORiskEntry>>(
        API_ENDPOINTS.CYBER_VCISO_RISKS,
        buildSuiteQueryParams(params),
      ),
    queryKey: 'vciso-risks',
    defaultSort: { column: 'residual_score', direction: 'desc' },
    wsTopics: ['vciso.risks'],
  });

  // ── Acceptance Table (filtered by status=accepted) ──────
  const {
    tableProps: acceptanceTableProps,
    refetch: refetchAcceptance,
  } = useDataTable<VCISORiskEntry>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISORiskEntry>>(
        API_ENDPOINTS.CYBER_VCISO_RISKS,
        {
          ...buildSuiteQueryParams(params),
          status: 'accepted',
        },
      ),
    queryKey: 'vciso-risks-accepted',
    defaultSort: { column: 'updated_at', direction: 'desc' },
    wsTopics: ['vciso.risks'],
  });

  // ── Business Impact data ────────────────────────────────
  const {
    data: allRisksData,
    isLoading: allRisksLoading,
    error: allRisksError,
    mutate: refetchAllRisks,
  } = useRealtimeData<PaginatedResponse<VCISORiskEntry>>(API_ENDPOINTS.CYBER_VCISO_RISKS, {
    params: { per_page: 500 },
    wsTopics: ['vciso.risks'],
  });

  // ── Close risk mutation ─────────────────────────────────
  const closeMutation = useApiMutation<VCISORiskEntry, Record<string, unknown>>(
    'put',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_RISKS}/${(variables as Record<string, string>).id}`,
    {
      successMessage: 'Risk closed successfully',
      invalidateKeys: ['vciso-risks', 'vciso-risks-accepted', API_ENDPOINTS.CYBER_VCISO_RISKS_STATS],
      onSuccess: () => {
        setCloseTarget(null);
        refetch();
      },
    },
  );

  // ── Revoke acceptance mutation ──────────────────────────
  const revokeMutation = useApiMutation<VCISORiskEntry, Record<string, unknown>>(
    'put',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_RISKS}/${(variables as Record<string, string>).id}`,
    {
      successMessage: 'Risk acceptance revoked',
      invalidateKeys: ['vciso-risks', 'vciso-risks-accepted', API_ENDPOINTS.CYBER_VCISO_RISKS_STATS],
      onSuccess: () => {
        setRevokeTarget(null);
        refetch();
        refetchAcceptance();
      },
    },
  );

  // ── Columns ─────────────────────────────────────────────
  const riskColumns = useMemo(() => getRiskColumns(), []);
  const acceptanceColumns = useMemo(() => getAcceptanceColumns(), []);

  // ── Row actions ─────────────────────────────────────────
  const riskRowActions: RowAction<VCISORiskEntry>[] = [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (row) => setSelectedRisk(row),
    },
    {
      label: 'Accept Risk',
      icon: CheckCircle,
      onClick: (row) => setAcceptTarget(row),
      hidden: (row) => row.status === 'accepted' || row.status === 'closed',
    },
    {
      label: 'Close Risk',
      icon: XCircle,
      variant: 'destructive',
      onClick: (row) => setCloseTarget(row),
      hidden: (row) => row.status === 'closed',
    },
  ];

  const acceptanceRowActions: RowAction<VCISORiskEntry>[] = [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (row) => setSelectedRisk(row),
    },
    {
      label: 'Revoke Acceptance',
      icon: XCircle,
      variant: 'destructive',
      onClick: (row) => setRevokeTarget(row),
    },
  ];

  // ── Business Impact: group by department ────────────────
  const departmentGroups = useMemo(() => {
    const risks = allRisksData?.data ?? [];
    const groups = new Map<string, VCISORiskEntry[]>();
    for (const risk of risks) {
      const dept = risk.department || 'Unassigned';
      const existing = groups.get(dept);
      if (existing) {
        existing.push(risk);
      } else {
        groups.set(dept, [risk]);
      }
    }
    return Array.from(groups.entries())
      .sort(([, a], [, b]) => b.length - a.length);
  }, [allRisksData]);

  // ── Heat matrix data ───────────────────────────────────
  const heatMatrix = useMemo(() => {
    const risks = allRisksData?.data ?? [];
    const matrix: number[][] = Array.from({ length: 5 }, () => Array(5).fill(0) as number[]);
    for (const risk of risks) {
      const li = LIKELIHOOD_VALUES.indexOf(risk.likelihood);
      const ii = IMPACT_VALUES.indexOf(risk.impact);
      if (li >= 0 && ii >= 0) {
        matrix[li][ii]++;
      }
    }
    return matrix;
  }, [allRisksData]);

  const handleRefreshAll = () => {
    refetch();
    refetchAcceptance();
    void refetchAllRisks();
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {/* Page Header */}
        <PageHeader
          title="Risk Register"
          description="Identify, assess, and manage organizational risks with business impact alignment and acceptance workflows."
          actions={
            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Add Risk
            </Button>
          }
        />

        {/* KPI Stats Row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <KpiCard
            title="Total Risks"
            value={stats?.total ?? 0}
            icon={ShieldAlert}
            iconColor="text-blue-600"
            loading={statsLoading}
            description="All registered risks"
          />
          <KpiCard
            title="Avg Residual Score"
            value={stats?.avg_residual_score?.toFixed(1) ?? '0'}
            icon={Target}
            iconColor="text-amber-600"
            loading={statsLoading}
            description="Average residual risk"
          />
          <KpiCard
            title="Overdue Reviews"
            value={stats?.overdue_reviews ?? 0}
            icon={Clock}
            iconColor="text-red-600"
            loading={statsLoading}
            description="Past review date"
          />
          <KpiCard
            title="Accepted Risks"
            value={stats?.accepted_count ?? 0}
            icon={CheckCircle}
            iconColor="text-green-600"
            loading={statsLoading}
            description="Formally accepted"
          />
        </div>

        {/* Tabs */}
        <Tabs defaultValue="register" className="space-y-4">
          <TabsList>
            <TabsTrigger value="register">Risk Register</TabsTrigger>
            <TabsTrigger value="acceptance">Risk Acceptance</TabsTrigger>
            <TabsTrigger value="impact">Business Impact</TabsTrigger>
          </TabsList>

          {/* ── Risk Register Tab ──────────────────────────────── */}
          <TabsContent value="register" className="space-y-4">
            <DataTable
              columns={riskColumns}
              filters={RISK_FILTERS}
              rowActions={riskRowActions}
              searchPlaceholder="Search risks..."
              emptyState={{
                icon: ShieldAlert,
                title: 'No risks found',
                description: 'No risks match the current filters or no risks have been registered yet.',
                action: {
                  label: 'Add Risk',
                  onClick: () => setShowCreateDialog(true),
                  icon: Plus,
                },
              }}
              onRowClick={(row) => setSelectedRisk(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...tableProps}
            />
          </TabsContent>

          {/* ── Risk Acceptance Tab ────────────────────────────── */}
          <TabsContent value="acceptance" className="space-y-4">
            <DataTable
              columns={acceptanceColumns}
              rowActions={acceptanceRowActions}
              searchPlaceholder="Search accepted risks..."
              emptyState={{
                icon: CheckCircle,
                title: 'No accepted risks',
                description: 'No risks have been formally accepted yet.',
              }}
              onRowClick={(row) => setSelectedRisk(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...acceptanceTableProps}
            />
          </TabsContent>

          {/* ── Business Impact Tab ───────────────────────────── */}
          <TabsContent value="impact" className="space-y-6">
            {allRisksError ? (
              <ErrorState
                message="Failed to load business impact data"
                onRetry={() => void refetchAllRisks()}
              />
            ) : allRisksLoading ? (
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                {[1, 2, 3, 4].map((i) => (
                  <Card key={i}>
                    <CardHeader>
                      <div className="h-5 w-40 animate-pulse rounded bg-muted" />
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        <div className="h-4 w-full animate-pulse rounded bg-muted" />
                        <div className="h-4 w-3/4 animate-pulse rounded bg-muted" />
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : (
              <>
                {/* Risk Heat Matrix */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                      <BarChart3 className="h-5 w-5 text-muted-foreground" />
                      Risk Heat Matrix
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="overflow-x-auto">
                      <table className="w-full border-collapse">
                        <thead>
                          <tr>
                            <th className="p-2 text-xs text-muted-foreground text-left">
                              Likelihood / Impact
                            </th>
                            {IMPACT_LABELS.map((label) => (
                              <th
                                key={label}
                                className="p-2 text-xs font-medium text-center text-muted-foreground min-w-[90px]"
                              >
                                {label}
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {LIKELIHOOD_LABELS.slice()
                            .reverse()
                            .map((label, reversedIdx) => {
                              const likelihoodIdx = 4 - reversedIdx;
                              return (
                                <tr key={label}>
                                  <td className="p-2 text-xs font-medium text-muted-foreground whitespace-nowrap">
                                    {label}
                                  </td>
                                  {IMPACT_LABELS.map((impactLabel, impactIdx) => {
                                    const count = heatMatrix[likelihoodIdx][impactIdx];
                                    return (
                                      <td key={impactLabel} className="p-1">
                                        <div
                                          className={cn(
                                            'flex items-center justify-center rounded-lg border p-3 text-sm font-bold transition-colors',
                                            count > 0
                                              ? getHeatColor(likelihoodIdx, impactIdx)
                                              : 'bg-muted/30 text-muted-foreground border-transparent',
                                          )}
                                        >
                                          {count}
                                        </div>
                                      </td>
                                    );
                                  })}
                                </tr>
                              );
                            })}
                        </tbody>
                      </table>
                    </div>
                  </CardContent>
                </Card>

                {/* Department Groups */}
                <div className="space-y-4">
                  <h3 className="text-lg font-semibold flex items-center gap-2">
                    <Building2 className="h-5 w-5 text-muted-foreground" />
                    Risks by Department
                  </h3>

                  {departmentGroups.length === 0 ? (
                    <Card>
                      <CardContent className="py-12 text-center text-muted-foreground text-sm">
                        No risks to display.
                      </CardContent>
                    </Card>
                  ) : (
                    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                      {departmentGroups.map(([department, risks]) => {
                        const avgResidual =
                          risks.length > 0
                            ? risks.reduce((sum, r) => sum + r.residual_score, 0) / risks.length
                            : 0;
                        const criticalCount = risks.filter(
                          (r) => r.residual_score > 60,
                        ).length;
                        const services = new Set<string>();
                        for (const r of risks) {
                          for (const s of r.business_services) {
                            services.add(s);
                          }
                        }

                        return (
                          <Card key={department}>
                            <CardHeader className="pb-3">
                              <div className="flex items-center justify-between">
                                <CardTitle className="text-sm font-semibold">
                                  {department}
                                </CardTitle>
                                <Badge variant="secondary" className="text-xs">
                                  {risks.length} risk{risks.length !== 1 ? 's' : ''}
                                </Badge>
                              </div>
                            </CardHeader>
                            <CardContent className="space-y-3">
                              <div className="grid grid-cols-1 gap-3 text-center sm:grid-cols-3">
                                <div>
                                  <p className="text-xs text-muted-foreground">Avg Residual</p>
                                  <p
                                    className={cn(
                                      'text-lg font-bold',
                                      avgResidual <= 30
                                        ? 'text-green-600'
                                        : avgResidual <= 60
                                          ? 'text-amber-600'
                                          : 'text-red-600',
                                    )}
                                  >
                                    {avgResidual.toFixed(1)}
                                  </p>
                                </div>
                                <div>
                                  <p className="text-xs text-muted-foreground">Critical</p>
                                  <p
                                    className={cn(
                                      'text-lg font-bold',
                                      criticalCount > 0
                                        ? 'text-red-600'
                                        : 'text-green-600',
                                    )}
                                  >
                                    {criticalCount}
                                  </p>
                                </div>
                                <div>
                                  <p className="text-xs text-muted-foreground">Services</p>
                                  <p className="text-lg font-bold text-blue-600">
                                    {services.size}
                                  </p>
                                </div>
                              </div>

                              {services.size > 0 && (
                                <>
                                  <Separator />
                                  <div>
                                    <p className="text-xs text-muted-foreground mb-1.5">
                                      Business Services
                                    </p>
                                    <div className="flex flex-wrap gap-1">
                                      {Array.from(services)
                                        .slice(0, 5)
                                        .map((s) => (
                                          <Badge
                                            key={s}
                                            variant="outline"
                                            className="text-xs"
                                          >
                                            {s}
                                          </Badge>
                                        ))}
                                      {services.size > 5 && (
                                        <Badge variant="secondary" className="text-xs">
                                          +{services.size - 5} more
                                        </Badge>
                                      )}
                                    </div>
                                  </div>
                                </>
                              )}

                              {/* Top risks in department */}
                              <Separator />
                              <div>
                                <p className="text-xs text-muted-foreground mb-1.5">
                                  Top Risks
                                </p>
                                <div className="space-y-1.5">
                                  {risks
                                    .sort((a, b) => b.residual_score - a.residual_score)
                                    .slice(0, 3)
                                    .map((r) => (
                                      <div
                                        key={r.id}
                                        className="flex items-center justify-between text-sm cursor-pointer hover:bg-muted/40 rounded-lg px-2 py-1 transition-colors"
                                        onClick={() => setSelectedRisk(r)}
                                      >
                                        <span className="truncate mr-2">{r.title}</span>
                                        <span
                                          className={cn(
                                            'shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold',
                                            residualScoreColor(r.residual_score),
                                          )}
                                        >
                                          {r.residual_score}
                                        </span>
                                      </div>
                                    ))}
                                </div>
                              </div>
                            </CardContent>
                          </Card>
                        );
                      })}
                    </div>
                  )}
                </div>
              </>
            )}
          </TabsContent>
        </Tabs>
      </div>

      {/* ── Detail Panel ──────────────────────────────────────── */}
      {selectedRisk && (
        <RiskDetailPanel
          open={!!selectedRisk}
          onOpenChange={(o) => {
            if (!o) setSelectedRisk(null);
          }}
          risk={selectedRisk}
          onUpdated={handleRefreshAll}
        />
      )}

      {/* ── Create Dialog ─────────────────────────────────────── */}
      <RiskFormDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onCreated={handleRefreshAll}
      />

      {/* ── Accept Risk Dialog ────────────────────────────────── */}
      {acceptTarget && (
        <RiskAcceptanceDialog
          open={!!acceptTarget}
          onOpenChange={(o) => {
            if (!o) setAcceptTarget(null);
          }}
          risk={acceptTarget}
          onAccepted={handleRefreshAll}
        />
      )}

      {/* ── Close Risk Confirm ────────────────────────────────── */}
      <ConfirmDialog
        open={!!closeTarget}
        onOpenChange={(o) => {
          if (!o) setCloseTarget(null);
        }}
        title="Close Risk"
        description={`Are you sure you want to close the risk "${closeTarget?.title ?? ''}"? This indicates the risk is no longer applicable or has been fully resolved.`}
        confirmLabel="Close Risk"
        variant="destructive"
        loading={closeMutation.isPending}
        onConfirm={async () => {
          if (closeTarget) {
            closeMutation.mutate({
              id: closeTarget.id,
              status: 'closed',
            });
          }
        }}
      />

      {/* ── Revoke Acceptance Confirm ─────────────────────────── */}
      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(o) => {
          if (!o) setRevokeTarget(null);
        }}
        title="Revoke Risk Acceptance"
        description={`Are you sure you want to revoke the acceptance for "${revokeTarget?.title ?? ''}"? The risk will be moved back to "open" status for re-evaluation.`}
        confirmLabel="Revoke Acceptance"
        variant="destructive"
        loading={revokeMutation.isPending}
        onConfirm={async () => {
          if (revokeTarget) {
            revokeMutation.mutate({
              id: revokeTarget.id,
              status: 'open',
              acceptance_rationale: null,
              acceptance_approved_by: null,
              acceptance_approved_by_name: null,
              acceptance_expiry: null,
            });
          }
        }}
      />
    </PermissionRedirect>
  );
}
