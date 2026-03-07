'use client';

import { useState } from 'react';
import { AlertTriangle, Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { apiPost } from '@/lib/api';
import { showSuccess, showApiError } from '@/lib/toast';
import type { HumanTask } from '@/types/models';

interface TaskRejectDialogProps {
  task: HumanTask;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function TaskRejectDialog({
  task,
  open,
  onOpenChange,
  onSuccess,
}: TaskRejectDialogProps) {
  const [reason, setReason] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const canSubmit = reason.trim().length >= 10;

  const handleReject = async () => {
    if (!canSubmit) return;
    setIsSubmitting(true);
    try {
      await apiPost(`/api/v1/workflows/tasks/${task.id}/reject`, { reason: reason.trim() });
      showSuccess('Task rejected.');
      onOpenChange(false);
      setReason('');
      onSuccess();
    } catch {
      showApiError(new Error('Failed to reject task.'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenChange = (open: boolean) => {
    if (!open) setReason('');
    onOpenChange(open);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-destructive/10">
              <AlertTriangle className="h-5 w-5 text-destructive" />
            </div>
            <div>
              <DialogTitle>Reject Task</DialogTitle>
              <DialogDescription className="mt-0.5">
                Provide a reason for rejecting this task.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="reject-reason">
              Reason <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="reject-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Explain why you're rejecting this task..."
              className="min-h-[100px]"
              disabled={isSubmitting}
            />
            {reason.length > 0 && reason.trim().length < 10 && (
              <p className="text-xs text-destructive">
                Please provide at least 10 characters.
              </p>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            Rejecting this task will return it to the unassigned pool or escalate it.
          </p>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleReject}
            disabled={!canSubmit || isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Rejecting...
              </>
            ) : (
              'Reject Task'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
