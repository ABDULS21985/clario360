'use client';

import { useState } from 'react';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { apiPost } from '@/lib/api';
import { showSuccess, showError } from '@/lib/toast';

interface WorkflowCancelDialogProps {
  instanceId: string;
  definitionName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function WorkflowCancelDialog({
  instanceId,
  definitionName,
  open,
  onOpenChange,
  onSuccess,
}: WorkflowCancelDialogProps) {
  const [isLoading, setIsLoading] = useState(false);

  const handleConfirm = async () => {
    setIsLoading(true);
    try {
      await apiPost(`/api/v1/workflows/instances/${instanceId}/cancel`);
      showSuccess('Workflow cancelled.');
      onSuccess();
    } catch {
      showError('Failed to cancel workflow. Please try again.');
      throw new Error('cancel failed'); // prevent dialog close
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Cancel Workflow"
      description={`This will cancel the workflow "${definitionName}" and all pending tasks. Active tasks will be marked as cancelled. This cannot be undone.`}
      confirmLabel="Cancel Workflow"
      variant="destructive"
      typeToConfirm="CANCEL"
      onConfirm={handleConfirm}
      loading={isLoading}
    />
  );
}
