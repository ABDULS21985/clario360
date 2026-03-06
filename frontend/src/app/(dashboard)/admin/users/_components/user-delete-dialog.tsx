'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { apiDelete } from '@/lib/api';
import { isApiError } from '@/types/api';
import type { User } from '@/types/models';

interface UserDeleteDialogProps {
  user: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function UserDeleteDialog({
  user,
  open,
  onOpenChange,
  onSuccess,
}: UserDeleteDialogProps) {
  const [loading, setLoading] = useState(false);
  const name = `${user.first_name} ${user.last_name}`.trim();

  const handleConfirm = async () => {
    setLoading(true);
    try {
      await apiDelete(`/api/v1/users/${user.id}`);
      toast.success(`User ${name} has been deleted.`);
      onOpenChange(false);
      onSuccess();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to delete user.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete User"
      description={`This action will permanently deactivate ${name}'s account. All their sessions will be revoked and they will lose access to the platform. This action cannot be undone.`}
      confirmLabel="Delete User"
      variant="destructive"
      typeToConfirm={user.email}
      onConfirm={handleConfirm}
      loading={loading}
    />
  );
}
