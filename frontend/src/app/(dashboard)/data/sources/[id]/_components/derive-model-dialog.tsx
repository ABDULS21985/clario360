'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { deriveModelName } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';

const schema = z.object({
  name: z.string().min(2, 'Model name is required'),
  auto_generate_quality_rules: z.boolean().default(true),
});

type FormValues = z.infer<typeof schema>;

interface DeriveModelDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sourceId: string;
  tableName: string | null;
}

export function DeriveModelDialog({
  open,
  onOpenChange,
  sourceId,
  tableName,
}: DeriveModelDialogProps) {
  const router = useRouter();
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: tableName ? deriveModelName(tableName) : '',
      auto_generate_quality_rules: true,
    },
  });

  useEffect(() => {
    if (!tableName || !open) {
      return;
    }
    form.reset({
      name: deriveModelName(tableName),
      auto_generate_quality_rules: true,
    });
  }, [form, open, tableName]);

  const mutation = useApiMutation('post', () => '/api/v1/data/models/derive', {
    invalidateKeys: ['data-models'],
    onSuccess: (data) => {
      const model = data as { data?: { id?: string } };
      const id = model.data?.id;
      if (id) {
        showSuccess('Model derived successfully.', 'The schema is now available as a governed model.');
        onOpenChange(false);
        router.push(`/data/models/${id}`);
      }
    },
    onError: (error) => showApiError(error),
  });

  const onSubmit = (values: FormValues) => {
    if (!tableName) {
      return;
    }
    mutation.mutate({
      source_id: sourceId,
      table_name: tableName,
      name: values.name,
      auto_generate_quality_rules: values.auto_generate_quality_rules,
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Derive model{tableName ? ` from ${tableName}` : ''}</DialogTitle>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          <FormField name="name" label="Model name" required>
            <Input {...form.register('name')} />
          </FormField>

          <div className="flex items-center gap-3 rounded-lg border p-3">
            <Checkbox
              id="auto_generate_quality_rules"
              checked={form.watch('auto_generate_quality_rules')}
              onCheckedChange={(checked) => form.setValue('auto_generate_quality_rules', Boolean(checked))}
            />
            <Label htmlFor="auto_generate_quality_rules">Auto-generate quality rules</Label>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending || !tableName}>
              {mutation.isPending ? 'Deriving…' : 'Derive model'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
