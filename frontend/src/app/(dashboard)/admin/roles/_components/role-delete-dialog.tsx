'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { apiDelete } from '@/lib/api';
import { isApiError } from '@/types/api';
import type { Role } from '@/types/models';

interface RoleDeleteDialogProps {
  role: Role;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function RoleDeleteDialog({
  role,
  open,
  onOpenChange,
  onSuccess,
}: RoleDeleteDialogProps) {
  const [loading, setLoading] = useState(false);

  const handleConfirm = async () => {
    setLoading(true);
    try {
      await apiDelete(`/api/v1/roles/${role.id}`);
      toast.success(`Role "${role.name}" has been deleted.`);
      onOpenChange(false);
      onSuccess();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to delete role.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete Role"
      description={`Are you sure you want to delete the "${role.name}" role? Users assigned this role will lose the associated permissions. This action cannot be undone.`}
      confirmLabel="Delete Role"
      variant="destructive"
      typeToConfirm={role.name}
      onConfirm={handleConfirm}
      loading={loading}
    />
  );
}
