'use client';

import { useState } from 'react';
import { type ColumnDef, type Row } from '@tanstack/react-table';
import {
  CheckCircle2,
  Clock,
  FileWarning,
  Plus,
  ShieldOff,
  XCircle,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type {
  DSPMRiskException,
  DSPMExceptionType,
  DSPMApprovalStatus,
} from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';

const EXCEPTION_TYPE_COLORS: Record<string, string> = {
  posture_finding: 'bg-amber-100 text-amber-800',
  policy_violation: 'bg-red-100 text-red-700',
  overprivileged_access: 'bg-purple-100 text-purple-700',
  exposure_risk: 'bg-orange-100 text-orange-700',
  encryption_gap: 'bg-blue-100 text-blue-700',
};

const APPROVAL_STATUS_COLORS: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-800',
  approved: 'bg-green-100 text-green-700',
  rejected: 'bg-red-100 text-red-700',
  expired: 'bg-gray-100 text-gray-600',
};

const STATUS_COLORS: Record<string, string> = {
  active: 'bg-green-100 text-green-700',
  expired: 'bg-gray-100 text-gray-600',
  revoked: 'bg-red-100 text-red-700',
  superseded: 'bg-blue-100 text-blue-700',
};

const EXCEPTION_TYPES: DSPMExceptionType[] = [
  'posture_finding', 'policy_violation', 'overprivileged_access', 'exposure_risk', 'encryption_gap',
];

function buildExceptionColumns(
  onApprove: (id: string) => Promise<void>,
  onReject: (id: string) => Promise<void>,
): ColumnDef<DSPMRiskException>[] {
  return [
    {
      id: 'exception_type',
      accessorKey: 'exception_type',
      header: 'Type',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => {
        const t = row.original.exception_type;
        return (
          <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${EXCEPTION_TYPE_COLORS[t] ?? 'bg-muted text-muted-foreground'}`}>
            {t.replace(/_/g, ' ')}
          </span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'justification',
      accessorKey: 'justification',
      header: 'Justification',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => (
        <div>
          <p className="text-sm line-clamp-1">{row.original.justification}</p>
          {row.original.data_asset_id && (
            <p className="mt-0.5 text-xs text-muted-foreground">Asset: {row.original.data_asset_id.slice(0, 8)}...</p>
          )}
          {row.original.policy_id && (
            <p className="mt-0.5 text-xs text-muted-foreground">Policy: {row.original.policy_id.slice(0, 8)}...</p>
          )}
        </div>
      ),
    },
    {
      id: 'risk_level',
      accessorKey: 'risk_level',
      header: 'Risk Level',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => {
        const level = row.original.risk_level;
        const color = level === 'critical' ? 'text-red-600' : level === 'high' ? 'text-orange-600' : level === 'medium' ? 'text-amber-600' : 'text-blue-600';
        return (
          <span className={`text-sm font-medium capitalize ${color}`}>{level}</span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'requested_by',
      accessorKey: 'requested_by',
      header: 'Requested By',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => (
        <span className="text-sm text-muted-foreground">{row.original.requested_by}</span>
      ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => {
        const status = row.original.status;
        return (
          <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${STATUS_COLORS[status] ?? 'bg-muted text-muted-foreground'}`}>
            {status}
          </span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'approval_status',
      accessorKey: 'approval_status',
      header: 'Approval',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => {
        const approval = row.original.approval_status;
        return (
          <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${APPROVAL_STATUS_COLORS[approval] ?? 'bg-muted text-muted-foreground'}`}>
            {approval}
          </span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'expires_at',
      accessorKey: 'expires_at',
      header: 'Expires',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => {
        const dt = row.original.expires_at;
        const isExpired = new Date(dt) < new Date();
        return (
          <span className={`text-xs ${isExpired ? 'text-red-600' : 'text-muted-foreground'}`}>
            {new Date(dt).toLocaleDateString()}
          </span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'review_count',
      accessorKey: 'review_count',
      header: 'Reviews',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => (
        <span className="text-sm tabular-nums">{row.original.review_count}</span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<DSPMRiskException> }) => {
        const exception = row.original;
        if (exception.approval_status !== 'pending') return null;
        return (
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs text-green-600 hover:text-green-700"
              onClick={(e) => {
                e.stopPropagation();
                void onApprove(exception.id);
              }}
            >
              Approve
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs text-red-600 hover:text-red-700"
              onClick={(e) => {
                e.stopPropagation();
                void onReject(exception.id);
              }}
            >
              Reject
            </Button>
          </div>
        );
      },
    },
  ];
}

interface ExceptionForm {
  exception_type: DSPMExceptionType;
  justification: string;
  business_reason: string;
  compensating_controls: string;
  data_asset_id: string;
  policy_id: string;
  remediation_id: string;
  expires_at: string;
  risk_score: string;
}

const INITIAL_FORM: ExceptionForm = {
  exception_type: 'posture_finding',
  justification: '',
  business_reason: '',
  compensating_controls: '',
  data_asset_id: '',
  policy_id: '',
  remediation_id: '',
  expires_at: '',
  risk_score: '50',
};

export default function RiskExceptionsPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState<ExceptionForm>(INITIAL_FORM);
  const [creating, setCreating] = useState(false);

  const { tableProps, refetch, data } = useDataTable<DSPMRiskException>({
    queryKey: 'cyber-dspm-exceptions',
    fetchFn: (params: FetchParams) => {
      const { filters, ...rest } = params;
      return apiGet<PaginatedResponse<DSPMRiskException>>(API_ENDPOINTS.CYBER_DSPM_EXCEPTIONS, { ...rest, ...filters } as Record<string, unknown>);
    },
    defaultSort: { column: 'created_at', direction: 'desc' },
  });

  async function handleApprove(exceptionId: string) {
    try {
      await apiPost(API_ENDPOINTS.CYBER_DSPM_EXCEPTIONS + '/' + exceptionId + '/approve');
      toast.success('Exception approved');
      refetch();
    } catch {
      toast.error('Failed to approve exception');
    }
  }

  async function handleReject(exceptionId: string) {
    try {
      await apiPost(API_ENDPOINTS.CYBER_DSPM_EXCEPTIONS + '/' + exceptionId + '/reject');
      toast.success('Exception rejected');
      refetch();
    } catch {
      toast.error('Failed to reject exception');
    }
  }

  const exceptionColumns = buildExceptionColumns(handleApprove, handleReject);

  const totalExceptions = tableProps.totalRows;
  const pendingCount = data.filter((e) => e.approval_status === 'pending').length;
  const approvedCount = data.filter((e) => e.approval_status === 'approved').length;
  const expiredCount = data.filter((e) => e.status === 'expired').length;

  const filters = [
    {
      key: 'approval_status',
      label: 'Approval Status',
      type: 'multi-select' as const,
      options: ['pending', 'approved', 'rejected', 'expired'].map((s) => ({
        label: s.charAt(0).toUpperCase() + s.slice(1),
        value: s,
      })),
    },
    {
      key: 'exception_type',
      label: 'Exception Type',
      type: 'multi-select' as const,
      options: EXCEPTION_TYPES.map((t) => ({
        label: t.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase()),
        value: t,
      })),
    },
    {
      key: 'status',
      label: 'Status',
      type: 'multi-select' as const,
      options: ['active', 'expired', 'revoked', 'superseded'].map((s) => ({
        label: s.charAt(0).toUpperCase() + s.slice(1),
        value: s,
      })),
    },
  ];

  async function handleCreateException() {
    if (!form.justification.trim()) {
      toast.error('Justification is required');
      return;
    }
    if (!form.expires_at) {
      toast.error('Expiration date is required');
      return;
    }
    setCreating(true);
    try {
      await apiPost(API_ENDPOINTS.CYBER_DSPM_EXCEPTIONS, {
        exception_type: form.exception_type,
        justification: form.justification,
        business_reason: form.business_reason || undefined,
        compensating_controls: form.compensating_controls || undefined,
        data_asset_id: form.data_asset_id || undefined,
        policy_id: form.policy_id || undefined,
        remediation_id: form.remediation_id || undefined,
        expires_at: new Date(form.expires_at).toISOString(),
        risk_score: parseInt(form.risk_score, 10) || 50,
      });
      toast.success('Exception request submitted');
      setCreateOpen(false);
      setForm(INITIAL_FORM);
      refetch();
    } catch {
      toast.error('Failed to create exception request');
    } finally {
      setCreating(false);
    }
  }

  const kpis = [
    { label: 'Total Exceptions', value: totalExceptions, icon: ShieldOff, color: 'text-blue-600' },
    { label: 'Pending Review', value: pendingCount, icon: Clock, color: 'text-amber-600' },
    { label: 'Approved', value: approvedCount, icon: CheckCircle2, color: 'text-green-600' },
    { label: 'Expired', value: expiredCount, icon: XCircle, color: 'text-gray-500' },
  ];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Risk Exceptions"
          description="Manage risk acceptance exceptions with approval workflows and periodic reviews"
          actions={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Request Exception
            </Button>
          }
        />

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {kpis.map((kpi) => {
            const Icon = kpi.icon;
            return (
              <Card key={kpi.label}>
                <CardContent className="flex items-center gap-4 p-5">
                  <Icon className={`h-5 w-5 ${kpi.color}`} />
                  <div>
                    <p className="text-xs text-muted-foreground">{kpi.label}</p>
                    <p className="text-2xl font-bold tabular-nums">{kpi.value}</p>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>

        <div className="rounded-xl border bg-card">
          <div className="border-b px-5 py-4">
            <h3 className="text-sm font-semibold">Exception Registry</h3>
            <p className="text-xs text-muted-foreground">All risk exceptions with their approval and review status</p>
          </div>
          <div className="p-5">
            {tableProps.isLoading ? (
              <LoadingSkeleton variant="table-row" count={6} />
            ) : tableProps.error ? (
              <ErrorState message="Failed to load exceptions" onRetry={refetch} />
            ) : (
              <DataTable
                {...tableProps}
                columns={exceptionColumns}
                filters={filters}
                onSortChange={() => undefined}
                searchPlaceholder="Search exceptions..."
                emptyState={{
                  icon: FileWarning,
                  title: 'No exceptions found',
                  description: 'No risk exceptions have been requested yet.',
                  action: { label: 'Request Exception', onClick: () => setCreateOpen(true) },
                }}
              />
            )}
          </div>
        </div>

        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Request Risk Exception</DialogTitle>
              <DialogDescription>
                Submit a risk acceptance exception for review and approval.
              </DialogDescription>
            </DialogHeader>
            <div className="max-h-[60vh] space-y-4 overflow-y-auto py-2">
              <div className="space-y-2">
                <Label>Exception Type</Label>
                <Select value={form.exception_type} onValueChange={(v) => setForm({ ...form, exception_type: v as DSPMExceptionType })}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {EXCEPTION_TYPES.map((t) => (
                      <SelectItem key={t} value={t}>
                        {t.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase())}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="exc-justification">Justification</Label>
                <Input
                  id="exc-justification"
                  placeholder="Why is this exception needed?"
                  value={form.justification}
                  onChange={(e) => setForm({ ...form, justification: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="exc-business-reason">Business Reason</Label>
                <Input
                  id="exc-business-reason"
                  placeholder="Business impact or justification"
                  value={form.business_reason}
                  onChange={(e) => setForm({ ...form, business_reason: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="exc-controls">Compensating Controls</Label>
                <Input
                  id="exc-controls"
                  placeholder="What mitigations are in place?"
                  value={form.compensating_controls}
                  onChange={(e) => setForm({ ...form, compensating_controls: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="exc-asset-id">Data Asset ID</Label>
                  <Input
                    id="exc-asset-id"
                    placeholder="Optional"
                    value={form.data_asset_id}
                    onChange={(e) => setForm({ ...form, data_asset_id: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="exc-policy-id">Policy ID</Label>
                  <Input
                    id="exc-policy-id"
                    placeholder="Optional"
                    value={form.policy_id}
                    onChange={(e) => setForm({ ...form, policy_id: e.target.value })}
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="exc-remediation-id">Remediation ID</Label>
                  <Input
                    id="exc-remediation-id"
                    placeholder="Optional"
                    value={form.remediation_id}
                    onChange={(e) => setForm({ ...form, remediation_id: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="exc-risk-score">Risk Score</Label>
                  <Input
                    id="exc-risk-score"
                    type="number"
                    min="0"
                    max="100"
                    value={form.risk_score}
                    onChange={(e) => setForm({ ...form, risk_score: e.target.value })}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="exc-expires">Expires At</Label>
                <Input
                  id="exc-expires"
                  type="date"
                  value={form.expires_at}
                  onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreateException} disabled={creating}>
                {creating ? 'Submitting...' : 'Submit Request'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </PermissionRedirect>
  );
}
