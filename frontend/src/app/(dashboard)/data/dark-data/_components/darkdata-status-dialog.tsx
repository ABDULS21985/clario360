'use client';

import { useEffect } from 'react';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/shared/forms/form-field';
import { Textarea } from '@/components/ui/textarea';
import type { DarkDataAsset, DarkDataGovernanceStatus } from '@/lib/data-suite';

const schema = z.object({
  governance_notes: z.string().min(3, 'Notes are required'),
});

type DarkDataStatusValues = z.infer<typeof schema>;

interface DarkDataStatusDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  asset: DarkDataAsset | null;
  targetStatus: DarkDataGovernanceStatus | null;
  submitting: boolean;
  onSubmit: (values: DarkDataStatusValues) => void;
}

export function DarkDataStatusDialog({
  open,
  onOpenChange,
  asset,
  targetStatus,
  submitting,
  onSubmit,
}: DarkDataStatusDialogProps) {
  const form = useForm<DarkDataStatusValues>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: {
      governance_notes: '',
    },
  });

  useEffect(() => {
    if (open) {
      form.reset({ governance_notes: '' });
    }
  }, [form, open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="capitalize">
            {targetStatus === 'archived' ? 'Archive asset' : 'Schedule deletion'}
          </DialogTitle>
        </DialogHeader>

        <FormProvider {...form}>
          <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
            <div className="text-sm text-muted-foreground">
              {asset ? `Update governance for ${asset.name}.` : 'Select an asset.'}
            </div>

            <FormField name="governance_notes" label="Notes" required>
              <Textarea
                {...form.register('governance_notes')}
                rows={4}
                placeholder="Explain why this asset should be archived or scheduled for deletion."
              />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={!form.formState.isValid || submitting}>
                {submitting
                  ? 'Saving…'
                  : targetStatus === 'archived'
                    ? 'Archive asset'
                    : 'Schedule deletion'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

