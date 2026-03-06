'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
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
import { Combobox } from '@/components/shared/forms/combobox';
import { useQuery } from '@tanstack/react-query';
import { apiGet, apiPost } from '@/lib/api';
import { useAuth } from '@/hooks/use-auth';
import { showSuccess, showError } from '@/lib/toast';
import type { HumanTask, User } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

interface TaskDelegateDialogProps {
  task: HumanTask;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function TaskDelegateDialog({
  task,
  open,
  onOpenChange,
  onSuccess,
}: TaskDelegateDialogProps) {
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [reason, setReason] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { user } = useAuth();

  const { data: usersData } = useQuery({
    queryKey: ['role-users', task.assignee_role],
    queryFn: () =>
      apiGet<PaginatedResponse<User>>(
        `/api/v1/roles/${task.assignee_role}/users`,
        { per_page: 100 },
      ),
    enabled: open && !!task.assignee_role,
  });

  const userOptions = (usersData?.data ?? [])
    .filter((u) => u.id !== user?.id)
    .map((u) => ({
      label: `${u.first_name} ${u.last_name} (${u.email})`,
      value: u.id,
    }));

  const handleDelegate = async () => {
    if (!selectedUserId) return;
    setIsSubmitting(true);
    try {
      const selectedUser = usersData?.data.find((u) => u.id === selectedUserId);
      await apiPost(`/api/v1/workflows/tasks/${task.id}/delegate`, {
        user_id: selectedUserId,
        reason: reason.trim() || undefined,
      });
      const name = selectedUser ? `${selectedUser.first_name} ${selectedUser.last_name}` : 'user';
      showSuccess(`Task delegated to ${name}.`);
      onOpenChange(false);
      setSelectedUserId('');
      setReason('');
      onSuccess();
    } catch {
      showError('Failed to delegate task. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setSelectedUserId('');
      setReason('');
    }
    onOpenChange(open);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Delegate Task</DialogTitle>
          <DialogDescription>
            Transfer this task to another user with the required role.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label>
              Delegate To <span className="text-destructive">*</span>
            </Label>
            <Combobox
              options={userOptions}
              value={selectedUserId}
              onChange={setSelectedUserId}
              placeholder="Search users..."
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="delegate-reason">Reason (optional)</Label>
            <Textarea
              id="delegate-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Why are you delegating this task?"
              className="min-h-[80px]"
              disabled={isSubmitting}
            />
          </div>
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
            onClick={handleDelegate}
            disabled={!selectedUserId || isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Delegating...
              </>
            ) : (
              'Delegate Task'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
