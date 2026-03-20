'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { HDFSConnectionValues, KerberosConfigValues } from '@/lib/data-suite/forms';
import { StringListField } from '@/app/(dashboard)/data/sources/_components/connection-forms/string-list-field';

interface HDFSFormProps {
  form: UseFormReturn<HDFSConnectionValues>;
}

export function HDFSForm({ form }: HDFSFormProps) {
  const kerberosEnabled = Boolean(form.watch('kerberos'));

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
      <StringListField
        name="name_nodes"
        label="NameNodes"
        required
        values={form.watch('name_nodes')}
        onChange={(values) => form.setValue('name_nodes', values, { shouldValidate: true })}
        placeholder="namenode:8020"
        description="Enter one or more HDFS NameNode addresses."
      />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField name="user" label="User">
          <Input {...form.register('user')} placeholder="hdfs" />
        </FormField>
        <FormField name="max_file_size_mb" label="Max File Size (MB)" required>
          <Input type="number" {...form.register('max_file_size_mb', { valueAsNumber: true })} />
        </FormField>
      </div>

      <StringListField
        name="base_paths"
        label="Base Paths"
        values={form.watch('base_paths')}
        onChange={(values) => form.setValue('base_paths', values, { shouldValidate: true })}
        placeholder="/user/hive/warehouse"
        description="Directories to scan for warehouse data and DSPM inspection."
      />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField name="audit_log_path" label="Audit Log Path">
          <Input {...form.register('audit_log_path')} placeholder="/var/log/hdfs/audit.log" />
        </FormField>
        <FormField name="kerberos" label="Authentication">
          <Select
            value={kerberosEnabled ? 'kerberos' : 'simple'}
            onValueChange={(value) =>
              form.setValue('kerberos', value === 'kerberos' ? { realm: '', kdc: '', principal: '', keytab: '' } : undefined, {
                shouldValidate: true,
              })
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="simple">Simple</SelectItem>
              <SelectItem value="kerberos">Kerberos</SelectItem>
            </SelectContent>
          </Select>
        </FormField>
      </div>

      {kerberosEnabled ? (
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
            <Input value={form.watch('kerberos')?.keytab ?? ''} onChange={(event) => updateKerberos({ keytab: event.target.value })} placeholder="/etc/security/keytabs/hdfs.keytab" />
          </FormField>
        </div>
      ) : null}
    </div>
  );
}
