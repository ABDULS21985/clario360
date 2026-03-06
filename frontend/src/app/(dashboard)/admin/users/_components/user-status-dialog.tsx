'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { apiPut } from '@/lib/api';
import { isApiError } from '@/types/api';
import type { User } from '@/types/models';

interface UserStatusDialogProps {
  user: User;
  targetStatus: 'active' | 'suspended';
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function UserStatusDialog({
  user,
  targetStatus,
  open,
  onOpenChange,
  onSuccess,
}: UserStatusDialogProps) {
  const [loading, setLoading] = useState(false);
  const name = `${user.first_name} ${user.last_name}`.trim();
  const isSuspending = targetStatus === 'suspended';

  const handleConfirm = async () => {
    setLoading(true);
    try {
      await apiPut(`/api/v1/users/${user.id}/status`, { status: targetStatus });
      toast.success(
        isSuspending
          ? `${name} has been suspended. All active sessions have been revoked.`
          : `${name} has been activated.`
      );
      onOpenChange(false);
      onSuccess();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to update user status.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={isSuspending ? 'Suspend User' : 'Activate User'}
      description={
        isSuspending
          ? `Are you sure you want to suspend ${name}? They will be immediately logged out and unable to access the platform until reactivated.`
          : `This will restore access for ${name}. They will be able to log in again.`
      }
      confirmLabel={isSuspending ? 'Suspend User' : 'Activate User'}
      variant={isSuspending ? 'destructive' : 'default'}
      onConfirm={handleConfirm}
      loading={loading}
    />
  );
}
