'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import {
  User,
  Calendar,
  Clock,
  FileText,
  Link2,
  CheckCircle,
  XCircle,
  ArrowUpCircle,
} from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator, type Severity } from '@/components/shared/severity-indicator';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Separator } from '@/components/ui/separator';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { approvalStatusConfig } from '@/lib/status-configs';
import { formatDate, formatDateTime, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type {
  VCISOApprovalRequest,
  ApprovalRequestType,
} from '@/types/cyber';

interface ApprovalDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  approval: VCISOApprovalRequest;
  onActionComplete: () => void;
}

const TYPE_LABELS: Record<ApprovalRequestType, string> = {
  risk_acceptance: 'Risk Acceptance',
  policy_exception: 'Policy Exception',
  remediation: 'Remediation',
  budget: 'Budget',
  vendor_onboarding: 'Vendor Onboarding',
};

export function ApprovalDetailPanel({
  open,
  onOpenChange,
  approval,
  onActionComplete,
}: ApprovalDetailPanelProps) {
  const [decisionNotes, setDecisionNotes] = useState('');
  const isPending = approval.status === 'pending';
  const isOverdue = new Date(approval.deadline) < new Date();

  const approveMutation = useApiMutation<VCISOApprovalRequest, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_APPROVALS}/${approval.id}/decision`,
    {
      successMessage: 'Approval granted',
      invalidateKeys: ['vciso-approvals'],
      onSuccess: () => {
        setDecisionNotes('');
        onOpenChange(false);
        onActionComplete();
      },
    },
  );

  const rejectMutation = useApiMutation<VCISOApprovalRequest, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_APPROVALS}/${approval.id}/decision`,
    {
      successMessage: 'Request rejected',
      invalidateKeys: ['vciso-approvals'],
      onSuccess: () => {
        setDecisionNotes('');
        onOpenChange(false);
        onActionComplete();
      },
    },
  );

  const escalateMutation = useApiMutation<VCISOApprovalRequest, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_APPROVALS}/${approval.id}/decision`,
    {
      successMessage: 'Request escalated',
      invalidateKeys: ['vciso-approvals'],
      onSuccess: () => {
        setDecisionNotes('');
        onOpenChange(false);
        onActionComplete();
      },
    },
  );

  const isActioning =
    approveMutation.isPending ||
    rejectMutation.isPending ||
    escalateMutation.isPending;

  const handleApprove = () => {
    if (!decisionNotes.trim()) {
      toast.error('Decision notes are required');
      return;
    }
    approveMutation.mutate({
      status: 'approved',
      decision_notes: decisionNotes.trim(),
    });
  };

  const handleReject = () => {
    if (!decisionNotes.trim()) {
      toast.error('Decision notes are required');
      return;
    }
    rejectMutation.mutate({
      status: 'rejected',
      decision_notes: decisionNotes.trim(),
    });
  };

  const handleEscalate = () => {
    if (!decisionNotes.trim()) {
      toast.error('Decision notes are required');
      return;
    }
    escalateMutation.mutate({
      status: 'escalated',
      decision_notes: decisionNotes.trim(),
    });
  };

  return (
    <DetailPanel
      open={open}
      onOpenChange={(o) => {
        if (!o) setDecisionNotes('');
        onOpenChange(o);
      }}
      title={approval.title}
      description={TYPE_LABELS[approval.type] ?? titleCase(approval.type)}
      width="xl"
    >
      <div className="space-y-6">
        {/* Status & Priority */}
        <div className="flex items-center gap-3">
          <StatusBadge
            status={approval.status}
            config={approvalStatusConfig}
            size="lg"
          />
          <SeverityIndicator severity={approval.priority as Severity} />
        </div>

        <Separator />

        {/* Metadata */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Request Details
          </h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Type
              </p>
              <Badge variant="outline">
                {TYPE_LABELS[approval.type] ?? titleCase(approval.type)}
              </Badge>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Priority
              </p>
              <SeverityIndicator severity={approval.priority as Severity} />
            </div>
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Requested By
              </p>
              <div className="flex items-center gap-1.5 text-sm">
                <User className="h-3.5 w-3.5 text-muted-foreground" />
                {approval.requested_by_name}
              </div>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Approver
              </p>
              <div className="flex items-center gap-1.5 text-sm">
                <User className="h-3.5 w-3.5 text-muted-foreground" />
                {approval.approver_name}
              </div>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Deadline
              </p>
              <div className="flex items-center gap-1.5 text-sm">
                <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
                <span className={cn(isOverdue && isPending && 'text-red-600 font-medium')}>
                  {formatDate(approval.deadline)}
                  {isOverdue && isPending && ' (Overdue)'}
                </span>
              </div>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Created
              </p>
              <div className="flex items-center gap-1.5 text-sm">
                <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                {formatDateTime(approval.created_at)}
              </div>
            </div>
          </div>
        </div>

        <Separator />

        {/* Linked Entity */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Linked Entity
          </h3>
          <div className="flex items-center gap-2 rounded-lg border p-3">
            <Link2 className="h-4 w-4 text-muted-foreground shrink-0" />
            <div>
              <p className="text-sm font-medium">{titleCase(approval.linked_entity_type)}</p>
              <p className="text-xs text-muted-foreground font-mono">
                {approval.linked_entity_id}
              </p>
            </div>
          </div>
        </div>

        <Separator />

        {/* Description */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Description
          </h3>
          <p className="text-sm text-foreground leading-relaxed whitespace-pre-wrap">
            {approval.description}
          </p>
        </div>

        {/* Decision Info (if already decided) */}
        {approval.decided_at && approval.decision_notes && (
          <>
            <Separator />
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Decision
              </h3>
              <div
                className={cn(
                  'rounded-lg border p-4 space-y-2',
                  approval.status === 'approved' && 'border-green-200 bg-green-50',
                  approval.status === 'rejected' && 'border-red-200 bg-red-50',
                  approval.status === 'escalated' && 'border-purple-200 bg-purple-50',
                )}
              >
                <div className="flex items-center gap-2 text-sm font-medium">
                  {approval.status === 'approved' && (
                    <CheckCircle className="h-4 w-4 text-green-600" />
                  )}
                  {approval.status === 'rejected' && (
                    <XCircle className="h-4 w-4 text-red-600" />
                  )}
                  {approval.status === 'escalated' && (
                    <ArrowUpCircle className="h-4 w-4 text-purple-600" />
                  )}
                  <span>{titleCase(approval.status)}</span>
                </div>
                <p className="text-sm whitespace-pre-wrap">{approval.decision_notes}</p>
                <p className="text-xs text-muted-foreground">
                  Decided: {formatDateTime(approval.decided_at)}
                </p>
              </div>
            </div>
          </>
        )}

        {/* Decision Form (only for pending) */}
        {isPending && (
          <>
            <Separator />
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Make Decision
              </h3>
              <div className="space-y-2">
                <Label htmlFor="decision-notes">
                  Decision Notes <span className="text-destructive">*</span>
                </Label>
                <Textarea
                  id="decision-notes"
                  value={decisionNotes}
                  onChange={(e) => setDecisionNotes(e.target.value)}
                  placeholder="Provide reasoning for your decision..."
                  rows={4}
                  disabled={isActioning}
                />
              </div>
              <div className="flex items-center gap-2">
                <Button
                  onClick={handleApprove}
                  disabled={isActioning || !decisionNotes.trim()}
                  className="bg-green-600 hover:bg-green-700"
                >
                  <CheckCircle className="mr-1.5 h-4 w-4" />
                  {approveMutation.isPending ? 'Approving...' : 'Approve'}
                </Button>
                <Button
                  variant="destructive"
                  onClick={handleReject}
                  disabled={isActioning || !decisionNotes.trim()}
                >
                  <XCircle className="mr-1.5 h-4 w-4" />
                  {rejectMutation.isPending ? 'Rejecting...' : 'Reject'}
                </Button>
                <Button
                  variant="outline"
                  onClick={handleEscalate}
                  disabled={isActioning || !decisionNotes.trim()}
                >
                  <ArrowUpCircle className="mr-1.5 h-4 w-4" />
                  {escalateMutation.isPending ? 'Escalating...' : 'Escalate'}
                </Button>
              </div>
            </div>
          </>
        )}
      </div>
    </DetailPanel>
  );
}
