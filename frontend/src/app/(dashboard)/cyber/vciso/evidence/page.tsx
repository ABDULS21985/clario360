'use client';

import { useState, useMemo, useCallback } from 'react';
import { type ColumnDef, type Row } from '@tanstack/react-table';
import {
  Archive,
  CheckCircle,
  Download,
  Eye,
  FileText,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Shield,
  Trash2,
  Upload,
  AlertTriangle,
  Layers,
  Bot,
  Monitor,
  User,
} from 'lucide-react';

import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { buildSuiteQueryParams } from '@/lib/suite-api';
import { API_ENDPOINTS } from '@/lib/constants';
import { evidenceStatusConfig } from '@/lib/status-configs';
import { formatDate, formatBytes, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, FilterConfig } from '@/types/table';
import type {
  VCISOEvidence,
  VCISOEvidenceStats,
} from '@/types/cyber';

import { EvidenceDetailPanel } from './_components/evidence-detail-panel';
import { EvidenceFormDialog } from './_components/evidence-form-dialog';

// ─── Filter Configurations ──────────────────────────────────────────────────

const EVIDENCE_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'select',
    options: [
      { label: 'Screenshot', value: 'screenshot' },
      { label: 'Log', value: 'log' },
      { label: 'Configuration', value: 'config' },
      { label: 'Report', value: 'report' },
      { label: 'Policy', value: 'policy' },
      { label: 'Certificate', value: 'certificate' },
      { label: 'Other', value: 'other' },
    ],
  },
  {
    key: 'source',
    label: 'Source',
    type: 'select',
    options: [
      { label: 'Manual', value: 'manual' },
      { label: 'Automated', value: 'automated' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Current', value: 'current' },
      { label: 'Stale', value: 'stale' },
      { label: 'Expired', value: 'expired' },
    ],
  },
];

// ─── Fetch Function ─────────────────────────────────────────────────────────

function fetchEvidence(params: FetchParams): Promise<PaginatedResponse<VCISOEvidence>> {
  return apiGet<PaginatedResponse<VCISOEvidence>>(
    API_ENDPOINTS.CYBER_VCISO_EVIDENCE,
    buildSuiteQueryParams(params),
  );
}

// ─── Type Badge Color Map ───────────────────────────────────────────────────

const TYPE_BADGE_CLASSES: Record<string, string> = {
  screenshot: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  log: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
  config: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  report: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  policy: 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300',
  certificate: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  other: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
};

// ─── Main Page Component ────────────────────────────────────────────────────

export default function EvidencePage() {
  const [selectedEvidence, setSelectedEvidence] = useState<VCISOEvidence | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [editEvidence, setEditEvidence] = useState<VCISOEvidence | null>(null);

  // ── Stats ───────────────────────────────────────────────────────────────
  const {
    data: statsEnvelope,
    isLoading: statsLoading,
    error: statsError,
    mutate: refetchStats,
  } = useRealtimeData<{ data: VCISOEvidenceStats }>(
    API_ENDPOINTS.CYBER_VCISO_EVIDENCE_STATS,
    {
      wsTopics: ['evidence.created', 'evidence.updated', 'evidence.deleted'],
    },
  );
  const stats = statsEnvelope?.data;

  // ── Data Table ──────────────────────────────────────────────────────────
  const { tableProps, refetch, data: evidenceData } = useDataTable<VCISOEvidence>({
    fetchFn: fetchEvidence,
    queryKey: 'vciso-evidence',
    defaultPageSize: 25,
    defaultSort: { column: 'collected_at', direction: 'desc' },
    wsTopics: ['evidence.created', 'evidence.updated', 'evidence.deleted'],
  });

  // ── Mutations ───────────────────────────────────────────────────────────
  const { mutate: deleteEvidence } = useApiMutation<unknown, { id: string }>(
    'delete',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_EVIDENCE}/${variables.id}`,
    {
      successMessage: 'Evidence deleted',
      invalidateKeys: ['vciso-evidence', 'vciso-evidence-stats'],
    },
  );

  const { mutate: verifyEvidence } = useApiMutation<unknown, { id: string }>(
    'put',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_EVIDENCE}/${variables.id}/verify`,
    {
      successMessage: 'Evidence verified',
      invalidateKeys: ['vciso-evidence', 'vciso-evidence-stats'],
    },
  );

  // ── Handlers ────────────────────────────────────────────────────────────
  const handleView = useCallback((evidence: VCISOEvidence) => {
    setSelectedEvidence(evidence);
    setDetailOpen(true);
  }, []);

  const handleUpload = useCallback(() => {
    setEditEvidence(null);
    setFormOpen(true);
  }, []);

  // ── Stale/Expired items for Collection Status tab ───────────────────────
  const staleExpiredItems = useMemo(
    () => evidenceData.filter((e) => e.status === 'stale' || e.status === 'expired'),
    [evidenceData],
  );

  // ── Columns ─────────────────────────────────────────────────────────────
  const columns = useMemo<ColumnDef<VCISOEvidence>[]>(
    () => [
      {
        id: 'title',
        accessorKey: 'title',
        header: 'Title',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <button
            className="text-left font-medium hover:underline max-w-[180px] sm:max-w-[280px] truncate block"
            onClick={() => handleView(row.original)}
          >
            {row.original.title}
          </button>
        ),
        enableSorting: true,
      },
      {
        id: 'type',
        accessorKey: 'type',
        header: 'Type',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <Badge
            variant="secondary"
            className={cn(
              'text-xs capitalize',
              TYPE_BADGE_CLASSES[row.original.type] ?? TYPE_BADGE_CLASSES.other,
            )}
          >
            {titleCase(row.original.type)}
          </Badge>
        ),
        enableSorting: true,
      },
      {
        id: 'source',
        accessorKey: 'source',
        header: 'Source',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <Badge
            variant="secondary"
            className={cn(
              'text-xs',
              row.original.source === 'automated'
                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
                : '',
            )}
          >
            {titleCase(row.original.source)}
          </Badge>
        ),
        enableSorting: true,
      },
      {
        id: 'status',
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <StatusBadge status={row.original.status} config={evidenceStatusConfig} />
        ),
        enableSorting: true,
      },
      {
        id: 'frameworks',
        header: 'Frameworks',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => {
          const fw = row.original.frameworks;
          if (!fw || fw.length === 0) return <span className="text-muted-foreground">--</span>;
          const shown = fw.slice(0, 2);
          const extra = fw.length - 2;
          return (
            <div className="flex flex-wrap gap-1">
              {shown.map((f) => (
                <Badge key={f} variant="outline" className="text-xs">
                  {f}
                </Badge>
              ))}
              {extra > 0 && (
                <Badge variant="outline" className="text-xs text-muted-foreground">
                  +{extra}
                </Badge>
              )}
            </div>
          );
        },
      },
      {
        id: 'file_name',
        header: 'File',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <span className="text-sm text-muted-foreground max-w-[100px] sm:max-w-[150px] truncate block">
            {row.original.file_name ?? '\u2014'}
          </span>
        ),
      },
      {
        id: 'collected_at',
        accessorKey: 'collected_at',
        header: 'Collected',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <span className="text-sm text-muted-foreground">
            {formatDate(row.original.collected_at)}
          </span>
        ),
        enableSorting: true,
      },
      {
        id: 'expires_at',
        accessorKey: 'expires_at',
        header: 'Expires',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <span className="text-sm text-muted-foreground">
            {row.original.expires_at ? formatDate(row.original.expires_at) : '\u2014'}
          </span>
        ),
        enableSorting: true,
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }: { row: Row<VCISOEvidence> }) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => handleView(row.original)}>
                <Eye className="mr-2 h-3.5 w-3.5" />
                View
              </DropdownMenuItem>
              {row.original.file_url && (
                <DropdownMenuItem asChild>
                  <a
                    href={row.original.file_url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Download className="mr-2 h-3.5 w-3.5" />
                    Download
                  </a>
                </DropdownMenuItem>
              )}
              <DropdownMenuItem
                onClick={() => verifyEvidence({ id: row.original.id })}
              >
                <CheckCircle className="mr-2 h-3.5 w-3.5" />
                Verify
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive"
                onClick={() => deleteEvidence({ id: row.original.id })}
              >
                <Trash2 className="mr-2 h-3.5 w-3.5" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ),
        enableSorting: false,
      },
    ],
    [handleView, verifyEvidence, deleteEvidence],
  );

  // ── KPI computation ─────────────────────────────────────────────────────
  const staleExpiredCount = (stats?.stale_count ?? 0) + (stats?.expired_count ?? 0);
  const controlsTotal =
    (stats?.controls_with_evidence ?? 0) + (stats?.controls_without_evidence ?? 0);
  const controlCoverageChange =
    controlsTotal > 0
      ? ((stats?.controls_with_evidence ?? 0) / controlsTotal) * 100 - 100
      : 0;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Audit Evidence Repository"
          description="Manage evidence collection, track compliance artifacts, and automate evidence gathering across frameworks."
          actions={
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  void refetch();
                  void refetchStats();
                }}
              >
                <RefreshCw className="mr-1.5 h-4 w-4" />
                Refresh
              </Button>
              <Button size="sm" onClick={handleUpload}>
                <Upload className="mr-1.5 h-4 w-4" />
                Upload Evidence
              </Button>
            </div>
          }
        />

        {/* KPI Stats Row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <KpiCard
            title="Total Evidence"
            value={stats?.total ?? 0}
            icon={Archive}
            iconColor="text-blue-600"
            loading={statsLoading}
            description={`${stats?.by_source?.manual ?? 0} manual, ${stats?.by_source?.automated ?? 0} automated`}
          />
          <KpiCard
            title="Needs Attention"
            value={staleExpiredCount}
            icon={AlertTriangle}
            iconColor="text-amber-600"
            loading={statsLoading}
            description={`${stats?.stale_count ?? 0} stale, ${stats?.expired_count ?? 0} expired`}
            className={staleExpiredCount > 0 ? 'border-amber-200' : ''}
          />
          <KpiCard
            title="Frameworks Covered"
            value={stats?.frameworks_covered ?? 0}
            icon={Shield}
            iconColor="text-emerald-600"
            loading={statsLoading}
          />
          <KpiCard
            title="Controls with Evidence"
            value={stats?.controls_with_evidence ?? 0}
            icon={Layers}
            iconColor="text-purple-600"
            loading={statsLoading}
            change={controlsTotal > 0 ? controlCoverageChange : undefined}
            changeLabel={controlsTotal > 0 ? `of ${controlsTotal}` : undefined}
          />
        </div>

        {/* Tabs */}
        <Tabs defaultValue="repository" className="space-y-4">
          <TabsList>
            <TabsTrigger value="repository">Evidence Repository</TabsTrigger>
            <TabsTrigger value="collection">Collection Status</TabsTrigger>
          </TabsList>

          {/* ── Evidence Repository Tab ──────────────────────────────────── */}
          <TabsContent value="repository" className="space-y-4">
            <DataTable
              columns={columns}
              filters={EVIDENCE_FILTERS}
              searchPlaceholder="Search evidence..."
              emptyState={{
                icon: FileText,
                title: 'No evidence found',
                description:
                  'Upload evidence or adjust your filters to see results.',
                action: {
                  label: 'Upload Evidence',
                  onClick: handleUpload,
                  icon: Plus,
                },
              }}
              getRowId={(row) => row.id}
              onRowClick={handleView}
              {...tableProps}
            />
          </TabsContent>

          {/* ── Collection Status Tab ────────────────────────────────────── */}
          <TabsContent value="collection" className="space-y-6">
            {statsLoading ? (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <LoadingSkeleton variant="card" />
                <LoadingSkeleton variant="card" />
              </div>
            ) : statsError || !stats ? (
              <ErrorState
                message="Failed to load evidence statistics."
                onRetry={() => void refetchStats()}
              />
            ) : (
              <>
                {/* Source Breakdown */}
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <Card>
                    <CardHeader className="pb-3">
                      <CardTitle className="text-sm font-semibold flex items-center gap-2">
                        <User className="h-4 w-4 text-muted-foreground" />
                        Manual Collection
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-3xl font-semibold tracking-tight">
                        {stats.by_source?.manual ?? 0}
                      </p>
                      <p className="text-sm text-muted-foreground mt-1">
                        Evidence items collected manually by team members
                      </p>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-3">
                      <CardTitle className="text-sm font-semibold flex items-center gap-2">
                        <Bot className="h-4 w-4 text-blue-600" />
                        Automated Collection
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-3xl font-semibold tracking-tight">
                        {stats.by_source?.automated ?? 0}
                      </p>
                      <p className="text-sm text-muted-foreground mt-1">
                        Evidence items gathered by automated collection pipelines
                      </p>
                    </CardContent>
                  </Card>
                </div>

                {/* Type Breakdown */}
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-semibold">Evidence by Type</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
                      {Object.entries(stats.by_type ?? {}).map(([type, count]) => (
                        <div
                          key={type}
                          className="flex items-center justify-between rounded-lg border p-3"
                        >
                          <div className="flex items-center gap-2">
                            <Badge
                              variant="secondary"
                              className={cn(
                                'text-xs capitalize',
                                TYPE_BADGE_CLASSES[type] ?? TYPE_BADGE_CLASSES.other,
                              )}
                            >
                              {titleCase(type)}
                            </Badge>
                          </div>
                          <span className="text-lg font-semibold tabular-nums">{count}</span>
                        </div>
                      ))}
                      {Object.keys(stats.by_type ?? {}).length === 0 && (
                        <p className="col-span-full text-sm text-muted-foreground text-center py-4">
                          No evidence data available
                        </p>
                      )}
                    </div>
                  </CardContent>
                </Card>

                {/* Controls Coverage */}
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-semibold">Control Evidence Coverage</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                      <div className="rounded-lg border border-green-200 bg-green-50 dark:bg-green-900/10 p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <CheckCircle className="h-5 w-5 text-green-600" />
                          <span className="text-sm font-medium text-green-800 dark:text-green-300">
                            Controls with Evidence
                          </span>
                        </div>
                        <p className="text-3xl font-semibold text-green-900 dark:text-green-200">
                          {stats.controls_with_evidence}
                        </p>
                      </div>
                      <div className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-900/10 p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <AlertTriangle className="h-5 w-5 text-red-600" />
                          <span className="text-sm font-medium text-red-800 dark:text-red-300">
                            Controls without Evidence
                          </span>
                        </div>
                        <p className="text-3xl font-semibold text-red-900 dark:text-red-200">
                          {stats.controls_without_evidence}
                        </p>
                      </div>
                    </div>
                    {controlsTotal > 0 && (
                      <div className="mt-4">
                        <div className="flex items-center justify-between text-sm mb-1.5">
                          <span className="text-muted-foreground">Coverage</span>
                          <span className="font-medium">
                            {((stats.controls_with_evidence / controlsTotal) * 100).toFixed(1)}%
                          </span>
                        </div>
                        <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                          <div
                            className="h-full rounded-full bg-green-500 transition-all"
                            style={{
                              width: `${(stats.controls_with_evidence / controlsTotal) * 100}%`,
                            }}
                          />
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* Stale/Expired Items */}
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-semibold flex items-center gap-2">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                      Items Requiring Attention ({staleExpiredCount})
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    {staleExpiredItems.length > 0 ? (
                      <div className="space-y-2 max-h-80 overflow-y-auto">
                        {staleExpiredItems.map((item) => (
                          <div
                            key={item.id}
                            className="flex items-center justify-between rounded-lg border p-3 cursor-pointer hover:bg-muted/50 transition-colors"
                            onClick={() => handleView(item)}
                          >
                            <div className="min-w-0 flex-1">
                              <p className="text-sm font-medium truncate">{item.title}</p>
                              <div className="flex items-center gap-2 mt-1">
                                <StatusBadge
                                  status={item.status}
                                  config={evidenceStatusConfig}
                                  size="sm"
                                />
                                <span className="text-xs text-muted-foreground">
                                  {titleCase(item.type)}
                                </span>
                                {item.expires_at && (
                                  <span className="text-xs text-muted-foreground">
                                    Expires {formatDate(item.expires_at)}
                                  </span>
                                )}
                              </div>
                            </div>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="shrink-0"
                              onClick={(e) => {
                                e.stopPropagation();
                                verifyEvidence({ id: item.id });
                              }}
                            >
                              <CheckCircle className="h-3.5 w-3.5 mr-1" />
                              Verify
                            </Button>
                          </div>
                        ))}
                      </div>
                    ) : staleExpiredCount > 0 ? (
                      <p className="text-sm text-muted-foreground py-4 text-center">
                        {staleExpiredCount} items need attention. Adjust table filters to view them.
                      </p>
                    ) : (
                      <p className="text-sm text-muted-foreground py-4 text-center">
                        All evidence items are current. No action required.
                      </p>
                    )}
                  </CardContent>
                </Card>
              </>
            )}
          </TabsContent>
        </Tabs>

        {/* Detail Panel */}
        <EvidenceDetailPanel
          evidence={selectedEvidence}
          open={detailOpen}
          onOpenChange={setDetailOpen}
          onVerified={() => {
            void refetch();
            void refetchStats();
          }}
        />

        {/* Form Dialog */}
        <EvidenceFormDialog
          open={formOpen}
          onOpenChange={setFormOpen}
          evidence={editEvidence}
        />
      </div>
    </PermissionRedirect>
  );
}
