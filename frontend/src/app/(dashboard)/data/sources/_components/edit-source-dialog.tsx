'use client';

import { useEffect } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { z } from 'zod';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { FormField } from '@/components/shared/forms/form-field';
import { type DataSource, dataSuiteApi } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';

const schema = z.object({
  name: z.string().min(2, 'Name is required').max(255),
  description: z.string().optional(),
  sync_frequency: z.string().nullable(),
  connection_config_json: z
    .string()
    .min(2, 'Connection config is required')
    .refine((value) => {
      try {
        JSON.parse(value);
        return true;
      } catch {
        return false;
      }
    }, 'Connection config must be valid JSON'),
});

type FormValues = z.infer<typeof schema>;

interface EditSourceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  source: DataSource | null;
  onUpdated?: () => void;
}

export function EditSourceDialog({
  open,
  onOpenChange,
  source,
  onUpdated,
}: EditSourceDialogProps) {
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      description: '',
      sync_frequency: null,
      connection_config_json: '{}',
    },
  });

  useEffect(() => {
    if (!source || !open) {
      return;
    }
    form.reset({
      name: source.name,
      description: source.description,
      sync_frequency: source.sync_frequency ?? null,
      connection_config_json: JSON.stringify(source.connection_config ?? {}, null, 2),
    });
  }, [form, open, source]);

  const onSubmit = async (values: FormValues) => {
    if (!source) {
      return;
    }

    try {
      await dataSuiteApi.updateSource(source.id, {
        name: values.name,
        description: values.description,
        sync_frequency: values.sync_frequency,
        connection_config: JSON.parse(values.connection_config_json),
      });
      showSuccess('Source updated successfully.');
      onUpdated?.();
      onOpenChange(false);
    } catch (error) {
      showApiError(error);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Edit source{source ? `: ${source.name}` : ''}</DialogTitle>
        </DialogHeader>
        <FormProvider {...form}>
          <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="name" label="Name" required>
                <Input {...form.register('name')} />
              </FormField>
              <FormField name="sync_frequency" label="Sync frequency">
                <Input {...form.register('sync_frequency')} placeholder="0 * * * * or leave blank for manual" />
              </FormField>
            </div>

            <FormField name="description" label="Description">
              <Textarea rows={3} {...form.register('description')} />
            </FormField>

            <FormField
              name="connection_config_json"
              label="Connection config JSON"
              description="Sanitized secrets are not returned by the backend. Re-enter secret fields if you update the connection."
              required
            >
              <Textarea rows={12} className="font-mono text-xs" {...form.register('connection_config_json')} />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit">Save changes</Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
