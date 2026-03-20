'use client';

import { useEffect, useMemo } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { UserCheck } from 'lucide-react';
import { z } from 'zod';
import { FormField } from '@/components/shared/forms/form-field';
import { Combobox } from '@/components/shared/forms/combobox';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { apiGet, apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type { PaginatedResponse } from '@/types/api';
import type { CyberAlert } from '@/types/cyber';
import type { User } from '@/types/models';

const schema = z.object({
  assigned_to: z.string().uuid('Select an analyst'),
});

type FormValues = z.infer<typeof schema>;

interface AlertAssignDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alert?: CyberAlert | null;
  alertIds?: string[];
  onSuccess?: () => void;
}

export function AlertAssignDialog({
  open,
  onOpenChange,
  alert,
  alertIds,
  onSuccess,
}: AlertAssignDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { assigned_to: '' },
  });

  const targetIds = useMemo(
    () => (alertIds && alertIds.length > 0 ? alertIds : alert ? [alert.id] : []),
    [alert, alertIds],
  );

  const usersQuery = useQuery({
    queryKey: ['alert-assign-users'],
    queryFn: () => apiGet<PaginatedResponse<User>>('/api/v1/users', {
      page: 1,
      per_page: 100,
      status: 'active',
      sort: 'created_at',
      order: 'desc',
    }),
    enabled: open,
  });

  const options = useMemo(() => (
    (usersQuery.data?.data ?? []).map((user) => ({
      value: user.id,
      label: `${user.first_name} ${user.last_name} (${user.email})`,
    }))
  ), [usersQuery.data?.data]);

  useEffect(() => {
    if (!open) {
      methods.reset({ assigned_to: '' });
    }
  }, [methods, open]);

  async function handleSubmit(values: FormValues) {
    if (targetIds.length === 0) {
      toast.error('No alerts selected');
      return;
    }

    await Promise.all(targetIds.map((id) => (
      apiPut(API_ENDPOINTS.CYBER_ALERT_ASSIGN(id), {
        assigned_to: values.assigned_to,
      })
    )));

    toast.success(targetIds.length === 1 ? 'Alert assigned successfully' : `${targetIds.length} alerts assigned`);
    methods.reset({ assigned_to: '' });
    onOpenChange(false);
    onSuccess?.();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserCheck className="h-5 w-5 text-primary" />
            {targetIds.length > 1 ? `Assign ${targetIds.length} alerts` : 'Assign Alert'}
          </DialogTitle>
          <DialogDescription>
            {targetIds.length > 1
              ? 'Route the selected alerts to an analyst for investigation.'
              : `Route ${alert?.title ?? 'this alert'} to an analyst for investigation.`}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField
              name="assigned_to"
              label="Analyst"
              required
              description="Acknowledging an alert will auto-assign it to the acting analyst if it is still unowned."
            >
              <Combobox
                options={options}
                value={methods.watch('assigned_to')}
                onChange={(value) => methods.setValue('assigned_to', value, { shouldValidate: true })}
                placeholder={usersQuery.isLoading ? 'Loading analysts…' : 'Select an analyst'}
                searchPlaceholder="Search analysts"
                disabled={usersQuery.isLoading || options.length === 0}
                className="w-full justify-between"
              />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={methods.formState.isSubmitting || usersQuery.isLoading}>
                {methods.formState.isSubmitting ? 'Assigning…' : 'Assign'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
