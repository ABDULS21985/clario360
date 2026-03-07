'use client';

import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/shared/forms/form-field';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { contradictionResolutionSchema, type ContradictionResolutionValues } from '@/lib/data-suite/forms';
import { type Contradiction } from '@/lib/data-suite';

interface ContradictionResolveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  contradiction: Contradiction | null;
  submitting: boolean;
  onSubmit: (values: ContradictionResolutionValues) => void;
}

export function ContradictionResolveDialog({
  open,
  onOpenChange,
  contradiction,
  submitting,
  onSubmit,
}: ContradictionResolveDialogProps) {
  const form = useForm<ContradictionResolutionValues>({
    resolver: zodResolver(contradictionResolutionSchema),
    defaultValues: {
      resolution_action: 'data_reconciled',
      resolution_notes: '',
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Resolve Contradiction</DialogTitle>
        </DialogHeader>
        <FormProvider {...form}>
          <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
            <FormField name="resolution_action" label="Resolution action" required>
              <Select value={form.watch('resolution_action')} onValueChange={(value) => form.setValue('resolution_action', value as ContradictionResolutionValues['resolution_action'], { shouldValidate: true })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="source_a_corrected">Source A corrected</SelectItem>
                  <SelectItem value="source_b_corrected">Source B corrected</SelectItem>
                  <SelectItem value="both_corrected">Both corrected</SelectItem>
                  <SelectItem value="data_reconciled">Data reconciled</SelectItem>
                  <SelectItem value="accepted_as_is">Accepted as is</SelectItem>
                  <SelectItem value="false_positive">False positive</SelectItem>
                </SelectContent>
              </Select>
            </FormField>

            <FormField name="resolution_notes" label="Resolution notes" required>
              <Textarea rows={5} {...form.register('resolution_notes')} placeholder={contradiction?.title} />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? 'Submitting…' : 'Resolve'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
