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
import { Textarea } from '@/components/ui/textarea';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { ArrowUpCircle } from 'lucide-react';
import type { CyberAlert } from '@/types/cyber';

const schema = z.object({
  escalate_to: z.string().min(1, 'Target is required'),
  reason: z.string().min(5, 'Reason must be at least 5 characters'),
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
    defaultValues: { escalate_to: '', reason: '' },
  });

  const { mutate, isPending } = useApiMutation<CyberAlert, FormValues>(
    'post',
    `${API_ENDPOINTS.CYBER_ALERTS}/${alert.id}/escalate`,
    {
      successMessage: 'Alert escalated',
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
          <DialogTitle className="flex items-center gap-2 text-amber-600">
            <ArrowUpCircle className="h-5 w-5" />
            Escalate Alert
          </DialogTitle>
          <DialogDescription>
            Escalate <strong>{alert.title}</strong> to a higher-tier analyst or manager.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit((d) => mutate(d))} className="space-y-4">
            <FormField name="escalate_to" label="Escalate To" required>
              <Input placeholder="Analyst name, team, or email" {...methods.register('escalate_to')} />
            </FormField>
            <FormField name="reason" label="Escalation Reason" required>
              <Textarea
                placeholder="Explain why this alert requires escalation…"
                rows={3}
                {...methods.register('reason')}
              />
            </FormField>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" variant="default" disabled={isPending}
                className="bg-amber-600 text-white hover:bg-amber-700">
                {isPending ? 'Escalating…' : 'Escalate'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
