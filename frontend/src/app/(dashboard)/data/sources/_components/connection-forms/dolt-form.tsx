'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import type { DoltConnectionValues } from '@/lib/data-suite/forms';

interface DoltFormProps {
  form: UseFormReturn<DoltConnectionValues>;
}

export function DoltForm({ form }: DoltFormProps) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField name="host" label="Host" required>
          <Input {...form.register('host')} placeholder="dolt-server.example.com" />
        </FormField>
        <FormField name="port" label="Port" required>
          <Input type="number" {...form.register('port', { valueAsNumber: true })} />
        </FormField>
        <FormField name="database" label="Database" required>
          <Input {...form.register('database')} />
        </FormField>
        <FormField name="branch" label="Branch">
          <Input {...form.register('branch')} placeholder="main" />
        </FormField>
        <FormField name="username" label="Username" required>
          <Input {...form.register('username')} />
        </FormField>
        <FormField name="password" label="Password" required>
          <Input type="password" autoComplete="new-password" {...form.register('password')} />
        </FormField>
      </div>

      <div className="flex items-center justify-between rounded-lg border p-4">
        <div>
          <Label>TLS / SSL</Label>
          <p className="text-xs text-muted-foreground">Enable TLS for encrypted MySQL wire-protocol sessions.</p>
        </div>
        <Switch checked={form.watch('use_tls')} onCheckedChange={(checked) => form.setValue('use_tls', checked, { shouldValidate: true })} />
      </div>
    </div>
  );
}
