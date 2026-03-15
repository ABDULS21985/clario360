'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import type { S3ConnectionValues } from '@/lib/data-suite/forms';

interface S3FormProps {
  form: UseFormReturn<S3ConnectionValues>;
}

export function S3Form({ form }: S3FormProps) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField name="endpoint" label="Endpoint" required>
          <Input {...form.register('endpoint')} placeholder="https://s3.amazonaws.com" />
        </FormField>
        <FormField name="bucket" label="Bucket" required>
          <Input {...form.register('bucket')} />
        </FormField>
        <FormField name="prefix" label="Prefix">
          <Input {...form.register('prefix')} />
        </FormField>
        <FormField name="region" label="Region">
          <Input {...form.register('region')} />
        </FormField>
        <FormField name="access_key" label="Access key" required>
          <Input {...form.register('access_key')} />
        </FormField>
        <FormField name="secret_key" label="Secret key" required>
          <Input type="password" autoComplete="new-password" {...form.register('secret_key')} />
        </FormField>
      </div>

      <div className="flex items-center gap-3 rounded-lg border p-4">
        <Switch checked={form.watch('use_ssl')} onCheckedChange={(checked) => form.setValue('use_ssl', checked, { shouldValidate: true })} />
        <Label>Use SSL</Label>
      </div>
    </div>
  );
}
