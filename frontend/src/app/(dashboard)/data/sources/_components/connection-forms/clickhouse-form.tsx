'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { ClickHouseConnectionValues } from '@/lib/data-suite/forms';

interface ClickHouseFormProps {
  form: UseFormReturn<ClickHouseConnectionValues>;
}

export function ClickHouseForm({ form }: ClickHouseFormProps) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField name="host" label="Host" required>
          <Input {...form.register('host')} placeholder="clickhouse.example.com" />
        </FormField>
        <FormField name="port" label="Port" required description="9000 for native TCP, 8123 for HTTP.">
          <Input type="number" {...form.register('port', { valueAsNumber: true })} />
        </FormField>
        <FormField name="database" label="Database" required>
          <Input {...form.register('database')} placeholder="default" />
        </FormField>
        <FormField name="protocol" label="Protocol" required>
          <Select value={form.watch('protocol')} onValueChange={(value) => form.setValue('protocol', value as ClickHouseConnectionValues['protocol'], { shouldValidate: true })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="native">Native</SelectItem>
              <SelectItem value="http">HTTP</SelectItem>
            </SelectContent>
          </Select>
        </FormField>
        <FormField name="username" label="Username" required>
          <Input {...form.register('username')} placeholder="default" />
        </FormField>
        <FormField name="password" label="Password" required>
          <Input type="password" autoComplete="new-password" {...form.register('password')} />
        </FormField>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="flex items-center justify-between rounded-lg border p-4">
          <div>
            <Label>TLS / SSL</Label>
            <p className="text-xs text-muted-foreground">Encrypt the connection to ClickHouse.</p>
          </div>
          <Switch checked={form.watch('secure')} onCheckedChange={(checked) => form.setValue('secure', checked, { shouldValidate: true })} />
        </div>
        <div className="flex items-center justify-between rounded-lg border p-4">
          <div>
            <Label>Compression</Label>
            <p className="text-xs text-muted-foreground">Enable LZ4 compression for faster transfer.</p>
          </div>
          <Switch checked={form.watch('compression')} onCheckedChange={(checked) => form.setValue('compression', checked, { shouldValidate: true })} />
        </div>
      </div>
    </div>
  );
}
