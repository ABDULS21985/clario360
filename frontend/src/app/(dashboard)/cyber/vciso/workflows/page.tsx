'use client';

import { useState, useMemo } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import {
  Plus,
  Eye,
  Users,
  CheckCircle,
  XCircle,
  ArrowUpCircle,
  Shield,
  Clock,
  ClipboardCheck,
  AlertTriangle,
  GitPullRequestArrow,
} from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator, type Severity } from '@/components/shared/severity-indicator';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, formatDateTime, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import { ownershipStatusConfig, approvalStatusConfig } from '@/lib/status-configs';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig, RowAction } from '@/types/table';
import type {
  VCISOControlOwnership,
  VCISOApprovalRequest,
  ApprovalRequestType,
} from '@/types/cyber';

import { OwnershipFormDialog } from './_components/ownership-form-dialog';
import { OwnershipDetailPanel } from './_components/ownership-detail-panel';
import { ApprovalDetailPanel } from './_components/approval-detail-panel';
import { ApprovalActionDialog } from './_components/approval-action-dialog';

// ── Constants ────────────────────────────────────────────────────────────────

const TYPE_LABELS: Record<ApprovalRequestType, string> = {
  risk_acceptance: 'Risk Acceptance',
  policy_exception: 'Policy Exception',
  remediation: 'Remediation',
  budget: 'Budget',
  vendor_onboarding: 'Vendor Onboarding',
};

// ── Ownership Filters ────────────────────────────────────────────────────────

const OWNERSHIP_FILTERS: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Assigned', value: 'assigned' },
      { label: 'Pending Review', value: 'pending_review' },
      { label: 'Reviewed', value: 'reviewed' },
    ],
  },
  {
    key: 'framework',
    label: 'Framework',
    type: 'select',
    options: [
      { label: 'NIST 800-53', value: 'NIST 800-53' },
      { label: 'ISO 27001', value: 'ISO 27001' },
      { label: 'CIS Controls', value: 'CIS Controls' },
      { label: 'SOC 2', value: 'SOC 2' },
      { label: 'PCI DSS', value: 'PCI DSS' },
      { label: 'HIPAA', value: 'HIPAA' },
    ],
  },
];

// ── Approval Filters ─────────────────────────────────────────────────────────

const APPROVAL_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'select',
    options: [
      { label: 'Risk Acceptance', value: 'risk_acceptance' },
      { label: 'Policy Exception', value: 'policy_exception' },
      { label: 'Remediation', value: 'remediation' },
      { label: 'Budget', value: 'budget' },
      { label: 'Vendor Onboarding', value: 'vendor_onboarding' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Pending', value: 'pending' },
      { label: 'Approved', value: 'approved' },
      { label: 'Rejected', value: 'rejected' },
      { label: 'Escalated', value: 'escalated' },
    ],
  },
  {
    key: 'priority',
    label: 'Priority',
    type: 'select',
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
    ],
  },
];

// ── Ownership Columns ────────────────────────────────────────────────────────

function getOwnershipColumns(): ColumnDef<VCISOControlOwnership>[] {
  return [
    {
      accessorKey: 'control_name',
      header: 'Control Name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-foreground">{row.original.control_name}</span>
      ),
    },
    {
      accessorKey: 'framework',
      header: 'Framework',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline" className="text-xs">
          {row.original.framework}
        </Badge>
      ),
    },
    {
      accessorKey: 'owner_name',
      header: 'Owner',
      enableSorting: true,
      cell: ({ row }) => <span className="text-sm">{row.original.owner_name}</span>,
    },
    {
      accessorKey: 'delegate_name',
      header: 'Delegate',
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.delegate_name || '\u2014'}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={ownershipStatusConfig} />
      ),
    },
    {
      accessorKey: 'last_reviewed_at',
      header: 'Last Reviewed',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.last_reviewed_at
            ? formatDate(row.original.last_reviewed_at)
            : 'Never'}
        </span>
      ),
    },
    {
      accessorKey: 'next_review_date',
      header: 'Next Review',
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
}

// ── Approval Columns ─────────────────────────────────────────────────────────

function getApprovalColumns(): ColumnDef<VCISOApprovalRequest>[] {
  return [
    {
      accessorKey: 'title',
      header: 'Title',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-foreground max-w-[240px] truncate block">
          {row.original.title}
        </span>
      ),
    },
    {
      accessorKey: 'type',
      header: 'Type',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline" className="text-xs">
          {TYPE_LABELS[row.original.type] ?? titleCase(row.original.type)}
        </Badge>
      ),
    },
    {
      accessorKey: 'priority',
      header: 'Priority',
      enableSorting: true,
      cell: ({ row }) => (
        <SeverityIndicator severity={row.original.priority as Severity} />
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={approvalStatusConfig} />
      ),
    },
    {
      accessorKey: 'requested_by_name',
      header: 'Requested By',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.requested_by_name}</span>
      ),
    },
    {
      accessorKey: 'approver_name',
      header: 'Approver',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.approver_name}</span>
      ),
    },
    {
      accessorKey: 'deadline',
      header: 'Deadline',
      enableSorting: true,
      cell: ({ row }) => {
        const isOverdue =
          new Date(row.original.deadline) < new Date() &&
          row.original.status === 'pending';
        return (
          <span
            className={cn(
              'text-sm',
              isOverdue && 'text-red-600 font-medium',
            )}
          >
            {formatDate(row.original.deadline)}
          </span>
        );
      },
    },
    {
      accessorKey: 'created_at',
      header: 'Created',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(row.original.created_at)}
        </span>
      ),
    },
  ];
}

// ── Approval KPI Stats ───────────────────────────────────────────────────────

interface ApprovalStats {
  pending: number;
  overdue: number;
  approved_this_month: number;
  rejected_this_month: number;
}

function ApprovalKpiCards() {
  const { data: allApprovals, isLoading } = useRealtimeData<
    PaginatedResponse<VCISOApprovalRequest>
  >(API_ENDPOINTS.CYBER_VCISO_APPROVALS, {
    params: { per_page: 500 },
    wsTopics: ['vciso.approvals'],
  });

  const stats = useMemo<ApprovalStats>(() => {
    const items = allApprovals?.data ?? [];
    const now = new Date();
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);

    let pending = 0;
    let overdue = 0;
    let approvedThisMonth = 0;
    let rejectedThisMonth = 0;

    for (const item of items) {
      if (item.status === 'pending') {
        pending++;
        if (new Date(item.deadline) < now) {
          overdue++;
        }
      }
      if (
        item.status === 'approved' &&
        item.decided_at &&
        new Date(item.decided_at) >= monthStart
      ) {
        approvedThisMonth++;
      }
      if (
        item.status === 'rejected' &&
        item.decided_at &&
        new Date(item.decided_at) >= monthStart
      ) {
        rejectedThisMonth++;
      }
    }

    return {
      pending,
      overdue,
      approved_this_month: approvedThisMonth,
      rejected_this_month: rejectedThisMonth,
    };
  }, [allApprovals]);

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <KpiCard
        title="Pending Approvals"
        value={stats.pending}
        icon={Clock}
        iconColor="text-yellow-600"
        loading={isLoading}
        description="Awaiting decision"
      />
      <KpiCard
        title="Overdue"
        value={stats.overdue}
        icon={AlertTriangle}
        iconColor="text-red-600"
        loading={isLoading}
        description="Deadline passed"
      />
      <KpiCard
        title="Approved This Month"
        value={stats.approved_this_month}
        icon={CheckCircle}
        iconColor="text-green-600"
        loading={isLoading}
        description="Granted this month"
      />
      <KpiCard
        title="Rejected This Month"
        value={stats.rejected_this_month}
        icon={XCircle}
        iconColor="text-red-600"
        loading={isLoading}
        description="Declined this month"
      />
    </div>
  );
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function VCISOWorkflowsPage() {
  const [activeTab, setActiveTab] = useState('ownership');

  // ── Ownership state ────────────────────────────────
  const [showOwnershipForm, setShowOwnershipForm] = useState(false);
  const [reassignTarget, setReassignTarget] = useState<VCISOControlOwnership | null>(null);
  const [selectedOwnership, setSelectedOwnership] = useState<VCISOControlOwnership | null>(null);
  const [markReviewedTarget, setMarkReviewedTarget] = useState<VCISOControlOwnership | null>(null);

  // ── Approval state ─────────────────────────────────
  const [selectedApproval, setSelectedApproval] = useState<VCISOApprovalRequest | null>(null);
  const [approvalAction, setApprovalAction] = useState<{
    approval: VCISOApprovalRequest;
    action: 'approve' | 'reject' | 'escalate';
  } | null>(null);

  // ── Ownership Table ────────────────────────────────
  const {
    tableProps: ownershipTableProps,
    refetch: refetchOwnership,
  } = useDataTable<VCISOControlOwnership>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOControlOwnership>>(
        API_ENDPOINTS.CYBER_VCISO_CONTROL_OWNERSHIP,
        params as unknown as Record<string, unknown>,
      ),
    queryKey: 'vciso-control-ownership',
    defaultSort: { column: 'next_review_date', direction: 'asc' },
    wsTopics: ['vciso.control-ownership'],
  });

  // ── Approval Table ─────────────────────────────────
  const {
    tableProps: approvalTableProps,
    refetch: refetchApprovals,
  } = useDataTable<VCISOApprovalRequest>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOApprovalRequest>>(
        API_ENDPOINTS.CYBER_VCISO_APPROVALS,
        params as unknown as Record<string, unknown>,
      ),
    queryKey: 'vciso-approvals',
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['vciso.approvals'],
  });

  // ── Mark Reviewed Mutation ─────────────────────────
  const markReviewedMutation = useApiMutation<VCISOControlOwnership, Record<string, unknown>>(
    'put',
    (variables) =>
      `${API_ENDPOINTS.CYBER_VCISO_CONTROL_OWNERSHIP}/${(variables as Record<string, string>).id}`,
    {
      successMessage: 'Control marked as reviewed',
      invalidateKeys: ['vciso-control-ownership'],
      onSuccess: () => {
        setMarkReviewedTarget(null);
        refetchOwnership();
      },
    },
  );

  // ── Columns ────────────────────────────────────────
  const ownershipColumns = useMemo(() => getOwnershipColumns(), []);
  const approvalColumns = useMemo(() => getApprovalColumns(), []);

  // ── Ownership Row Actions ──────────────────────────
  const ownershipRowActions: RowAction<VCISOControlOwnership>[] = [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (row) => setSelectedOwnership(row),
    },
    {
      label: 'Reassign',
      icon: Users,
      onClick: (row) => setReassignTarget(row),
    },
    {
      label: 'Mark Reviewed',
      icon: CheckCircle,
      onClick: (row) => setMarkReviewedTarget(row),
      hidden: (row) => row.status === 'reviewed',
    },
  ];

  // ── Approval Row Actions ───────────────────────────
  const approvalRowActions: RowAction<VCISOApprovalRequest>[] = [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (row) => setSelectedApproval(row),
    },
    {
      label: 'Approve',
      icon: CheckCircle,
      onClick: (row) => setApprovalAction({ approval: row, action: 'approve' }),
      hidden: (row) => row.status !== 'pending',
    },
    {
      label: 'Reject',
      icon: XCircle,
      variant: 'destructive',
      onClick: (row) => setApprovalAction({ approval: row, action: 'reject' }),
      hidden: (row) => row.status !== 'pending',
    },
    {
      label: 'Escalate',
      icon: ArrowUpCircle,
      onClick: (row) => setApprovalAction({ approval: row, action: 'escalate' }),
      hidden: (row) => row.status !== 'pending',
    },
  ];

  // ── Header actions based on tab ────────────────────
  const headerActions = useMemo(() => {
    if (activeTab === 'ownership') {
      return (
        <Button onClick={() => setShowOwnershipForm(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Assign Ownership
        </Button>
      );
    }
    return null;
  }, [activeTab]);

  const handleRefreshAll = () => {
    refetchOwnership();
    refetchApprovals();
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {/* Page Header */}
        <PageHeader
          title="Workflows"
          description="Manage control ownership assignments and approval workflows for governance decisions."
          actions={headerActions}
        />

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="ownership" className="gap-1.5">
              <Shield className="h-4 w-4" />
              Control Ownership
            </TabsTrigger>
            <TabsTrigger value="approvals" className="gap-1.5">
              <GitPullRequestArrow className="h-4 w-4" />
              Approval Queue
            </TabsTrigger>
          </TabsList>

          {/* ── Control Ownership Tab ─────────────────────── */}
          <TabsContent value="ownership" className="mt-6 space-y-4">
            <DataTable
              columns={ownershipColumns}
              filters={OWNERSHIP_FILTERS}
              rowActions={ownershipRowActions}
              searchPlaceholder="Search controls..."
              emptyState={{
                icon: Shield,
                title: 'No control ownership records',
                description:
                  'No controls have been assigned to owners yet. Start by assigning ownership.',
                action: {
                  label: 'Assign Ownership',
                  onClick: () => setShowOwnershipForm(true),
                  icon: Plus,
                },
              }}
              onRowClick={(row) => setSelectedOwnership(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...ownershipTableProps}
            />
          </TabsContent>

          {/* ── Approval Queue Tab ────────────────────────── */}
          <TabsContent value="approvals" className="mt-6 space-y-6">
            {/* KPI Row */}
            <ApprovalKpiCards />

            {/* Approval Table */}
            <DataTable
              columns={approvalColumns}
              filters={APPROVAL_FILTERS}
              rowActions={approvalRowActions}
              searchPlaceholder="Search approvals..."
              emptyState={{
                icon: ClipboardCheck,
                title: 'No approval requests',
                description:
                  'No approval requests have been submitted yet. Requests will appear here when created.',
              }}
              onRowClick={(row) => setSelectedApproval(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...approvalTableProps}
            />
          </TabsContent>
        </Tabs>
      </div>

      {/* ── Ownership Form Dialog (Create) ────────────── */}
      <OwnershipFormDialog
        open={showOwnershipForm}
        onOpenChange={setShowOwnershipForm}
        onSuccess={handleRefreshAll}
      />

      {/* ── Ownership Form Dialog (Reassign) ──────────── */}
      {reassignTarget && (
        <OwnershipFormDialog
          open={!!reassignTarget}
          onOpenChange={(o) => {
            if (!o) setReassignTarget(null);
          }}
          ownership={reassignTarget}
          onSuccess={handleRefreshAll}
        />
      )}

      {/* ── Ownership Detail Panel ────────────────────── */}
      {selectedOwnership && (
        <OwnershipDetailPanel
          open={!!selectedOwnership}
          onOpenChange={(o) => {
            if (!o) setSelectedOwnership(null);
          }}
          ownership={selectedOwnership}
          onReassign={() => {
            setReassignTarget(selectedOwnership);
            setSelectedOwnership(null);
          }}
          onMarkReviewed={() => {
            setMarkReviewedTarget(selectedOwnership);
            setSelectedOwnership(null);
          }}
        />
      )}

      {/* ── Mark Reviewed Confirm ─────────────────────── */}
      <ConfirmDialog
        open={!!markReviewedTarget}
        onOpenChange={(o) => {
          if (!o) setMarkReviewedTarget(null);
        }}
        title="Mark as Reviewed"
        description={`Mark the control "${markReviewedTarget?.control_name ?? ''}" as reviewed? This will update the review timestamp and set the status to reviewed.`}
        confirmLabel="Mark Reviewed"
        loading={markReviewedMutation.isPending}
        onConfirm={async () => {
          if (markReviewedTarget) {
            markReviewedMutation.mutate({
              id: markReviewedTarget.id,
              status: 'reviewed',
              last_reviewed_at: new Date().toISOString(),
            });
          }
        }}
      />

      {/* ── Approval Detail Panel ─────────────────────── */}
      {selectedApproval && (
        <ApprovalDetailPanel
          open={!!selectedApproval}
          onOpenChange={(o) => {
            if (!o) setSelectedApproval(null);
          }}
          approval={selectedApproval}
          onActionComplete={handleRefreshAll}
        />
      )}

      {/* ── Approval Action Dialog ────────────────────── */}
      {approvalAction && (
        <ApprovalActionDialog
          open={!!approvalAction}
          onOpenChange={(o) => {
            if (!o) setApprovalAction(null);
          }}
          approval={approvalAction.approval}
          action={approvalAction.action}
          onSuccess={handleRefreshAll}
        />
      )}
    </PermissionRedirect>
  );
}
