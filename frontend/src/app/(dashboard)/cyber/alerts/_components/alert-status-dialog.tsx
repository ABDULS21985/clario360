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
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { CyberAlert, AlertStatus } from '@/types/cyber';

const STATUSES: { value: AlertStatus; label: string }[] = [
  { value: 'acknowledged', label: 'Acknowledged' },
  { value: 'investigating', label: 'Investigating' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'closed', label: 'Closed' },
  { value: 'false_positive', label: 'False Positive' },
];

const schema = z.object({
  status: z.enum(['acknowledged', 'investigating', 'in_progress', 'resolved', 'closed', 'false_positive']),
  resolution_notes: z.string().optional(),
  false_positive_reason: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface AlertStatusDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alert: CyberAlert;
  onSuccess?: () => void;
}

export function AlertStatusDialog({ open, onOpenChange, alert, onSuccess }: AlertStatusDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { status: 'acknowledged', resolution_notes: '', false_positive_reason: '' },
  });

  const watchStatus = methods.watch('status');

  const { mutate, isPending } = useApiMutation<CyberAlert, FormValues>(
    'put',
    `${API_ENDPOINTS.CYBER_ALERTS}/${alert.id}/status`,
    {
      successMessage: 'Alert status updated',
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
          <DialogTitle>Update Alert Status</DialogTitle>
          <DialogDescription>Change the status of: <strong>{alert.title}</strong></DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit((d) => mutate(d))} className="space-y-4">
            <FormField name="status" label="New Status" required>
              <Select
                value={watchStatus}
                onValueChange={(v) => methods.setValue('status', v as FormValues['status'])}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {STATUSES.map((s) => (
                    <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            {(watchStatus === 'resolved' || watchStatus === 'closed') && (
              <FormField name="resolution_notes" label="Resolution Notes">
                <Textarea
                  placeholder="Describe how the alert was resolved…"
                  rows={3}
                  {...methods.register('resolution_notes')}
                />
              </FormField>
            )}

            {watchStatus === 'false_positive' && (
              <FormField name="false_positive_reason" label="Reason" required>
                <Textarea
                  placeholder="Why is this a false positive?"
                  rows={3}
                  {...methods.register('false_positive_reason')}
                />
              </FormField>
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Updating…' : 'Update Status'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
