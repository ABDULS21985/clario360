'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { Button } from '@/components/ui/button';
import { APIForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/api-form';
import { ClickHouseForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/clickhouse-form';
import { CSVForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/csv-form';
import { DagsterForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/dagster-form';
import { DoltForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/dolt-form';
import { HDFSForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/hdfs-form';
import { HiveForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/hive-form';
import { ImpalaForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/impala-form';
import { MySQLForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/mysql-form';
import { PostgresForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/postgres-form';
import { S3Form } from '@/app/(dashboard)/data/sources/_components/connection-forms/s3-form';
import { SparkForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/spark-form';
import {
  apiConnectionSchema,
  clickhouseConnectionSchema,
  csvConnectionSchema,
  dagsterConnectionSchema,
  doltConnectionSchema,
  hdfsConnectionSchema,
  hiveConnectionSchema,
  impalaConnectionSchema,
  mysqlConnectionSchema,
  postgresConnectionSchema,
  s3ConnectionSchema,
  sparkConnectionSchema,
  type APIConnectionValues,
  type ClickHouseConnectionValues,
  type CSVConnectionValues,
  type DagsterConnectionValues,
  type DoltConnectionValues,
  type HDFSConnectionValues,
  type HiveConnectionValues,
  type ImpalaConnectionValues,
  type MySQLConnectionValues,
  type PostgresConnectionValues,
  type S3ConnectionValues,
  type SourceTypeValue,
  type SparkConnectionValues,
} from '@/lib/data-suite/forms';
import type { JsonObject } from '@/lib/data-suite';

interface WizardStepConnectionProps {
  sourceType: SourceTypeValue;
  defaultValues: JsonObject;
  onSave: (value: JsonObject) => void;
}

export function WizardStepConnection({
  sourceType,
  defaultValues,
  onSave,
}: WizardStepConnectionProps) {
  switch (sourceType) {
    case 'postgresql':
      return <PostgresConnectionStep defaultValues={defaultValues as Partial<PostgresConnectionValues>} onSave={onSave} />;
    case 'mysql':
      return <MySQLConnectionStep defaultValues={defaultValues as Partial<MySQLConnectionValues>} onSave={onSave} />;
    case 'api':
      return <APIConnectionStep defaultValues={defaultValues as Partial<APIConnectionValues>} onSave={onSave} />;
    case 'csv':
      return <CSVConnectionStep defaultValues={defaultValues as Partial<CSVConnectionValues>} onSave={onSave} />;
    case 's3':
      return <S3ConnectionStep defaultValues={defaultValues as Partial<S3ConnectionValues>} onSave={onSave} />;
    case 'clickhouse':
      return <ClickHouseConnectionStep defaultValues={defaultValues as Partial<ClickHouseConnectionValues>} onSave={onSave} />;
    case 'impala':
      return <ImpalaConnectionStep defaultValues={defaultValues as Partial<ImpalaConnectionValues>} onSave={onSave} />;
    case 'hive':
      return <HiveConnectionStep defaultValues={defaultValues as Partial<HiveConnectionValues>} onSave={onSave} />;
    case 'hdfs':
      return <HDFSConnectionStep defaultValues={defaultValues as Partial<HDFSConnectionValues>} onSave={onSave} />;
    case 'spark':
      return <SparkConnectionStep defaultValues={defaultValues as Partial<SparkConnectionValues>} onSave={onSave} />;
    case 'dagster':
      return <DagsterConnectionStep defaultValues={defaultValues as Partial<DagsterConnectionValues>} onSave={onSave} />;
    case 'dolt':
      return <DoltConnectionStep defaultValues={defaultValues as Partial<DoltConnectionValues>} onSave={onSave} />;
    default:
      return null;
  }
}

function PostgresConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<PostgresConnectionValues>;
  onSave: (value: PostgresConnectionValues) => void;
}) {
  const form = useForm<PostgresConnectionValues>({
    resolver: zodResolver(postgresConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      host: '',
      port: 5432,
      database: '',
      username: '',
      password: '',
      ssl_mode: 'require',
      schema: 'public',
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <PostgresForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function MySQLConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<MySQLConnectionValues>;
  onSave: (value: MySQLConnectionValues) => void;
}) {
  const form = useForm<MySQLConnectionValues>({
    resolver: zodResolver(mysqlConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      host: '',
      port: 3306,
      database: '',
      username: '',
      password: '',
      tls_mode: 'true',
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <MySQLForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function APIConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<APIConnectionValues>;
  onSave: (value: APIConnectionValues) => void;
}) {
  const form = useForm<APIConnectionValues>({
    resolver: zodResolver(apiConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      base_url: '',
      data_path: '',
      allow_http: false,
      allow_private_addresses: false,
      allowlisted_hosts: [],
      rate_limit: undefined,
      pagination_type: 'offset',
      pagination_config: {},
      query_params: {},
      headers: {},
      auth_type: 'none',
      auth_config: {},
      ...defaultValues,
    } as APIConnectionValues,
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <APIForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function CSVConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<CSVConnectionValues>;
  onSave: (value: CSVConnectionValues) => void;
}) {
  const form = useForm<CSVConnectionValues>({
    resolver: zodResolver(csvConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      minio_endpoint: '',
      bucket: '',
      file_path: '',
      delimiter: ',',
      has_header: true,
      encoding: 'UTF-8',
      quote_char: '"',
      access_key: '',
      secret_key: '',
      use_ssl: true,
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <CSVForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function S3ConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<S3ConnectionValues>;
  onSave: (value: S3ConnectionValues) => void;
}) {
  const form = useForm<S3ConnectionValues>({
    resolver: zodResolver(s3ConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      endpoint: '',
      bucket: '',
      prefix: '',
      region: '',
      access_key: '',
      secret_key: '',
      use_ssl: true,
      allowed_formats: [],
      max_objects: undefined,
      schema_from_first: true,
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <S3Form form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function ClickHouseConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<ClickHouseConnectionValues>;
  onSave: (value: ClickHouseConnectionValues) => void;
}) {
  const form = useForm<ClickHouseConnectionValues>({
    resolver: zodResolver(clickhouseConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      host: '',
      port: 9000,
      database: 'default',
      protocol: 'native',
      username: 'default',
      password: '',
      secure: false,
      compression: true,
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <ClickHouseForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function ImpalaConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<ImpalaConnectionValues>;
  onSave: (value: ImpalaConnectionValues) => void;
}) {
  const form = useForm<ImpalaConnectionValues>({
    resolver: zodResolver(impalaConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      host: '',
      port: 21050,
      database: 'default',
      auth_type: 'noauth',
      username: '',
      password: '',
      use_tls: false,
      audit_log_table: '',
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <ImpalaForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function HiveConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<HiveConnectionValues>;
  onSave: (value: HiveConnectionValues) => void;
}) {
  const form = useForm<HiveConnectionValues>({
    resolver: zodResolver(hiveConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      host: '',
      port: 10000,
      database: 'default',
      auth_type: 'noauth',
      username: '',
      password: '',
      transport_mode: 'binary',
      http_path: '/cliservice',
      use_tls: false,
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <HiveForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function HDFSConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<HDFSConnectionValues>;
  onSave: (value: HDFSConnectionValues) => void;
}) {
  const form = useForm<HDFSConnectionValues>({
    resolver: zodResolver(hdfsConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      name_nodes: [''],
      user: '',
      base_paths: ['/user/hive/warehouse'],
      max_file_size_mb: 100,
      audit_log_path: '',
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <HDFSForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function SparkConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<SparkConnectionValues>;
  onSave: (value: SparkConnectionValues) => void;
}) {
  const form = useForm<SparkConnectionValues>({
    resolver: zodResolver(sparkConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      thrift: {
        host: '',
        port: 10001,
        database: 'default',
        username: '',
        password: '',
        auth_type: 'noauth',
      },
      rest: {
        master_url: '',
        history_url: '',
      },
      query_timeout_seconds: 120,
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <SparkForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function DagsterConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<DagsterConnectionValues>;
  onSave: (value: DagsterConnectionValues) => void;
}) {
  const form = useForm<DagsterConnectionValues>({
    resolver: zodResolver(dagsterConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      graphql_url: '',
      api_token: '',
      workspace: '',
      timeout_seconds: 30,
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <DagsterForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function DoltConnectionStep({
  defaultValues,
  onSave,
}: {
  defaultValues: Partial<DoltConnectionValues>;
  onSave: (value: DoltConnectionValues) => void;
}) {
  const form = useForm<DoltConnectionValues>({
    resolver: zodResolver(doltConnectionSchema),
    mode: 'onChange',
    defaultValues: {
      host: '',
      port: 3306,
      database: '',
      username: '',
      password: '',
      branch: 'main',
      use_tls: false,
      ...defaultValues,
    },
  });

  return (
    <FormProvider {...form}>
      <form className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
        <DoltForm form={form} />
        <ConnectionStepActions isValid={form.formState.isValid} />
      </form>
    </FormProvider>
  );
}

function ConnectionStepActions({ isValid }: { isValid: boolean }) {
  return (
    <div className="flex justify-end">
      <Button type="submit" disabled={!isValid}>
        Continue
      </Button>
    </div>
  );
}
