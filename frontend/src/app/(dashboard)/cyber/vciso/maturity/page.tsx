'use client';

import { useState, useMemo } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import {
  Gauge,
  Plus,
  Eye,
  CheckCircle,
  Clock,
  DollarSign,
  TrendingDown,
  ArrowDownRight,
  BarChart3,
  Target,
  Play,
  AlertTriangle,
  Lightbulb,
  Users,
  Cog,
  Cpu,
} from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { StatusBadge } from '@/components/shared/status-badge';
import { GaugeChart } from '@/components/shared/charts/gauge-chart';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { DetailPanel } from '@/components/shared/detail-panel';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs';
import { Separator } from '@/components/ui/separator';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import {
  budgetItemStatusConfig,
  maturityAssessmentStatusConfig,
} from '@/lib/status-configs';
import { formatDate, formatCurrency, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig, RowAction } from '@/types/table';
import type {
  VCISOMaturityAssessment,
  VCISOMaturityDimension,
  VCISOBenchmark,
  VCISOBudgetItem,
  VCISOBudgetSummary,
  MaturityCategory,
} from '@/types/cyber';

import { MaturityDimensionCard } from './_components/maturity-dimension-card';
import { BudgetFormDialog } from './_components/budget-form-dialog';
import { BudgetDetailPanel } from './_components/budget-detail-panel';

// ── Helpers ──────────────────────────────────────────────────────────────────

const CATEGORY_BADGE_CLASSES: Record<MaturityCategory, string> = {
  people: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  process: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  technology: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
};

function gapColor(gap: number): string {
  if (gap > 0) return 'text-green-600';
  if (gap === 0) return 'text-muted-foreground';
  return 'text-red-600';
}

function gapBgColor(gap: number): string {
  if (gap > 0) return 'bg-green-50 dark:bg-green-900/10';
  if (gap === 0) return '';
  return 'bg-red-50 dark:bg-red-900/10';
}

// ── Budget Filters ──────────────────────────────────────────────────────────

const BUDGET_FILTERS: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Proposed', value: 'proposed' },
      { label: 'Approved', value: 'approved' },
      { label: 'In Progress', value: 'in_progress' },
      { label: 'Completed', value: 'completed' },
      { label: 'Deferred', value: 'deferred' },
    ],
  },
  {
    key: 'type',
    label: 'Type',
    type: 'select',
    options: [
      { label: 'CapEx', value: 'capex' },
      { label: 'OpEx', value: 'opex' },
    ],
  },
  {
    key: 'fiscal_year',
    label: 'Fiscal Year',
    type: 'select',
    options: [
      { label: '2024', value: '2024' },
      { label: '2025', value: '2025' },
      { label: '2026', value: '2026' },
      { label: '2027', value: '2027' },
    ],
  },
];

// ── Budget Columns ──────────────────────────────────────────────────────────

function getBudgetColumns(): ColumnDef<VCISOBudgetItem>[] {
  return [
    {
      accessorKey: 'title',
      header: 'Title',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-foreground max-w-[200px] truncate block">
          {row.original.title}
        </span>
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
      accessorKey: 'type',
      header: 'Type',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge
          variant="secondary"
          className={cn(
            'text-xs',
            row.original.type === 'capex'
              ? 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300'
              : 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
          )}
        >
          {row.original.type === 'capex' ? 'CapEx' : 'OpEx'}
        </Badge>
      ),
    },
    {
      accessorKey: 'amount',
      header: 'Amount',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm font-medium tabular-nums">
          {formatCurrency(row.original.amount, row.original.currency)}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge
          status={row.original.status}
          config={budgetItemStatusConfig}
        />
      ),
    },
    {
      accessorKey: 'risk_reduction_estimate',
      header: 'Risk Reduction',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm font-medium text-green-600 tabular-nums">
          {row.original.risk_reduction_estimate}%
        </span>
      ),
    },
    {
      accessorKey: 'priority',
      header: 'Priority',
      enableSorting: true,
      cell: ({ row }) => {
        const p = row.original.priority;
        const color =
          p <= 1
            ? 'bg-red-100 text-red-700'
            : p <= 2
              ? 'bg-amber-100 text-amber-700'
              : 'bg-gray-100 text-gray-700';
        return (
          <span
            className={cn(
              'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold',
              color,
            )}
          >
            P{p}
          </span>
        );
      },
    },
    {
      accessorKey: 'fiscal_year',
      header: 'Fiscal Year',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.fiscal_year}
          {row.original.quarter ? ` ${row.original.quarter}` : ''}
        </span>
      ),
    },
    {
      accessorKey: 'owner_name',
      header: 'Owner',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.owner_name || 'Unassigned'}
        </span>
      ),
    },
  ];
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function MaturityBudgetPage() {
  const [selectedDimension, setSelectedDimension] =
    useState<VCISOMaturityDimension | null>(null);
  const [selectedBudgetItem, setSelectedBudgetItem] =
    useState<VCISOBudgetItem | null>(null);
  const [budgetDetailOpen, setBudgetDetailOpen] = useState(false);
  const [showBudgetForm, setShowBudgetForm] = useState(false);
  const [approveTarget, setApproveTarget] = useState<VCISOBudgetItem | null>(
    null,
  );
  const [deferTarget, setDeferTarget] = useState<VCISOBudgetItem | null>(null);

  // ── Maturity Assessment Data ────────────────────────────────────────────
  const {
    data: assessmentEnvelope,
    isLoading: assessmentLoading,
    error: assessmentError,
    mutate: refetchAssessment,
  } = useRealtimeData<{ data: VCISOMaturityAssessment[] }>(
    API_ENDPOINTS.CYBER_VCISO_MATURITY,
    {
      wsTopics: ['vciso.maturity'],
    },
  );
  const assessment = assessmentEnvelope?.data?.[0];

  // ── Benchmarking Data ──────────────────────────────────────────────────
  const {
    data: benchmarksEnvelope,
    isLoading: benchmarksLoading,
    error: benchmarksError,
    mutate: refetchBenchmarks,
  } = useRealtimeData<{ data: VCISOBenchmark[] }>(API_ENDPOINTS.CYBER_VCISO_BENCHMARKS, {
    wsTopics: ['vciso.benchmarks'],
  });
  const benchmarks = benchmarksEnvelope?.data;

  // ── Budget Summary ────────────────────────────────────────────────────
  const {
    data: budgetSummaryEnvelope,
    isLoading: summaryLoading,
    mutate: refetchSummary,
  } = useRealtimeData<{ data: VCISOBudgetSummary }>(
    API_ENDPOINTS.CYBER_VCISO_BUDGET_SUMMARY,
    {
      wsTopics: ['vciso.budget'],
    },
  );
  const budgetSummary = budgetSummaryEnvelope?.data;

  // ── Budget Table ──────────────────────────────────────────────────────
  const { tableProps, refetch: refetchBudget } = useDataTable<VCISOBudgetItem>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOBudgetItem>>(
        API_ENDPOINTS.CYBER_VCISO_BUDGET,
        params as unknown as Record<string, unknown>,
      ),
    queryKey: 'vciso-budget',
    defaultSort: { column: 'priority', direction: 'asc' },
    wsTopics: ['vciso.budget'],
  });

  // ── Start Assessment Mutation ─────────────────────────────────────────
  const startAssessmentMutation = useApiMutation<
    VCISOMaturityAssessment,
    Record<string, unknown>
  >('post', API_ENDPOINTS.CYBER_VCISO_MATURITY, {
    successMessage: 'Assessment started',
    invalidateKeys: [API_ENDPOINTS.CYBER_VCISO_MATURITY],
    onSuccess: () => {
      void refetchAssessment();
    },
  });

  // ── Approve Budget Item Mutation ──────────────────────────────────────
  const approveMutation = useApiMutation<
    VCISOBudgetItem,
    Record<string, unknown>
  >(
    'put',
    (variables) =>
      `${API_ENDPOINTS.CYBER_VCISO_BUDGET}/${(variables as Record<string, string>).id}`,
    {
      successMessage: 'Budget item approved',
      invalidateKeys: [
        'vciso-budget',
        API_ENDPOINTS.CYBER_VCISO_BUDGET_SUMMARY,
      ],
      onSuccess: () => {
        setApproveTarget(null);
        refetchBudget();
        void refetchSummary();
      },
    },
  );

  // ── Defer Budget Item Mutation ────────────────────────────────────────
  const deferMutation = useApiMutation<
    VCISOBudgetItem,
    Record<string, unknown>
  >(
    'put',
    (variables) =>
      `${API_ENDPOINTS.CYBER_VCISO_BUDGET}/${(variables as Record<string, string>).id}`,
    {
      successMessage: 'Budget item deferred',
      invalidateKeys: [
        'vciso-budget',
        API_ENDPOINTS.CYBER_VCISO_BUDGET_SUMMARY,
      ],
      onSuccess: () => {
        setDeferTarget(null);
        refetchBudget();
        void refetchSummary();
      },
    },
  );

  // ── Columns ───────────────────────────────────────────────────────────
  const budgetColumns = useMemo(() => getBudgetColumns(), []);

  // ── Budget Row Actions ────────────────────────────────────────────────
  const budgetRowActions: RowAction<VCISOBudgetItem>[] = [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (row) => {
        setSelectedBudgetItem(row);
        setBudgetDetailOpen(true);
      },
    },
    {
      label: 'Approve',
      icon: CheckCircle,
      onClick: (row) => setApproveTarget(row),
      hidden: (row) =>
        row.status !== 'proposed',
    },
    {
      label: 'Defer',
      icon: Clock,
      onClick: (row) => setDeferTarget(row),
      hidden: (row) =>
        row.status === 'deferred' ||
        row.status === 'completed',
    },
  ];

  // ── Benchmark Chart Data ──────────────────────────────────────────────
  const benchmarkChartData = useMemo(() => {
    if (!benchmarks || !Array.isArray(benchmarks)) return [];
    return benchmarks.map((b) => ({
      dimension: b.dimension,
      'Organization': b.organization_score,
      'Industry Avg': b.industry_average,
      'Peer Avg': b.peer_average,
      'Top Quartile': b.industry_top_quartile,
    }));
  }, [benchmarks]);

  const benchmarkBarSeries = [
    { key: 'Organization', label: 'Organization', color: '#1B5E20' },
    { key: 'Industry Avg', label: 'Industry Avg', color: '#64748b' },
    { key: 'Peer Avg', label: 'Peer Avg', color: '#0ea5e9' },
    { key: 'Top Quartile', label: 'Top Quartile', color: '#C6A962' },
  ];

  // ── Benchmark Gap Analysis ────────────────────────────────────────────
  const benchmarkGapStats = useMemo(() => {
    if (!benchmarks || !Array.isArray(benchmarks) || benchmarks.length === 0)
      return null;
    const totalGap = benchmarks.reduce((sum, b) => sum + b.gap, 0);
    const avgGap = totalGap / benchmarks.length;
    const needsImprovement = benchmarks.filter((b) => b.gap < 0);
    return { avgGap, needsImprovement };
  }, [benchmarks]);

  // ── Dimension groups by category ──────────────────────────────────────
  const dimensionsByCategory = useMemo(() => {
    if (!assessment?.dimensions) return new Map<MaturityCategory, VCISOMaturityDimension[]>();
    const map = new Map<MaturityCategory, VCISOMaturityDimension[]>();
    for (const dim of assessment.dimensions) {
      const existing = map.get(dim.category);
      if (existing) {
        existing.push(dim);
      } else {
        map.set(dim.category, [dim]);
      }
    }
    return map;
  }, [assessment]);

  const handleRefreshAll = () => {
    refetchBudget();
    void refetchSummary();
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {/* Page Header */}
        <PageHeader
          title="Maturity & Budget"
          description="Assess your security maturity, benchmark against industry peers, and prioritize security investments."
        />

        {/* Tabs */}
        <Tabs defaultValue="maturity" className="space-y-4">
          <TabsList>
            <TabsTrigger value="maturity">Maturity Assessment</TabsTrigger>
            <TabsTrigger value="benchmarking">Benchmarking</TabsTrigger>
            <TabsTrigger value="budget">Security Budget</TabsTrigger>
          </TabsList>

          {/* ═══════════════════════════════════════════════════════════════
              MATURITY ASSESSMENT TAB
          ═══════════════════════════════════════════════════════════════ */}
          <TabsContent value="maturity" className="space-y-6">
            {assessmentLoading ? (
              <div className="space-y-4">
                <LoadingSkeleton variant="chart" />
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                  <LoadingSkeleton variant="card" count={3} />
                </div>
              </div>
            ) : assessmentError ? (
              <ErrorState
                message="Failed to load maturity assessment data"
                onRetry={() => void refetchAssessment()}
              />
            ) : !assessment ? (
              /* No assessment yet */
              <Card>
                <CardContent className="py-16 text-center">
                  <Gauge className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
                  <h3 className="text-lg font-semibold mb-2">
                    No Assessment Started
                  </h3>
                  <p className="text-sm text-muted-foreground mb-6 max-w-md mx-auto">
                    Start a security maturity assessment to evaluate your
                    organization across people, process, and technology
                    dimensions.
                  </p>
                  <Button
                    onClick={() =>
                      startAssessmentMutation.mutate({ framework: 'NIST CSF' })
                    }
                    disabled={startAssessmentMutation.isPending}
                  >
                    <Play className="mr-2 h-4 w-4" />
                    {startAssessmentMutation.isPending
                      ? 'Starting...'
                      : 'Start Assessment'}
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <>
                {/* Assessment Overview */}
                <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                  {/* Gauge Chart */}
                  <Card className="lg:col-span-1">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-semibold">
                        Overall Maturity
                      </CardTitle>
                      <CardDescription>
                        {assessment.framework} Framework
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="flex flex-col items-center">
                      <GaugeChart
                        value={assessment.overall_score}
                        max={5}
                        thresholds={{ good: 80, warning: 50 }}
                        label="Maturity Level"
                        format="number"
                        size={200}
                      />
                      <div className="mt-4 text-center space-y-2">
                        <p className="text-lg font-bold">
                          Level {assessment.overall_level}/5
                        </p>
                        <StatusBadge
                          status={assessment.status}
                          config={maturityAssessmentStatusConfig}
                        />
                        {assessment.assessor_name && (
                          <p className="text-xs text-muted-foreground">
                            Assessed by {assessment.assessor_name}
                          </p>
                        )}
                        <p className="text-xs text-muted-foreground">
                          {formatDate(assessment.assessed_at)}
                        </p>
                      </div>
                    </CardContent>
                  </Card>

                  {/* Assessment Summary */}
                  <Card className="lg:col-span-2">
                    <CardHeader className="pb-3">
                      <div className="flex items-center justify-between">
                        <CardTitle className="text-sm font-semibold">
                          Assessment Summary
                        </CardTitle>
                        {assessment.status === 'in_progress' && (
                          <Button size="sm" variant="outline">
                            <Play className="mr-1.5 h-3.5 w-3.5" />
                            Continue Assessment
                          </Button>
                        )}
                      </div>
                    </CardHeader>
                    <CardContent>
                      {/* Category summary cards */}
                      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                        {Array.from(dimensionsByCategory.keys()).map(
                          (cat) => {
                            const dims = dimensionsByCategory.get(cat) ?? [];
                            const avgScore =
                              dims.length > 0
                                ? dims.reduce((s, d) => s + d.score, 0) /
                                  dims.length
                                : 0;
                            const totalFindings = dims.reduce(
                              (s, d) => s + d.findings.length,
                              0,
                            );
                            const totalRecs = dims.reduce(
                              (s, d) => s + d.recommendations.length,
                              0,
                            );
                            const CatIcon =
                              cat === 'people' || cat === 'operations'
                                ? Users
                                : cat === 'process' || cat === 'governance'
                                  ? Cog
                                  : Cpu;

                            return (
                              <div
                                key={cat}
                                className="rounded-lg border p-4 space-y-2"
                              >
                                <div className="flex items-center gap-2">
                                  <CatIcon className="h-4 w-4 text-muted-foreground" />
                                  <span className="text-sm font-semibold capitalize">
                                    {cat}
                                  </span>
                                </div>
                                <p className="text-2xl font-bold tabular-nums">
                                  {avgScore.toFixed(1)}
                                </p>
                                <p className="text-xs text-muted-foreground">
                                  {dims.length} dimensions
                                </p>
                                <div className="flex items-center gap-3 text-xs">
                                  <span className="flex items-center gap-1 text-amber-600">
                                    <AlertTriangle className="h-3 w-3" />
                                    {totalFindings}
                                  </span>
                                  <span className="flex items-center gap-1 text-blue-600">
                                    <Lightbulb className="h-3 w-3" />
                                    {totalRecs}
                                  </span>
                                </div>
                              </div>
                            );
                          },
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </div>

                {/* Dimension Cards Grid */}
                <div>
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <Target className="h-5 w-5 text-muted-foreground" />
                    Dimensions ({assessment.dimensions?.length ?? 0})
                  </h3>
                  <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                    {(assessment.dimensions ?? []).map((dim) => (
                      <MaturityDimensionCard
                        key={dim.name}
                        dimension={dim}
                        onViewDetails={(d) => setSelectedDimension(d)}
                      />
                    ))}
                  </div>
                </div>
              </>
            )}
          </TabsContent>

          {/* ═══════════════════════════════════════════════════════════════
              BENCHMARKING TAB
          ═══════════════════════════════════════════════════════════════ */}
          <TabsContent value="benchmarking" className="space-y-6">
            {benchmarksLoading ? (
              <div className="space-y-4">
                <LoadingSkeleton variant="chart" />
                <LoadingSkeleton variant="table-row" count={5} />
              </div>
            ) : benchmarksError ? (
              <ErrorState
                message="Failed to load benchmarking data"
                onRetry={() => void refetchBenchmarks()}
              />
            ) : !benchmarks || !Array.isArray(benchmarks) || benchmarks.length === 0 ? (
              <Card>
                <CardContent className="py-16 text-center">
                  <BarChart3 className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
                  <h3 className="text-lg font-semibold mb-2">
                    No Benchmark Data Available
                  </h3>
                  <p className="text-sm text-muted-foreground max-w-md mx-auto">
                    Complete a maturity assessment first to generate benchmarking
                    data against industry peers.
                  </p>
                </CardContent>
              </Card>
            ) : (
              <>
                {/* Benchmark Summary Card */}
                {benchmarkGapStats && (
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <Card>
                      <CardHeader className="pb-3">
                        <CardTitle className="text-sm font-semibold">
                          Average Gap
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <p
                          className={cn(
                            'text-3xl font-bold tabular-nums',
                            gapColor(benchmarkGapStats.avgGap),
                          )}
                        >
                          {benchmarkGapStats.avgGap > 0 ? '+' : ''}
                          {benchmarkGapStats.avgGap.toFixed(2)}
                        </p>
                        <p className="text-sm text-muted-foreground mt-1">
                          {benchmarkGapStats.avgGap >= 0
                            ? 'Performing at or above industry average'
                            : 'Below industry average across assessed dimensions'}
                        </p>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-3">
                        <CardTitle className="text-sm font-semibold">
                          Areas Needing Improvement
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <p className="text-3xl font-bold tabular-nums text-red-600">
                          {benchmarkGapStats.needsImprovement.length}
                        </p>
                        <p className="text-sm text-muted-foreground mt-1">
                          {benchmarkGapStats.needsImprovement.length > 0
                            ? `${benchmarkGapStats.needsImprovement.map((b) => b.dimension).join(', ')}`
                            : 'All dimensions meet or exceed benchmarks'}
                        </p>
                      </CardContent>
                    </Card>
                  </div>
                )}

                {/* Benchmark Bar Chart */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <BarChart3 className="h-5 w-5 text-muted-foreground" />
                      Benchmark Comparison
                    </CardTitle>
                    <CardDescription>
                      Organization score vs industry benchmarks across all
                      dimensions
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <BarChart
                      data={benchmarkChartData}
                      xKey="dimension"
                      yKeys={benchmarkBarSeries}
                      height={380}
                      showGrid
                      showLegend
                    />
                  </CardContent>
                </Card>

                {/* Gap Analysis Table */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">
                      Gap Analysis
                    </CardTitle>
                    <CardDescription>
                      Detailed comparison of your scores against industry
                      benchmarks
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="overflow-x-auto">
                      <table className="w-full border-collapse text-sm">
                        <thead>
                          <tr className="border-b">
                            <th className="text-left p-3 font-medium text-muted-foreground">
                              Dimension
                            </th>
                            <th className="text-left p-3 font-medium text-muted-foreground">
                              Category
                            </th>
                            <th className="text-right p-3 font-medium text-muted-foreground">
                              Org Score
                            </th>
                            <th className="text-right p-3 font-medium text-muted-foreground">
                              Industry Avg
                            </th>
                            <th className="text-right p-3 font-medium text-muted-foreground">
                              Peer Avg
                            </th>
                            <th className="text-right p-3 font-medium text-muted-foreground">
                              Top Quartile
                            </th>
                            <th className="text-right p-3 font-medium text-muted-foreground">
                              Gap
                            </th>
                          </tr>
                        </thead>
                        <tbody>
                          {benchmarks.map((b) => (
                            <tr
                              key={b.dimension}
                              className={cn(
                                'border-b last:border-0 transition-colors hover:bg-muted/50',
                                gapBgColor(b.gap),
                              )}
                            >
                              <td className="p-3 font-medium">
                                {b.dimension}
                              </td>
                              <td className="p-3">
                                <Badge
                                  variant="secondary"
                                  className={cn(
                                    'text-xs',
                                    CATEGORY_BADGE_CLASSES[b.category],
                                  )}
                                >
                                  {titleCase(b.category)}
                                </Badge>
                              </td>
                              <td className="p-3 text-right font-medium tabular-nums">
                                {b.organization_score.toFixed(2)}
                              </td>
                              <td className="p-3 text-right tabular-nums text-muted-foreground">
                                {b.industry_average.toFixed(2)}
                              </td>
                              <td className="p-3 text-right tabular-nums text-muted-foreground">
                                {b.peer_average.toFixed(2)}
                              </td>
                              <td className="p-3 text-right tabular-nums text-muted-foreground">
                                {b.industry_top_quartile.toFixed(2)}
                              </td>
                              <td
                                className={cn(
                                  'p-3 text-right font-bold tabular-nums',
                                  gapColor(b.gap),
                                )}
                              >
                                {b.gap > 0 ? '+' : ''}
                                {b.gap.toFixed(2)}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </CardContent>
                </Card>
              </>
            )}
          </TabsContent>

          {/* ═══════════════════════════════════════════════════════════════
              SECURITY BUDGET TAB
          ═══════════════════════════════════════════════════════════════ */}
          <TabsContent value="budget" className="space-y-6">
            {/* KPI Row */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <KpiCard
                title="Total Proposed"
                value={
                  budgetSummary
                    ? formatCurrency(
                        budgetSummary.total_proposed,
                        budgetSummary.currency,
                      )
                    : '$0'
                }
                icon={DollarSign}
                iconColor="text-amber-600"
                loading={summaryLoading}
                description="Pending approval"
              />
              <KpiCard
                title="Total Approved"
                value={
                  budgetSummary
                    ? formatCurrency(
                        budgetSummary.total_approved,
                        budgetSummary.currency,
                      )
                    : '$0'
                }
                icon={CheckCircle}
                iconColor="text-green-600"
                loading={summaryLoading}
                description="Approved for spending"
              />
              <KpiCard
                title="Total Spent"
                value={
                  budgetSummary
                    ? formatCurrency(
                        budgetSummary.total_spent,
                        budgetSummary.currency,
                      )
                    : '$0'
                }
                icon={TrendingDown}
                iconColor="text-blue-600"
                loading={summaryLoading}
                description={
                  budgetSummary && budgetSummary.total_approved > 0
                    ? `${((budgetSummary.total_spent / budgetSummary.total_approved) * 100).toFixed(1)}% of approved`
                    : 'Of approved budget'
                }
              />
              <KpiCard
                title="Risk Reduction"
                value={
                  budgetSummary
                    ? `${budgetSummary.total_risk_reduction}%`
                    : '0%'
                }
                icon={ArrowDownRight}
                iconColor="text-emerald-600"
                loading={summaryLoading}
                description="Estimated risk reduction"
              />
            </div>

            {/* Budget by Category Breakdown */}
            {budgetSummary &&
              Object.keys(budgetSummary.by_category).length > 0 && (
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-semibold">
                      Budget by Category
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
                      {Object.entries(budgetSummary.by_category)
                        .sort(([, a], [, b]) => b - a)
                        .map(([category, amount]) => (
                          <div
                            key={category}
                            className="flex items-center justify-between rounded-lg border p-3"
                          >
                            <span className="text-sm font-medium truncate mr-2">
                              {category}
                            </span>
                            <span className="text-sm font-semibold tabular-nums shrink-0">
                              {formatCurrency(amount, budgetSummary.currency)}
                            </span>
                          </div>
                        ))}
                    </div>
                  </CardContent>
                </Card>
              )}

            {/* Budget Data Table */}
            <DataTable
              columns={budgetColumns}
              filters={BUDGET_FILTERS}
              rowActions={budgetRowActions}
              searchPlaceholder="Search budget items..."
              emptyState={{
                icon: DollarSign,
                title: 'No budget items found',
                description:
                  'No budget items match the current filters or none have been created yet.',
                action: {
                  label: 'Add Budget Item',
                  onClick: () => setShowBudgetForm(true),
                  icon: Plus,
                },
              }}
              onRowClick={(row) => {
                setSelectedBudgetItem(row);
                setBudgetDetailOpen(true);
              }}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...tableProps}
            />

            {/* Add Budget Item Button (floating) */}
            <div className="flex justify-end">
              <Button onClick={() => setShowBudgetForm(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Add Budget Item
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </div>

      {/* ── Dimension Detail Panel ─────────────────────────────────────── */}
      <DetailPanel
        open={!!selectedDimension}
        onOpenChange={(o) => {
          if (!o) setSelectedDimension(null);
        }}
        title={selectedDimension?.name ?? 'Dimension Details'}
        description="Full dimension assessment details"
        width="lg"
      >
        {selectedDimension && (
          <div className="space-y-6">
            {/* Category & Score */}
            <div className="flex flex-wrap items-center gap-2">
              <Badge
                variant="secondary"
                className={cn(
                  'text-xs',
                  CATEGORY_BADGE_CLASSES[selectedDimension.category],
                )}
              >
                {titleCase(selectedDimension.category)}
              </Badge>
              <Badge variant="outline" className="text-xs">
                Score: {selectedDimension.score.toFixed(1)}
              </Badge>
            </div>

            {/* Level Progress */}
            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
                Maturity Level Progress
              </h4>
              <div className="flex items-center justify-between text-sm mb-1.5">
                <span>
                  Current: Level {selectedDimension.current_level}
                </span>
                <span>
                  Target: Level {selectedDimension.target_level}
                </span>
              </div>
              <div className="h-3 w-full rounded-full bg-muted overflow-hidden">
                <div
                  className="h-full rounded-full bg-primary transition-all duration-500"
                  style={{
                    width: `${Math.min((selectedDimension.current_level / selectedDimension.target_level) * 100, 100)}%`,
                  }}
                />
              </div>
            </div>

            <Separator />

            {/* Findings */}
            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2 flex items-center gap-1">
                <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />
                Findings ({selectedDimension.findings.length})
              </h4>
              {selectedDimension.findings.length > 0 ? (
                <ul className="space-y-2">
                  {selectedDimension.findings.map((finding, idx) => (
                    <li
                      key={idx}
                      className="text-sm text-foreground rounded-lg border border-amber-200/60 bg-amber-50/50 dark:bg-amber-900/10 dark:border-amber-800/30 px-3 py-2"
                    >
                      {finding}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No findings identified
                </p>
              )}
            </div>

            <Separator />

            {/* Recommendations */}
            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2 flex items-center gap-1">
                <Lightbulb className="h-3.5 w-3.5 text-blue-500" />
                Recommendations ({selectedDimension.recommendations.length})
              </h4>
              {selectedDimension.recommendations.length > 0 ? (
                <ul className="space-y-2">
                  {selectedDimension.recommendations.map((rec, idx) => (
                    <li
                      key={idx}
                      className="text-sm text-foreground rounded-lg border border-blue-200/60 bg-blue-50/50 dark:bg-blue-900/10 dark:border-blue-800/30 px-3 py-2"
                    >
                      {rec}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No recommendations
                </p>
              )}
            </div>
          </div>
        )}
      </DetailPanel>

      {/* ── Budget Detail Panel ────────────────────────────────────────── */}
      <BudgetDetailPanel
        item={selectedBudgetItem}
        open={budgetDetailOpen}
        onOpenChange={setBudgetDetailOpen}
      />

      {/* ── Budget Form Dialog ─────────────────────────────────────────── */}
      <BudgetFormDialog
        open={showBudgetForm}
        onOpenChange={setShowBudgetForm}
        onCreated={handleRefreshAll}
      />

      {/* ── Approve Budget Confirm ─────────────────────────────────────── */}
      <ConfirmDialog
        open={!!approveTarget}
        onOpenChange={(o) => {
          if (!o) setApproveTarget(null);
        }}
        title="Approve Budget Item"
        description={`Are you sure you want to approve "${approveTarget?.title ?? ''}"? This will move the item to approved status and allocate ${approveTarget ? formatCurrency(approveTarget.amount, approveTarget.currency) : ''}.`}
        confirmLabel="Approve"
        loading={approveMutation.isPending}
        onConfirm={async () => {
          if (approveTarget) {
            approveMutation.mutate({
              id: approveTarget.id,
              status: 'approved',
            });
          }
        }}
      />

      {/* ── Defer Budget Confirm ───────────────────────────────────────── */}
      <ConfirmDialog
        open={!!deferTarget}
        onOpenChange={(o) => {
          if (!o) setDeferTarget(null);
        }}
        title="Defer Budget Item"
        description={`Are you sure you want to defer "${deferTarget?.title ?? ''}"? This item will be moved to deferred status for future consideration.`}
        confirmLabel="Defer"
        variant="destructive"
        loading={deferMutation.isPending}
        onConfirm={async () => {
          if (deferTarget) {
            deferMutation.mutate({
              id: deferTarget.id,
              status: 'deferred',
            });
          }
        }}
      />
    </PermissionRedirect>
  );
}
