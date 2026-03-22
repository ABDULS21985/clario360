'use client';

import { useEffect } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { ShieldAlert } from 'lucide-react';
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
import { Textarea } from '@/components/ui/textarea';
import { apiPost, apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type { CyberAlert } from '@/types/cyber';

const schema = z.object({
  reason: z.string().min(5, 'Provide a clear reason'),
});

type FormValues = z.infer<typeof schema>;

interface AlertFalsePositiveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alert?: CyberAlert | null;
  alertIds?: string[];
  onSuccess?: () => void;
}

export function AlertFalsePositiveDialog({
  open,
  onOpenChange,
  alert,
  alertIds,
  onSuccess,
}: AlertFalsePositiveDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { reason: '' },
  });

  const targetIds = alertIds && alertIds.length > 0 ? alertIds : alert ? [alert.id] : [];

  useEffect(() => {
    if (!open) {
      methods.reset({ reason: '' });
    }
  }, [methods, open]);

  async function handleSubmit(values: FormValues) {
    if (targetIds.length === 0) {
      toast.error('No alerts selected');
      return;
    }

    if (targetIds.length > 1) {
      await apiPut(API_ENDPOINTS.CYBER_ALERT_BULK_FALSE_POSITIVE, {
        alert_ids: targetIds,
        reason: values.reason.trim(),
      });
    } else {
      await apiPut(API_ENDPOINTS.CYBER_ALERT_FALSE_POSITIVE(targetIds[0]), {
        reason: values.reason.trim(),
      });
    }

    // Submit rule feedback for single-alert false positives when tied to a detection rule
    if (alert?.rule_id && targetIds.length === 1) {
      try {
        await apiPost(API_ENDPOINTS.CYBER_RULE_FEEDBACK(alert.rule_id), {
          alert_id: alert.id,
          feedback: 'false_positive',
        });
      } catch {
        // Best-effort — the alert was already marked FP above
      }
    }

    toast.success(targetIds.length === 1 ? 'Alert marked as false positive' : `${targetIds.length} alerts marked as false positive`);
    methods.reset({ reason: '' });
    onOpenChange(false);
    onSuccess?.();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-purple-700">
            <ShieldAlert className="h-5 w-5" />
            Mark False Positive
          </DialogTitle>
          <DialogDescription>
            {targetIds.length > 1
              ? 'Document why the selected alerts are benign so rule feedback stays accurate.'
              : `Document why ${alert?.title ?? 'this alert'} is benign.`}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField name="reason" label="Reason" required>
              <Textarea
                rows={4}
                placeholder="Explain why this activity should not be treated as malicious."
                {...methods.register('reason')}
              />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={methods.formState.isSubmitting}>
                {methods.formState.isSubmitting ? 'Updating…' : 'Confirm'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
