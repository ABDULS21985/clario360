'use client';

import { useState, useMemo } from 'react';
import {
  Plus,
  Building2,
  ClipboardList,
  Eye,
  Edit,
  Search,
  Send,
  CheckCircle,
} from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator, type Severity } from '@/components/shared/severity-indicator';
import { KpiCard } from '@/components/shared/kpi-card';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import {
  vendorStatusConfig,
  questionnaireStatusConfig,
} from '@/lib/status-configs';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { ColumnDef } from '@tanstack/react-table';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig } from '@/types/table';
import type {
  VCISOVendor,
  VCISOQuestionnaire,
  QuestionnaireStatus,
} from '@/types/cyber';

import { VendorDetailPanel } from './_components/vendor-detail-panel';
import { VendorFormDialog } from './_components/vendor-form-dialog';
import { QuestionnaireFormDialog } from './_components/questionnaire-form-dialog';

// ─── KPI Stats ────────────────────────────────────────────────────────────────

interface ThirdPartyStats {
  total_vendors: number;
  critical_vendors: number;
  pending_reviews: number;
  open_questionnaires: number;
}

function ThirdPartyKpiCards() {
  const { data: envelope, isLoading } = useRealtimeData<{ data: ThirdPartyStats }>(
    `${API_ENDPOINTS.CYBER_VCISO_VENDORS}/stats`,
    { wsTopics: ['vciso.vendors'], pollInterval: 60_000 },
  );
  const data = envelope?.data;

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <KpiCard
        title="Total Vendors"
        value={data?.total_vendors ?? 0}
        icon={Building2}
        iconColor="text-blue-600"
        loading={isLoading}
      />
      <KpiCard
        title="Critical Vendors"
        value={data?.critical_vendors ?? 0}
        icon={Building2}
        iconColor="text-red-600"
        loading={isLoading}
      />
      <KpiCard
        title="Pending Reviews"
        value={data?.pending_reviews ?? 0}
        icon={Search}
        iconColor="text-yellow-600"
        loading={isLoading}
      />
      <KpiCard
        title="Open Questionnaires"
        value={data?.open_questionnaires ?? 0}
        icon={ClipboardList}
        iconColor="text-orange-600"
        loading={isLoading}
      />
    </div>
  );
}

// ─── Risk Score Badge ─────────────────────────────────────────────────────────

function RiskScoreBadge({ score }: { score: number }) {
  let colorClass = 'text-green-700 bg-green-100';
  if (score > 60) colorClass = 'text-red-700 bg-red-100';
  else if (score > 30) colorClass = 'text-amber-700 bg-amber-100';

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold',
        colorClass,
      )}
    >
      {score}
    </span>
  );
}

// ─── Vendors Tab ──────────────────────────────────────────────────────────────

function VendorsTab({ onCreateVendor }: { onCreateVendor: () => void }) {
  const [detailVendor, setDetailVendor] = useState<VCISOVendor | null>(null);
  const [editVendor, setEditVendor] = useState<VCISOVendor | null>(null);
  const [confirmReview, setConfirmReview] = useState<VCISOVendor | null>(null);

  const table = useDataTable<VCISOVendor>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOVendor>>(API_ENDPOINTS.CYBER_VCISO_VENDORS, params),
    queryKey: 'vciso-vendors',
    defaultSort: { column: 'updated_at', direction: 'desc' },
    wsTopics: ['vciso.vendors'],
  });

  const reviewMutation = useApiMutation<VCISOVendor, { status: string }>(
    'put',
    () => `${API_ENDPOINTS.CYBER_VCISO_VENDORS}/${confirmReview?.id}/status`,
    {
      invalidateKeys: ['vciso-vendors'],
      successMessage: 'Review started successfully',
      onSuccess: () => {
        setConfirmReview(null);
        table.refetch();
      },
    },
  );

  const filters: FilterConfig[] = [
    {
      key: 'risk_tier',
      label: 'Risk Tier',
      type: 'select',
      options: [
        { label: 'Critical', value: 'critical' },
        { label: 'High', value: 'high' },
        { label: 'Medium', value: 'medium' },
        { label: 'Low', value: 'low' },
      ],
    },
    {
      key: 'status',
      label: 'Status',
      type: 'select',
      options: [
        { label: 'Active', value: 'active' },
        { label: 'Onboarding', value: 'onboarding' },
        { label: 'Under Review', value: 'under_review' },
        { label: 'Offboarding', value: 'offboarding' },
        { label: 'Terminated', value: 'terminated' },
      ],
    },
  ];

  const columns: ColumnDef<VCISOVendor>[] = [
    {
      id: 'name',
      header: 'Name',
      accessorKey: 'name',
      enableSorting: true,
      cell: ({ row }) => (
        <button
          className="font-semibold text-sm hover:underline text-left max-w-[120px] sm:max-w-[200px] truncate block"
          onClick={(e) => {
            e.stopPropagation();
            setDetailVendor(row.original);
          }}
        >
          {row.original.name}
        </button>
      ),
    },
    {
      id: 'category',
      header: 'Category',
      accessorKey: 'category',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline">{row.original.category}</Badge>
      ),
    },
    {
      id: 'risk_tier',
      header: 'Risk Tier',
      accessorKey: 'risk_tier',
      enableSorting: true,
      cell: ({ row }) => (
        <SeverityIndicator
          severity={row.original.risk_tier as Severity}
          showLabel
          size="sm"
        />
      ),
    },
    {
      id: 'status',
      header: 'Status',
      accessorKey: 'status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge
          status={row.original.status}
          config={vendorStatusConfig}
          size="sm"
        />
      ),
    },
    {
      id: 'risk_score',
      header: 'Risk Score',
      accessorKey: 'risk_score',
      enableSorting: true,
      cell: ({ row }) => <RiskScoreBadge score={row.original.risk_score} />,
    },
    {
      id: 'controls',
      header: 'Controls',
      accessorKey: 'controls_met',
      enableSorting: false,
      cell: ({ row }) => {
        const { controls_met, controls_total } = row.original;
        const pct = controls_total > 0 ? Math.round((controls_met / controls_total) * 100) : 0;
        return (
          <div className="flex items-center gap-2 min-w-[120px]">
            <Progress value={pct} className="h-1.5 w-16" />
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {controls_met}/{controls_total}
            </span>
          </div>
        );
      },
    },
    {
      id: 'open_findings',
      header: 'Findings',
      accessorKey: 'open_findings',
      enableSorting: true,
      cell: ({ row }) => {
        const count = row.original.open_findings;
        if (count > 0) {
          return (
            <Badge variant="destructive" className="text-xs">
              {count}
            </Badge>
          );
        }
        return <span className="text-xs text-muted-foreground">0</span>;
      },
    },
    {
      id: 'next_review_date',
      header: 'Next Review',
      accessorKey: 'next_review_date',
      enableSorting: true,
      cell: ({ row }) => {
        const isOverdue = new Date(row.original.next_review_date) < new Date();
        return (
          <span
            className={cn(
              'text-sm',
              isOverdue && 'text-red-600 font-medium',
            )}
          >
            {formatDate(row.original.next_review_date)}
          </span>
        );
      },
    },
  ];

  const rowActions = (vendor: VCISOVendor) => [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (v: VCISOVendor) => setDetailVendor(v),
    },
    {
      label: 'Edit',
      icon: Edit,
      onClick: (v: VCISOVendor) => setEditVendor(v),
    },
    {
      label: 'Start Review',
      icon: Search,
      onClick: (v: VCISOVendor) => setConfirmReview(v),
    },
  ];

  return (
    <>
      <DataTable
        {...table.tableProps}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        onRowClick={(vendor) => setDetailVendor(vendor)}
        searchPlaceholder="Search vendors..."
        searchSlot={
          <SearchInput
            value={table.tableProps.searchValue ?? ''}
            onChange={table.tableProps.onSearchChange ?? (() => undefined)}
            placeholder="Search vendors..."
            loading={table.tableProps.isLoading}
          />
        }
        emptyState={{
          icon: Building2,
          title: 'No vendors found',
          description: 'Add your first third-party vendor to start tracking risk.',
          action: {
            label: 'Add Vendor',
            onClick: onCreateVendor,
            icon: Plus,
          },
        }}
      />

      {/* Detail Panel */}
      {detailVendor && (
        <VendorDetailPanel
          open={!!detailVendor}
          onOpenChange={(o) => !o && setDetailVendor(null)}
          vendor={detailVendor}
          onUpdated={() => {
            table.refetch();
            setDetailVendor(null);
          }}
        />
      )}

      {/* Edit Dialog */}
      <VendorFormDialog
        open={!!editVendor}
        onOpenChange={(o) => !o && setEditVendor(null)}
        vendor={editVendor}
        onSuccess={() => table.refetch()}
      />

      {/* Confirm Start Review */}
      {confirmReview && (
        <ConfirmDialog
          open={!!confirmReview}
          onOpenChange={(o) => !o && setConfirmReview(null)}
          title="Start Review"
          description={`Place "${confirmReview.name}" under review? The vendor status will change to "Under Review".`}
          confirmLabel="Start Review"
          onConfirm={() => reviewMutation.mutate({ status: 'under_review' })}
          loading={reviewMutation.isPending}
        />
      )}
    </>
  );
}

// ─── Questionnaires Tab ───────────────────────────────────────────────────────

function QuestionnairesTab({
  onCreateQuestionnaire,
}: {
  onCreateQuestionnaire: () => void;
}) {
  const [confirmAction, setConfirmAction] = useState<{
    questionnaire: VCISOQuestionnaire;
    type: 'send' | 'complete';
    title: string;
    description: string;
  } | null>(null);

  const table = useDataTable<VCISOQuestionnaire>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOQuestionnaire>>(
        API_ENDPOINTS.CYBER_VCISO_QUESTIONNAIRES,
        params,
      ),
    queryKey: 'vciso-questionnaires',
    defaultSort: { column: 'updated_at', direction: 'desc' },
    wsTopics: ['vciso.questionnaires'],
  });

  const statusMutation = useApiMutation<
    VCISOQuestionnaire,
    { status: QuestionnaireStatus }
  >(
    'put',
    () =>
      `${API_ENDPOINTS.CYBER_VCISO_QUESTIONNAIRES}/${confirmAction?.questionnaire.id}/status`,
    {
      invalidateKeys: ['vciso-questionnaires'],
      onSuccess: () => {
        toast.success(
          confirmAction?.type === 'send'
            ? 'Questionnaire sent successfully'
            : 'Questionnaire marked as completed',
        );
        setConfirmAction(null);
        table.refetch();
      },
    },
  );

  const filters: FilterConfig[] = [
    {
      key: 'type',
      label: 'Type',
      type: 'select',
      options: [
        { label: 'Vendor', value: 'vendor' },
        { label: 'Customer', value: 'customer' },
        { label: 'Audit', value: 'audit' },
        { label: 'Internal', value: 'internal' },
      ],
    },
    {
      key: 'status',
      label: 'Status',
      type: 'select',
      options: [
        { label: 'Draft', value: 'draft' },
        { label: 'Sent', value: 'sent' },
        { label: 'In Progress', value: 'in_progress' },
        { label: 'Completed', value: 'completed' },
        { label: 'Expired', value: 'expired' },
      ],
    },
  ];

  const columns: ColumnDef<VCISOQuestionnaire>[] = [
    {
      id: 'title',
      header: 'Title',
      accessorKey: 'title',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-semibold text-sm max-w-[140px] sm:max-w-[220px] truncate block">
          {row.original.title}
        </span>
      ),
    },
    {
      id: 'type',
      header: 'Type',
      accessorKey: 'type',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline">{titleCase(row.original.type)}</Badge>
      ),
    },
    {
      id: 'vendor_name',
      header: 'Vendor',
      accessorKey: 'vendor_name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.vendor_name || <span className="text-muted-foreground">--</span>}
        </span>
      ),
    },
    {
      id: 'status',
      header: 'Status',
      accessorKey: 'status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge
          status={row.original.status}
          config={questionnaireStatusConfig}
          size="sm"
        />
      ),
    },
    {
      id: 'progress',
      header: 'Progress',
      accessorKey: 'answered_questions',
      enableSorting: false,
      cell: ({ row }) => {
        const { answered_questions, total_questions } = row.original;
        const pct =
          total_questions > 0
            ? Math.round((answered_questions / total_questions) * 100)
            : 0;
        return (
          <div className="flex items-center gap-2 min-w-[120px]">
            <Progress value={pct} className="h-1.5 w-16" />
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {answered_questions}/{total_questions}
            </span>
          </div>
        );
      },
    },
    {
      id: 'score',
      header: 'Score',
      accessorKey: 'score',
      enableSorting: true,
      cell: ({ row }) => {
        const score = row.original.score;
        if (score == null) {
          return <span className="text-xs text-muted-foreground">--</span>;
        }
        return <RiskScoreBadge score={score} />;
      },
    },
    {
      id: 'due_date',
      header: 'Due Date',
      accessorKey: 'due_date',
      enableSorting: true,
      cell: ({ row }) => {
        const isOverdue =
          new Date(row.original.due_date) < new Date() &&
          row.original.status !== 'completed';
        return (
          <span
            className={cn(
              'text-sm',
              isOverdue && 'text-red-600 font-medium',
            )}
          >
            {formatDate(row.original.due_date)}
          </span>
        );
      },
    },
    {
      id: 'assigned_to_name',
      header: 'Assigned To',
      accessorKey: 'assigned_to_name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.assigned_to_name || (
            <span className="text-muted-foreground">Unassigned</span>
          )}
        </span>
      ),
    },
  ];

  const rowActions = (questionnaire: VCISOQuestionnaire) => {
    const actions: {
      label: string;
      icon: typeof Eye;
      onClick: (q: VCISOQuestionnaire) => void;
    }[] = [
      {
        label: 'View',
        icon: Eye,
        onClick: () => {},
      },
    ];

    if (questionnaire.status === 'draft') {
      actions.push({
        label: 'Send',
        icon: Send,
        onClick: (q: VCISOQuestionnaire) =>
          setConfirmAction({
            questionnaire: q,
            type: 'send',
            title: 'Send Questionnaire',
            description: `Send "${q.title}" to the assigned recipient? They will be notified via email.`,
          }),
      });
    }

    if (
      questionnaire.status === 'in_progress' ||
      questionnaire.status === 'sent'
    ) {
      actions.push({
        label: 'Complete',
        icon: CheckCircle,
        onClick: (q: VCISOQuestionnaire) =>
          setConfirmAction({
            questionnaire: q,
            type: 'complete',
            title: 'Mark as Completed',
            description: `Mark "${q.title}" as completed? This will finalize the questionnaire results.`,
          }),
      });
    }

    return actions;
  };

  const handleStatusChange = () => {
    if (!confirmAction) return;
    const statusMap: Record<string, QuestionnaireStatus> = {
      send: 'sent',
      complete: 'completed',
    };
    statusMutation.mutate({ status: statusMap[confirmAction.type] });
  };

  return (
    <>
      <DataTable
        {...table.tableProps}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        searchPlaceholder="Search questionnaires..."
        searchSlot={
          <SearchInput
            value={table.tableProps.searchValue ?? ''}
            onChange={table.tableProps.onSearchChange ?? (() => undefined)}
            placeholder="Search questionnaires..."
            loading={table.tableProps.isLoading}
          />
        }
        emptyState={{
          icon: ClipboardList,
          title: 'No questionnaires found',
          description: 'Create your first questionnaire to assess vendor security.',
          action: {
            label: 'Create Questionnaire',
            onClick: onCreateQuestionnaire,
            icon: Plus,
          },
        }}
      />

      {/* Confirm Send / Complete */}
      {confirmAction && (
        <ConfirmDialog
          open={!!confirmAction}
          onOpenChange={(o) => !o && setConfirmAction(null)}
          title={confirmAction.title}
          description={confirmAction.description}
          confirmLabel={confirmAction.title}
          onConfirm={handleStatusChange}
          loading={statusMutation.isPending}
        />
      )}
    </>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function ThirdPartyRiskPage() {
  const [activeTab, setActiveTab] = useState('vendors');
  const [createVendorOpen, setCreateVendorOpen] = useState(false);
  const [createQuestionnaireOpen, setCreateQuestionnaireOpen] = useState(false);

  const headerActions = useMemo(() => {
    if (activeTab === 'vendors') {
      return (
        <Button onClick={() => setCreateVendorOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Vendor
        </Button>
      );
    }
    if (activeTab === 'questionnaires') {
      return (
        <Button onClick={() => setCreateQuestionnaireOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Questionnaire
        </Button>
      );
    }
    return null;
  }, [activeTab]);

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Third-Party Risk Management"
          description="Monitor vendor risk, manage assessments, and track security questionnaires."
          actions={headerActions}
        />

        <ThirdPartyKpiCards />

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="vendors" className="gap-1.5">
              <Building2 className="h-4 w-4" />
              Vendors
            </TabsTrigger>
            <TabsTrigger value="questionnaires" className="gap-1.5">
              <ClipboardList className="h-4 w-4" />
              Questionnaires
            </TabsTrigger>
          </TabsList>

          <TabsContent value="vendors" className="mt-6">
            <VendorsTab onCreateVendor={() => setCreateVendorOpen(true)} />
          </TabsContent>

          <TabsContent value="questionnaires" className="mt-6">
            <QuestionnairesTab
              onCreateQuestionnaire={() => setCreateQuestionnaireOpen(true)}
            />
          </TabsContent>
        </Tabs>

        {/* Create Vendor Dialog */}
        <VendorFormDialog
          open={createVendorOpen}
          onOpenChange={setCreateVendorOpen}
          onSuccess={() => {}}
        />

        {/* Create Questionnaire Dialog */}
        <QuestionnaireFormDialog
          open={createQuestionnaireOpen}
          onOpenChange={setCreateQuestionnaireOpen}
          onSuccess={() => {}}
        />
      </div>
    </PermissionRedirect>
  );
}
