'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { CalendarClock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { FormField } from '@/components/shared/forms/form-field';
import { enterpriseApi, extendDueDateSchema, type ExtendDueDateFormValues } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaActionItem } from '@/types/suites';

interface ExtendDueDateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item: ActaActionItem | null;
}

export function ExtendDueDateDialog({
  open,
  onOpenChange,
  item,
}: ExtendDueDateDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm<ExtendDueDateFormValues>({
    resolver: zodResolver(extendDueDateSchema),
    defaultValues: {
      new_due_date: item?.due_date ?? '',
      reason: '',
    },
  });

  const extendMutation = useMutation({
    mutationFn: (payload: ExtendDueDateFormValues) =>
      item ? enterpriseApi.acta.extendActionItem(item.id, payload) : Promise.reject(new Error('No action item selected.')),
    onSuccess: async () => {
      showSuccess('Due date extended.', 'The action item timeline has been updated.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-action-items'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
      onOpenChange(false);
      form.reset({ new_due_date: item?.due_date ?? '', reason: '' });
    },
    onError: showApiError,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Extend Due Date</DialogTitle>
          <DialogDescription>
            Record the extension rationale and preserve the original due date for auditability.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => extendMutation.mutate(values))}
          >
            <div className="rounded-xl border px-4 py-3 text-sm">
              <p className="font-medium">{item?.title}</p>
              <p className="mt-1 text-muted-foreground">
                Extension #{item?.extended_count ?? 0} • original due {item?.original_due_date ?? item?.due_date}
              </p>
            </div>

            <FormField name="new_due_date" label="New due date" required>
              <Input type="date" {...form.register('new_due_date')} />
            </FormField>
            <FormField name="reason" label="Reason" required>
              <Textarea {...form.register('reason')} rows={4} />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={extendMutation.isPending}>
                <CalendarClock className="mr-1.5 h-4 w-4" />
                {extendMutation.isPending ? 'Saving…' : 'Extend due date'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
