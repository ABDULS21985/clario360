'use client';

import { useEffect } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { GitMerge } from 'lucide-react';
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
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type { CyberAlert } from '@/types/cyber';

const schema = z.object({
  primary_id: z.string().uuid('Select the primary alert'),
});

type FormValues = z.infer<typeof schema>;

interface AlertMergeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alerts: CyberAlert[];
  onSuccess?: () => void;
}

export function AlertMergeDialog({ open, onOpenChange, alerts, onSuccess }: AlertMergeDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { primary_id: alerts[0]?.id ?? '' },
  });

  useEffect(() => {
    if (open) {
      methods.reset({ primary_id: alerts[0]?.id ?? '' });
    }
  }, [alerts, methods, open]);

  async function handleSubmit(values: FormValues) {
    const mergeIds = alerts
      .map((alert) => alert.id)
      .filter((id) => id !== values.primary_id);

    if (mergeIds.length === 0) {
      toast.error('Select at least two alerts to merge');
      return;
    }

    await apiPost(API_ENDPOINTS.CYBER_ALERT_MERGE(values.primary_id), {
      merge_ids: mergeIds,
    });

    toast.success(`Merged ${mergeIds.length} related alerts`);
    onOpenChange(false);
    onSuccess?.();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <GitMerge className="h-5 w-5 text-primary" />
            Merge Alerts
          </DialogTitle>
          <DialogDescription>
            Choose which alert remains open. The others will be merged into it and marked accordingly.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField name="primary_id" label="Primary Alert" required>
              <Select
                value={methods.watch('primary_id')}
                onValueChange={(value) => methods.setValue('primary_id', value, { shouldValidate: true })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Choose primary alert" />
                </SelectTrigger>
                <SelectContent>
                  {alerts.map((alert) => (
                    <SelectItem key={alert.id} value={alert.id}>
                      {alert.title}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <div className="rounded-2xl border bg-muted/30 p-3">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                Merge Set
              </p>
              <div className="mt-3 space-y-2">
                {alerts.map((alert) => (
                  <div key={alert.id} className="rounded-xl border bg-background px-3 py-2 text-sm">
                    {alert.title}
                  </div>
                ))}
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={methods.formState.isSubmitting}>
                {methods.formState.isSubmitting ? 'Merging…' : 'Merge Alerts'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
