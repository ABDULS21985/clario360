'use client';

import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { RotateCcw, AlertTriangle } from 'lucide-react';
import type { RemediationAction } from '@/types/cyber';

interface RollbackDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  action: RemediationAction;
  onSuccess?: () => void;
}

export function RollbackDialog({ open, onOpenChange, action, onSuccess }: RollbackDialogProps) {
  const [confirm, setConfirm] = useState('');
  const [reason, setReason] = useState('');

  const { mutate, isPending } = useApiMutation<void, { reason: string }>(
    'post',
    `${API_ENDPOINTS.CYBER_REMEDIATION}/${action.id}/rollback`,
    {
      successMessage: 'Rollback initiated',
      invalidateKeys: ['cyber-remediation', `cyber-remediation-${action.id}`],
      onSuccess: () => {
        setConfirm('');
        setReason('');
        onOpenChange(false);
        onSuccess?.();
      },
    },
  );

  const handleClose = () => {
    setConfirm('');
    setReason('');
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-orange-600">
            <RotateCcw className="h-5 w-5" />
            Rollback Remediation
          </DialogTitle>
          <DialogDescription>
            Rolling back <strong>{action.title}</strong> will attempt to restore pre-execution state. This action requires elevated confirmation.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-start gap-3 rounded-xl border border-orange-200 bg-orange-50/50 p-3 dark:border-orange-900 dark:bg-orange-950/20">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-orange-600" />
            <p className="text-xs text-orange-800 dark:text-orange-300">
              Rollback may cause temporary service disruption. Ensure a maintenance window is in place before proceeding.
            </p>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="rollback-reason">Reason for Rollback</Label>
            <Textarea
              id="rollback-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Describe why this remediation needs to be rolled back…"
              rows={3}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="rollback-confirm">
              Type <strong>ROLLBACK</strong> to confirm
            </Label>
            <Input
              id="rollback-confirm"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              placeholder="ROLLBACK"
              className="font-mono"
            />
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>Cancel</Button>
          <Button
            type="button"
            disabled={confirm !== 'ROLLBACK' || !reason.trim() || isPending}
            onClick={() => mutate({ reason })}
            className="bg-orange-600 text-white hover:bg-orange-700"
          >
            <RotateCcw className="mr-1.5 h-4 w-4" />
            {isPending ? 'Rolling Back…' : 'Confirm Rollback'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
