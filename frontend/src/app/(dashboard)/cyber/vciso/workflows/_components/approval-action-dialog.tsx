'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type {
  VCISOApprovalRequest,
  ApprovalRequestStatus,
} from '@/types/cyber';

type ActionType = 'approve' | 'reject' | 'escalate';

interface ApprovalActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  approval: VCISOApprovalRequest;
  action: ActionType;
  onSuccess: () => void;
}

const ACTION_CONFIG: Record<
  ActionType,
  {
    title: string;
    description: (title: string) => string;
    confirmLabel: string;
    status: ApprovalRequestStatus;
    variant: 'default' | 'destructive' | 'outline';
  }
> = {
  approve: {
    title: 'Approve Request',
    description: (t) => `Approve the request "${t}"? This action cannot be undone.`,
    confirmLabel: 'Approve',
    status: 'approved',
    variant: 'default',
  },
  reject: {
    title: 'Reject Request',
    description: (t) => `Reject the request "${t}"? The requestor will be notified.`,
    confirmLabel: 'Reject',
    status: 'rejected',
    variant: 'destructive',
  },
  escalate: {
    title: 'Escalate Request',
    description: (t) =>
      `Escalate the request "${t}" to a higher authority? Provide your reasoning below.`,
    confirmLabel: 'Escalate',
    status: 'escalated',
    variant: 'default',
  },
};

export function ApprovalActionDialog({
  open,
  onOpenChange,
  approval,
  action,
  onSuccess,
}: ApprovalActionDialogProps) {
  const [decisionNotes, setDecisionNotes] = useState('');
  const config = ACTION_CONFIG[action];

  const mutation = useApiMutation<VCISOApprovalRequest, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_APPROVALS}/${approval.id}/decision`,
    {
      successMessage:
        action === 'approve'
          ? 'Request approved'
          : action === 'reject'
            ? 'Request rejected'
            : 'Request escalated',
      invalidateKeys: ['vciso-approvals'],
      onSuccess: () => {
        setDecisionNotes('');
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const handleSubmit = () => {
    if (!decisionNotes.trim()) {
      toast.error('Decision notes are required');
      return;
    }
    mutation.mutate({
      status: config.status,
      decision_notes: decisionNotes.trim(),
    });
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) {
      setDecisionNotes('');
    }
    onOpenChange(o);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{config.title}</DialogTitle>
          <DialogDescription>{config.description(approval.title)}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="action-decision-notes">
              Decision Notes <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="action-decision-notes"
              value={decisionNotes}
              onChange={(e) => setDecisionNotes(e.target.value)}
              placeholder="Provide reasoning for your decision..."
              rows={4}
              disabled={mutation.isPending}
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button
            variant={config.variant === 'destructive' ? 'destructive' : 'default'}
            onClick={handleSubmit}
            disabled={mutation.isPending || !decisionNotes.trim()}
          >
            {mutation.isPending ? 'Processing...' : config.confirmLabel}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
