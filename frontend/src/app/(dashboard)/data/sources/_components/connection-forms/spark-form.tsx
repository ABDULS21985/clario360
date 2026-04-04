'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import type { KerberosConfigValues, SparkConnectionValues } from '@/lib/data-suite/forms';

interface SparkFormProps {
  form: UseFormReturn<SparkConnectionValues>;
}

export function SparkForm({ form }: SparkFormProps) {
  const thrift = form.watch('thrift') ?? { host: '', port: 10001, database: 'default', username: '', password: '', auth_type: 'noauth' as const };
  const thriftAuthType = thrift.auth_type;

  const updateThrift = (patch: Partial<NonNullable<SparkConnectionValues['thrift']>>) => {
    form.setValue(
      'thrift',
      {
        host: thrift.host ?? '',
        port: thrift.port ?? 10001,
        database: thrift.database ?? 'default',
        username: thrift.username ?? '',
        password: thrift.password ?? '',
        auth_type: thrift.auth_type ?? 'noauth',
        kerberos: thrift.kerberos,
        ...patch,
      },
      { shouldValidate: true },
    );
  };

  const updateKerberos = (patch: Partial<KerberosConfigValues>) => {
    updateThrift({
      kerberos: {
        realm: thrift.kerberos?.realm ?? '',
        kdc: thrift.kerberos?.kdc ?? '',
        principal: thrift.kerberos?.principal ?? '',
        keytab: thrift.kerberos?.keytab ?? '',
        ...patch,
      },
    });
  };

  return (
    <div className="space-y-4">
      <div className="space-y-3 rounded-lg border p-4">
        <div>
          <h4 className="font-medium">SQL Access</h4>
          <p className="text-sm text-muted-foreground">Configure Spark Thrift Server for schema discovery and data access.</p>
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField name="thrift.host" label="Thrift Host" required>
            <Input value={thrift.host ?? ''} onChange={(event) => updateThrift({ host: event.target.value })} placeholder="spark-thrift.example.com" />
          </FormField>
          <FormField name="thrift.port" label="Thrift Port" required>
            <Input type="number" value={thrift.port ?? 10001} onChange={(event) => updateThrift({ port: Number(event.target.value) || 10001 })} />
          </FormField>
          <FormField name="thrift.database" label="Database">
            <Input value={thrift.database ?? 'default'} onChange={(event) => updateThrift({ database: event.target.value })} placeholder="default" />
          </FormField>
          <FormField name="thrift.auth_type" label="Authentication" required>
            <Select value={thriftAuthType} onValueChange={(value) => updateThrift({ auth_type: value as NonNullable<SparkConnectionValues['thrift']>['auth_type'] })}>
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

        {thriftAuthType === 'plain' ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <FormField name="thrift.username" label="Username" required>
              <Input value={thrift.username ?? ''} onChange={(event) => updateThrift({ username: event.target.value })} />
            </FormField>
            <FormField name="thrift.password" label="Password" required>
              <Input type="password" autoComplete="new-password" value={thrift.password ?? ''} onChange={(event) => updateThrift({ password: event.target.value })} />
            </FormField>
          </div>
        ) : null}

        {thriftAuthType === 'kerberos' ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <FormField name="thrift.kerberos.realm" label="Realm" required>
              <Input value={thrift.kerberos?.realm ?? ''} onChange={(event) => updateKerberos({ realm: event.target.value })} />
            </FormField>
            <FormField name="thrift.kerberos.kdc" label="KDC" required>
              <Input value={thrift.kerberos?.kdc ?? ''} onChange={(event) => updateKerberos({ kdc: event.target.value })} />
            </FormField>
            <FormField name="thrift.kerberos.principal" label="Principal" required>
              <Input value={thrift.kerberos?.principal ?? ''} onChange={(event) => updateKerberos({ principal: event.target.value })} />
            </FormField>
            <FormField name="thrift.kerberos.keytab" label="Keytab Path">
              <Input value={thrift.kerberos?.keytab ?? ''} onChange={(event) => updateKerberos({ keytab: event.target.value })} placeholder="/etc/security/keytabs/spark.keytab" />
            </FormField>
          </div>
        ) : null}
      </div>

      <Separator />

      <div className="space-y-3 rounded-lg border p-4">
        <div>
          <h4 className="font-medium">Monitoring</h4>
          <p className="text-sm text-muted-foreground">Spark REST endpoints provide job telemetry for lineage and UEBA.</p>
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField name="rest.master_url" label="Master URL" required>
            <Input {...form.register('rest.master_url')} placeholder="http://spark-master:8080" />
          </FormField>
          <FormField name="rest.history_url" label="History Server URL">
            <Input {...form.register('rest.history_url')} placeholder="http://spark-history:18080" />
          </FormField>
        </div>
      </div>
    </div>
  );
}
