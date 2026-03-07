'use client';

import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
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
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { UserCheck } from 'lucide-react';
import type { CyberAlert } from '@/types/cyber';

const schema = z.object({
  user_id: z.string().uuid('Must be a valid user ID').or(z.literal('')),
  user_name: z.string().min(1, 'User name is required'),
  note: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface AlertAssignDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alert: CyberAlert;
  onSuccess?: () => void;
}

export function AlertAssignDialog({ open, onOpenChange, alert, onSuccess }: AlertAssignDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { user_id: '', user_name: '', note: '' },
  });

  const { mutate, isPending } = useApiMutation<CyberAlert, FormValues>(
    'put',
    `${API_ENDPOINTS.CYBER_ALERTS}/${alert.id}/assign`,
    {
      successMessage: 'Alert assigned successfully',
      invalidateKeys: ['cyber-alerts', `cyber-alert-${alert.id}`],
      onSuccess: () => {
        methods.reset();
        onOpenChange(false);
        onSuccess?.();
      },
    },
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserCheck className="h-5 w-5 text-primary" />
            Assign Alert
          </DialogTitle>
          <DialogDescription>
            Assign <strong>{alert.title}</strong> to an analyst for investigation.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit((d) => mutate(d))} className="space-y-4">
            <FormField name="user_name" label="Analyst Name" required>
              <Input placeholder="John Smith" {...methods.register('user_name')} />
            </FormField>
            <FormField name="user_id" label="User ID (optional)">
              <Input placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" {...methods.register('user_id')} />
            </FormField>
            <FormField name="note" label="Assignment Note">
              <Input placeholder="Reason for assignment…" {...methods.register('note')} />
            </FormField>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Assigning…' : 'Assign'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
