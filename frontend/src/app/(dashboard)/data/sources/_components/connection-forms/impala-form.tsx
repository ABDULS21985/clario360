'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { ImpalaConnectionValues, KerberosConfigValues } from '@/lib/data-suite/forms';

interface ImpalaFormProps {
  form: UseFormReturn<ImpalaConnectionValues>;
}

export function ImpalaForm({ form }: ImpalaFormProps) {
  const authType = form.watch('auth_type');

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
          <Input {...form.register('host')} placeholder="impala-coordinator.example.com" />
        </FormField>
        <FormField name="port" label="Port" required>
          <Input type="number" {...form.register('port', { valueAsNumber: true })} />
        </FormField>
        <FormField name="database" label="Database">
          <Input {...form.register('database')} placeholder="default" />
        </FormField>
        <FormField name="auth_type" label="Authentication" required>
          <Select value={authType} onValueChange={(value) => form.setValue('auth_type', value as ImpalaConnectionValues['auth_type'], { shouldValidate: true })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="noauth">No Auth</SelectItem>
              <SelectItem value="ldap">LDAP</SelectItem>
              <SelectItem value="kerberos">Kerberos</SelectItem>
            </SelectContent>
          </Select>
        </FormField>
      </div>

      {authType === 'ldap' ? (
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
            <Input value={form.watch('kerberos')?.keytab ?? ''} onChange={(event) => updateKerberos({ keytab: event.target.value })} placeholder="/etc/security/keytabs/impala.keytab" />
          </FormField>
        </div>
      ) : null}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField name="audit_log_table" label="Audit Log Table">
          <Input {...form.register('audit_log_table')} placeholder="sys.impala_audit" />
        </FormField>
        <div className="flex items-center justify-between rounded-lg border p-4">
          <div>
            <Label>TLS / SSL</Label>
            <p className="text-xs text-muted-foreground">Enable TLS when Impala is fronted by secure transport.</p>
          </div>
          <Switch checked={form.watch('use_tls')} onCheckedChange={(checked) => form.setValue('use_tls', checked, { shouldValidate: true })} />
        </div>
      </div>
    </div>
  );
}
