'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Loader2 } from 'lucide-react';
import { apiPost } from '@/lib/api';
import { showSuccess, showError } from '@/lib/toast';
import type { HumanTask } from '@/types/models';

interface TaskClaimButtonProps {
  task: HumanTask;
  onSuccess: () => void;
}

export function TaskClaimButton({ task, onSuccess }: TaskClaimButtonProps) {
  const [isClaiming, setIsClaiming] = useState(false);

  const handleClaim = async () => {
    setIsClaiming(true);
    try {
      await apiPost(`/api/v1/workflows/tasks/${task.id}/claim`);
      showSuccess('Task claimed.');
      onSuccess();
    } catch (err: unknown) {
      const status = (err as { status?: number })?.status;
      if (status === 409) {
        showError('This task was already claimed by someone else.');
        onSuccess(); // refetch to get updated state
      } else if (status === 403) {
        showError("You don't have the required role to claim this task.");
      } else {
        showError('Failed to claim task. Please try again.');
      }
    } finally {
      setIsClaiming(false);
    }
  };

  return (
    <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed p-8 text-center">
      <p className="text-sm font-medium">This task is not yet claimed.</p>
      {task.assignee_role && (
        <p className="text-xs text-muted-foreground">
          Available for anyone with the{' '}
          <span className="font-medium">{task.assignee_role}</span> role.
        </p>
      )}
      <Button onClick={handleClaim} disabled={isClaiming} size="lg">
        {isClaiming ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Claiming...
          </>
        ) : (
          'Claim This Task'
        )}
      </Button>
    </div>
  );
}
