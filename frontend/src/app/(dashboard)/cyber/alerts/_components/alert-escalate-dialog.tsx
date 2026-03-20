'use client';

import { useEffect, useMemo } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { ArrowUpCircle } from 'lucide-react';
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
import { Textarea } from '@/components/ui/textarea';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type { PaginatedResponse } from '@/types/api';
import type { CyberAlert } from '@/types/cyber';
import type { User } from '@/types/models';

const schema = z.object({
  escalated_to: z.string().uuid('Select an escalation target'),
  reason: z.string().min(5, 'Provide a clear escalation reason'),
});

type FormValues = z.infer<typeof schema>;

interface AlertEscalateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alert: CyberAlert;
  onSuccess?: () => void;
}

export function AlertEscalateDialog({ open, onOpenChange, alert, onSuccess }: AlertEscalateDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { escalated_to: '', reason: '' },
  });

  const usersQuery = useQuery({
    queryKey: ['alert-escalate-users'],
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
    (usersQuery.data?.data ?? [])
      .filter((user) => user.id !== alert.assigned_to)
      .map((user) => ({
        value: user.id,
        label: `${user.first_name} ${user.last_name} (${user.email})`,
      }))
  ), [alert.assigned_to, usersQuery.data?.data]);

  useEffect(() => {
    if (!open) {
      methods.reset({ escalated_to: '', reason: '' });
    }
  }, [methods, open]);

  async function handleSubmit(values: FormValues) {
    await apiPost(API_ENDPOINTS.CYBER_ALERT_ESCALATE(alert.id), values);
    toast.success('Alert escalated');
    methods.reset({ escalated_to: '', reason: '' });
    onOpenChange(false);
    onSuccess?.();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-red-700">
            <ArrowUpCircle className="h-5 w-5" />
            Escalate Alert
          </DialogTitle>
          <DialogDescription>
            Escalate {alert.title} to a higher-tier analyst with a documented reason.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField name="escalated_to" label="Escalate To" required>
              <Combobox
                options={options}
                value={methods.watch('escalated_to')}
                onChange={(value) => methods.setValue('escalated_to', value, { shouldValidate: true })}
                placeholder={usersQuery.isLoading ? 'Loading analysts…' : 'Select escalation target'}
                searchPlaceholder="Search analysts"
                disabled={usersQuery.isLoading || options.length === 0}
                className="w-full justify-between"
              />
            </FormField>

            <FormField name="reason" label="Reason" required>
              <Textarea
                rows={4}
                placeholder="Explain why this alert needs to be escalated."
                {...methods.register('reason')}
              />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={methods.formState.isSubmitting || usersQuery.isLoading}>
                {methods.formState.isSubmitting ? 'Escalating…' : 'Escalate'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
