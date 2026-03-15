'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { HiveConnectionValues, KerberosConfigValues } from '@/lib/data-suite/forms';

interface HiveFormProps {
  form: UseFormReturn<HiveConnectionValues>;
}

export function HiveForm({ form }: HiveFormProps) {
  const authType = form.watch('auth_type');
  const transportMode = form.watch('transport_mode');

  const updateKerberos = (patch: Partial<KerberosConfigValues>) => {
    form.setValue(
      'kerberos',
      {
        realm: form.getValues('kerberos')?.realm ?? '',
        kdc: form.getValues('kerberos')?.kdc ?? '',
        principal: form.getValues('kerberos')?.principal ?? '',
        keytab: form.getValues('kerberos')?.keytab ?? '',
        ...patch,
      },
      { shouldValidate: true },
    );
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField name="host" label="Host" required>
          <Input {...form.register('host')} placeholder="hiveserver2.example.com" />
        </FormField>
        <FormField name="port" label="Port" required>
          <Input type="number" {...form.register('port', { valueAsNumber: true })} />
        </FormField>
        <FormField name="database" label="Database">
          <Input {...form.register('database')} placeholder="default" />
        </FormField>
        <FormField name="transport_mode" label="Transport Mode" required>
          <Select value={transportMode} onValueChange={(value) => form.setValue('transport_mode', value as HiveConnectionValues['transport_mode'], { shouldValidate: true })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="binary">Binary</SelectItem>
              <SelectItem value="http">HTTP</SelectItem>
            </SelectContent>
          </Select>
        </FormField>
        {transportMode === 'http' ? (
          <FormField name="http_path" label="HTTP Path" required>
            <Input {...form.register('http_path')} placeholder="/cliservice" />
          </FormField>
        ) : null}
        <FormField name="auth_type" label="Authentication" required>
          <Select value={authType} onValueChange={(value) => form.setValue('auth_type', value as HiveConnectionValues['auth_type'], { shouldValidate: true })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="noauth">No Auth</SelectItem>
              <SelectItem value="plain">Username / Password</SelectItem>
              <SelectItem value="kerberos">Kerberos</SelectItem>
            </SelectContent>
          </Select>
        </FormField>
      </div>

      {authType === 'plain' ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField name="username" label="Username" required>
            <Input {...form.register('username')} />
          </FormField>
          <FormField name="password" label="Password" required>
            <Input type="password" autoComplete="new-password" {...form.register('password')} />
          </FormField>
        </div>
      ) : null}

      {authType === 'kerberos' ? (
        <div className="grid grid-cols-1 gap-4 rounded-lg border p-4 md:grid-cols-2">
          <FormField name="kerberos.realm" label="Realm" required>
            <Input value={form.watch('kerberos')?.realm ?? ''} onChange={(event) => updateKerberos({ realm: event.target.value })} />
          </FormField>
          <FormField name="kerberos.kdc" label="KDC" required>
            <Input value={form.watch('kerberos')?.kdc ?? ''} onChange={(event) => updateKerberos({ kdc: event.target.value })} />
          </FormField>
          <FormField name="kerberos.principal" label="Principal" required>
            <Input value={form.watch('kerberos')?.principal ?? ''} onChange={(event) => updateKerberos({ principal: event.target.value })} />
          </FormField>
          <FormField name="kerberos.keytab" label="Keytab Path">
            <Input value={form.watch('kerberos')?.keytab ?? ''} onChange={(event) => updateKerberos({ keytab: event.target.value })} placeholder="/etc/security/keytabs/hive.keytab" />
          </FormField>
        </div>
      ) : null}

      <div className="flex items-center justify-between rounded-lg border p-4">
        <div>
          <Label>TLS / SSL</Label>
          <p className="text-xs text-muted-foreground">Enable TLS when HiveServer2 is configured with secure transport.</p>
        </div>
        <Switch checked={form.watch('use_tls')} onCheckedChange={(checked) => form.setValue('use_tls', checked, { shouldValidate: true })} />
      </div>
    </div>
  );
}
