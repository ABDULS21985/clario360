'use client';

import { useState } from 'react';
import { CheckCircle, Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { apiPost } from '@/lib/api';
import { showSuccess, showError } from '@/lib/toast';
import type { HumanTask } from '@/types/models';

interface TaskCompleteDialogProps {
  task: HumanTask;
  formData: Record<string, unknown>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function TaskCompleteDialog({
  task,
  formData,
  open,
  onOpenChange,
  onSuccess,
}: TaskCompleteDialogProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleComplete = async () => {
    setIsSubmitting(true);
    try {
      await apiPost(`/api/v1/workflows/tasks/${task.id}/complete`, { form_data: formData });
      showSuccess('Task completed successfully.');
      onOpenChange(false);
      onSuccess();
    } catch {
      showError('Failed to complete task. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  // Show a summary of filled form fields
  const filledEntries = Object.entries(formData).filter(
    ([, v]) => v !== undefined && v !== null && v !== '',
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-100">
              <CheckCircle className="h-5 w-5 text-green-600" />
            </div>
            <div>
              <DialogTitle>Complete Task</DialogTitle>
              <DialogDescription className="mt-0.5">
                Review your answers before completing.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {filledEntries.length > 0 && (
          <div className="rounded-lg border bg-muted/30 p-3">
            <p className="mb-2 text-xs font-medium text-muted-foreground">Your answers:</p>
            <dl className="space-y-1.5">
              {filledEntries.slice(0, 5).map(([key, value]) => (
                <div key={key} className="flex gap-2 text-sm">
                  <dt className="shrink-0 font-medium capitalize">
                    {key.replace(/_/g, ' ')}:
                  </dt>
                  <dd className="truncate text-muted-foreground">
                    {typeof value === 'boolean'
                      ? value
                        ? 'Yes'
                        : 'No'
                      : String(value)}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        )}

        <p className="text-xs text-muted-foreground">
          This action will advance the workflow to the next step and cannot be undone.
        </p>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button onClick={handleComplete} disabled={isSubmitting}>
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Completing...
              </>
            ) : (
              'Complete Task'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
