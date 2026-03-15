'use client';

import type { UseFormReturn } from 'react-hook-form';
import { Eye, EyeOff } from 'lucide-react';
import { useState } from 'react';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import type { MySQLConnectionValues } from '@/lib/data-suite/forms';

interface MySQLFormProps {
  form: UseFormReturn<MySQLConnectionValues>;
}

export function MySQLForm({ form }: MySQLFormProps) {
  const [showPassword, setShowPassword] = useState(false);

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <FormField name="host" label="Host" required>
        <Input {...form.register('host')} />
      </FormField>

      <FormField name="port" label="Port" required>
        <Input type="number" {...form.register('port', { valueAsNumber: true })} />
      </FormField>

      <FormField name="database" label="Database" required>
        <Input {...form.register('database')} />
      </FormField>

      <FormField name="username" label="Username" required>
        <Input {...form.register('username')} />
      </FormField>

      <FormField name="password" label="Password" required>
        <div className="flex gap-2">
          <Input
            type={showPassword ? 'text' : 'password'}
            autoComplete="new-password"
            {...form.register('password')}
          />
          <Button type="button" variant="outline" size="icon" onClick={() => setShowPassword((current) => !current)}>
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </Button>
        </div>
      </FormField>

      <FormField name="tls_mode" label="TLS" required>
        <Select value={form.watch('tls_mode')} onValueChange={(value) => form.setValue('tls_mode', value as MySQLConnectionValues['tls_mode'], { shouldValidate: true })}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="true">Enabled</SelectItem>
            <SelectItem value="preferred">Preferred</SelectItem>
            <SelectItem value="skip-verify">Skip verify</SelectItem>
            <SelectItem value="false">Disabled</SelectItem>
          </SelectContent>
        </Select>
      </FormField>
    </div>
  );
}
