'use client';

import { useEffect, useMemo } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { FormField } from '@/components/shared/forms/form-field';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { ALERT_STATUS_TRANSITIONS, getAlertStatusLabel } from '@/lib/cyber-alerts';
import { toast } from 'sonner';
import type { AlertStatus, CyberAlert } from '@/types/cyber';

const schema = z.object({
  status: z.enum([
    'acknowledged',
    'investigating',
    'in_progress',
    'resolved',
    'closed',
    'false_positive',
    'escalated',
  ]),
  notes: z.string().optional(),
  reason: z.string().optional(),
}).superRefine((values, ctx) => {
  if (values.status === 'resolved' && !values.notes?.trim()) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Resolution notes are required',
      path: ['notes'],
    });
  }
  if (values.status === 'false_positive' && !values.reason?.trim()) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'A false-positive reason is required',
      path: ['reason'],
    });
  }
});

type FormValues = z.infer<typeof schema>;

interface AlertStatusDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alert: CyberAlert;
  initialStatus?: AlertStatus;
  onSuccess?: () => void;
}

export function AlertStatusDialog({ open, onOpenChange, alert, initialStatus, onSuccess }: AlertStatusDialogProps) {
  const availableStatuses = useMemo(
    () => ALERT_STATUS_TRANSITIONS[alert.status] ?? [],
    [alert.status],
  );
  const preferredStatus = useMemo<FormValues['status']>(() => {
    if (initialStatus && availableStatuses.includes(initialStatus)) {
      return initialStatus as FormValues['status'];
    }
    return (availableStatuses[0] ?? 'acknowledged') as FormValues['status'];
  }, [availableStatuses, initialStatus]);

  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      status: preferredStatus,
      notes: '',
      reason: '',
    },
  });

  useEffect(() => {
    methods.reset({
      status: preferredStatus,
      notes: '',
      reason: '',
    });
  }, [methods, open, preferredStatus]);

  const nextStatus = methods.watch('status');

  async function handleSubmit(values: FormValues) {
    if (values.status === 'false_positive') {
      await apiPut(API_ENDPOINTS.CYBER_ALERT_FALSE_POSITIVE(alert.id), {
        reason: values.reason?.trim(),
      });
    } else {
      await apiPut(API_ENDPOINTS.CYBER_ALERT_STATUS(alert.id), {
        status: values.status,
        notes: values.notes?.trim() || undefined,
        reason: values.reason?.trim() || undefined,
      });
    }

    toast.success(`Alert moved to ${getAlertStatusLabel(values.status)}`);
    methods.reset({
      status: preferredStatus,
      notes: '',
      reason: '',
    });
    onOpenChange(false);
    onSuccess?.();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Update Alert Status</DialogTitle>
          <DialogDescription>
            Move {alert.title} to the next lifecycle stage. Only valid backend transitions are shown.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField name="status" label="New Status" required>
              <Select
                value={nextStatus}
                onValueChange={(value) => methods.setValue('status', value as FormValues['status'], { shouldValidate: true })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select status" />
                </SelectTrigger>
                <SelectContent>
                  {availableStatuses.map((status) => (
                    <SelectItem key={status} value={status}>
                      {getAlertStatusLabel(status)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            {(nextStatus === 'resolved' || nextStatus === 'closed' || nextStatus === 'investigating') && (
              <FormField name="notes" label={nextStatus === 'resolved' ? 'Resolution Summary' : 'Analyst Notes'}>
                <Textarea
                  rows={4}
                  placeholder={
                    nextStatus === 'resolved'
                      ? 'Document how the alert was resolved.'
                      : 'Add context for this transition.'
                  }
                  {...methods.register('notes')}
                />
              </FormField>
            )}

            {nextStatus === 'false_positive' && (
              <FormField name="reason" label="False Positive Reason" required>
                <Textarea
                  rows={4}
                  placeholder="Describe why this alert is benign."
                  {...methods.register('reason')}
                />
              </FormField>
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={methods.formState.isSubmitting || availableStatuses.length === 0}>
                {methods.formState.isSubmitting ? 'Updating…' : 'Update Status'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
