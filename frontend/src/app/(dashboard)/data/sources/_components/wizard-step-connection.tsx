'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useMemo } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { Button } from '@/components/ui/button';
import { APIForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/api-form';
import { CSVForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/csv-form';
import { MySQLForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/mysql-form';
import { PostgresForm } from '@/app/(dashboard)/data/sources/_components/connection-forms/postgres-form';
import { S3Form } from '@/app/(dashboard)/data/sources/_components/connection-forms/s3-form';
import {
  apiConnectionSchema,
  csvConnectionSchema,
  mysqlConnectionSchema,
  postgresConnectionSchema,
  s3ConnectionSchema,
  type APIConnectionValues,
  type CSVConnectionValues,
  type MySQLConnectionValues,
  type PostgresConnectionValues,
  type S3ConnectionValues,
  type SourceTypeValue,
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
      return (
        <PostgresConnectionStep
          defaultValues={defaultValues as Partial<PostgresConnectionValues>}
          onSave={(value) => onSave(value)}
        />
      );
    case 'mysql':
      return (
        <MySQLConnectionStep
          defaultValues={defaultValues as Partial<MySQLConnectionValues>}
          onSave={(value) => onSave(value)}
        />
      );
    case 'api':
      return (
        <APIConnectionStep
          defaultValues={defaultValues as Partial<APIConnectionValues>}
          onSave={(value) => onSave(value)}
        />
      );
    case 'csv':
      return (
        <CSVConnectionStep
          defaultValues={defaultValues as Partial<CSVConnectionValues>}
          onSave={(value) => onSave(value)}
        />
      );
    case 's3':
      return (
        <S3ConnectionStep
          defaultValues={defaultValues as Partial<S3ConnectionValues>}
          onSave={(value) => onSave(value)}
        />
      );
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
        <div className="flex justify-end">
          <Button type="submit" disabled={!form.formState.isValid}>
            Continue
          </Button>
        </div>
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
        <div className="flex justify-end">
          <Button type="submit" disabled={!form.formState.isValid}>
            Continue
          </Button>
        </div>
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
        <div className="flex justify-end">
          <Button type="submit" disabled={!form.formState.isValid}>
            Continue
          </Button>
        </div>
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
        <div className="flex justify-end">
          <Button type="submit" disabled={!form.formState.isValid}>
            Continue
          </Button>
        </div>
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
        <div className="flex justify-end">
          <Button type="submit" disabled={!form.formState.isValid}>
            Continue
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
