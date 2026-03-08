import { z } from 'zod';

export const sourceTypeSchema = z.enum([
  'postgresql',
  'mysql',
  'api',
  'csv',
  's3',
  'clickhouse',
  'impala',
  'hive',
  'hdfs',
  'spark',
  'dagster',
  'dolt',
  'stream',
]);

export const postgresConnectionSchema = z.object({
  host: z.string().min(1, 'Host is required'),
  port: z.coerce.number().int().min(1).max(65535).default(5432),
  database: z.string().min(1, 'Database is required'),
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
  ssl_mode: z.enum(['disable', 'require', 'verify-ca', 'verify-full']).default('require'),
  schema: z.string().default('public'),
});

export const mysqlConnectionSchema = z.object({
  host: z.string().min(1, 'Host is required'),
  port: z.coerce.number().int().min(1).max(65535).default(3306),
  database: z.string().min(1, 'Database is required'),
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
  tls_mode: z.enum(['true', 'false', 'skip-verify', 'preferred']).default('true'),
});

const basicAuthSchema = z.object({
  auth_type: z.literal('basic'),
  auth_config: z.object({
    username: z.string().min(1, 'Username is required'),
    password: z.string().min(1, 'Password is required'),
  }),
});

const bearerAuthSchema = z.object({
  auth_type: z.literal('bearer'),
  auth_config: z.object({
    token: z.string().min(1, 'Token is required'),
  }),
});

const apiKeyAuthSchema = z.object({
  auth_type: z.literal('api_key'),
  auth_config: z.object({
    key_name: z.string().min(1, 'Key name is required'),
    key_value: z.string().min(1, 'Key value is required'),
    location: z.enum(['header', 'query']).default('header'),
  }),
});

const oauthAuthSchema = z.object({
  auth_type: z.literal('oauth2'),
  auth_config: z.object({
    token_url: z.string().url('Token URL must be valid'),
    client_id: z.string().min(1, 'Client ID is required'),
    client_secret: z.string().min(1, 'Client secret is required'),
    scope: z.string().optional(),
  }),
});

const noAuthSchema = z.object({
  auth_type: z.literal('none'),
  auth_config: z.object({}).default({}),
});

export const apiConnectionSchema = z
  .object({
    base_url: z
      .string()
      .url('Base URL must be valid')
      .refine((value) => value.startsWith('https://'), 'Base URL must start with https://'),
    data_path: z.string().optional(),
    allow_http: z.boolean().default(false),
    allow_private_addresses: z.boolean().default(false),
    allowlisted_hosts: z.array(z.string()).default([]),
    rate_limit: z.coerce.number().int().positive().optional(),
    pagination_type: z.enum(['offset', 'cursor', 'page', 'link_header']).default('offset'),
    pagination_config: z.record(z.string(), z.string()).default({}),
    query_params: z.record(z.string(), z.string()).default({}),
    headers: z.record(z.string(), z.string()).default({}),
  })
  .and(z.discriminatedUnion('auth_type', [noAuthSchema, basicAuthSchema, bearerAuthSchema, apiKeyAuthSchema, oauthAuthSchema]));

export const csvConnectionSchema = z.object({
  minio_endpoint: z.string().min(1, 'Endpoint is required'),
  bucket: z.string().min(1, 'Bucket is required'),
  file_path: z.string().min(1, 'File path is required'),
  delimiter: z.string().default(','),
  has_header: z.boolean().default(true),
  encoding: z.enum(['UTF-8', 'Latin-1', 'Windows-1252']).default('UTF-8'),
  quote_char: z.string().default('"'),
  access_key: z.string().min(1, 'Access key is required'),
  secret_key: z.string().min(1, 'Secret key is required'),
  use_ssl: z.boolean().default(true),
  upload_file_id: z.string().optional(),
  upload_file_name: z.string().optional(),
});

export const s3ConnectionSchema = z.object({
  endpoint: z.string().min(1, 'Endpoint is required'),
  bucket: z.string().min(1, 'Bucket is required'),
  prefix: z.string().optional(),
  region: z.string().optional(),
  access_key: z.string().min(1, 'Access key is required'),
  secret_key: z.string().min(1, 'Secret key is required'),
  use_ssl: z.boolean().default(true),
  allowed_formats: z.array(z.string()).default([]),
  max_objects: z.coerce.number().int().positive().optional(),
  schema_from_first: z.boolean().default(true),
});

export const kerberosConfigSchema = z.object({
  realm: z.string().min(1, 'Realm is required'),
  kdc: z.string().min(1, 'KDC is required'),
  principal: z.string().min(1, 'Principal is required'),
  keytab: z.string().optional(),
});

export const clickhouseConnectionSchema = z.object({
  host: z.string().min(1, 'Host is required'),
  port: z.coerce.number().int().min(1).max(65535).default(9000),
  database: z.string().min(1, 'Database is required'),
  protocol: z.enum(['native', 'http']).default('native'),
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
  secure: z.boolean().default(false),
  compression: z.boolean().default(true),
  cluster: z.string().optional(),
  max_open_conns: z.coerce.number().int().positive().optional(),
  max_idle_conns: z.coerce.number().int().positive().optional(),
  dial_timeout_seconds: z.coerce.number().int().positive().optional(),
  read_timeout_seconds: z.coerce.number().int().positive().optional(),
});

export const impalaConnectionSchema = z
  .object({
    host: z.string().min(1, 'Host is required'),
    port: z.coerce.number().int().min(1).max(65535).default(21050),
    database: z.string().default('default'),
    auth_type: z.enum(['noauth', 'ldap', 'kerberos']).default('noauth'),
    username: z.string().optional(),
    password: z.string().optional(),
    use_tls: z.boolean().default(false),
    query_timeout_seconds: z.coerce.number().int().positive().optional(),
    audit_log_table: z.string().optional(),
    kerberos: kerberosConfigSchema.optional(),
  })
  .superRefine((value, ctx) => {
    if (value.auth_type === 'ldap') {
      if (!value.username?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['username'], message: 'Username is required for LDAP auth' });
      }
      if (!value.password?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['password'], message: 'Password is required for LDAP auth' });
      }
    }
    if (value.auth_type === 'kerberos' && !value.kerberos) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['kerberos'], message: 'Kerberos settings are required' });
    }
  });

export const hiveConnectionSchema = z
  .object({
    host: z.string().min(1, 'Host is required'),
    port: z.coerce.number().int().min(1).max(65535).default(10000),
    database: z.string().default('default'),
    auth_type: z.enum(['noauth', 'plain', 'kerberos']).default('noauth'),
    username: z.string().optional(),
    password: z.string().optional(),
    transport_mode: z.enum(['binary', 'http']).default('binary'),
    http_path: z.string().default('/cliservice'),
    use_tls: z.boolean().default(false),
    query_timeout_seconds: z.coerce.number().int().positive().optional(),
    kerberos: kerberosConfigSchema.optional(),
  })
  .superRefine((value, ctx) => {
    if (value.auth_type === 'plain') {
      if (!value.username?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['username'], message: 'Username is required for username/password auth' });
      }
      if (!value.password?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['password'], message: 'Password is required for username/password auth' });
      }
    }
    if (value.auth_type === 'kerberos' && !value.kerberos) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['kerberos'], message: 'Kerberos settings are required' });
    }
    if (value.transport_mode === 'http' && !value.http_path.trim()) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['http_path'], message: 'HTTP path is required for HTTP transport mode' });
    }
  });

export const hdfsConnectionSchema = z.object({
  name_nodes: z.array(z.string().min(1, 'NameNode address is required')).min(1, 'At least one NameNode is required'),
  user: z.string().optional(),
  base_paths: z.array(z.string().min(1)).default(['/user/hive/warehouse']),
  max_file_size_mb: z.coerce.number().int().positive().default(100),
  audit_log_path: z.string().optional(),
  kerberos: kerberosConfigSchema.optional(),
});

const sparkThriftSchema = z
  .object({
    host: z.string().min(1, 'Thrift host is required'),
    port: z.coerce.number().int().min(1).max(65535).default(10001),
    database: z.string().default('default'),
    username: z.string().optional(),
    password: z.string().optional(),
    auth_type: z.enum(['noauth', 'plain', 'kerberos']).default('noauth'),
    kerberos: kerberosConfigSchema.optional(),
  })
  .superRefine((value, ctx) => {
    if (value.auth_type === 'plain') {
      if (!value.username?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['username'], message: 'Username is required for username/password auth' });
      }
      if (!value.password?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['password'], message: 'Password is required for username/password auth' });
      }
    }
    if (value.auth_type === 'kerberos' && !value.kerberos) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['kerberos'], message: 'Kerberos settings are required' });
    }
  });

export const sparkConnectionSchema = z.object({
  thrift: sparkThriftSchema.optional(),
  rest: z.object({
    master_url: z.string().url('Master URL must be valid'),
    history_url: z.string().url('History URL must be valid').optional().or(z.literal('')),
  }),
  query_timeout_seconds: z.coerce.number().int().positive().default(120),
  max_open_conns: z.coerce.number().int().positive().optional(),
  max_idle_conns: z.coerce.number().int().positive().optional(),
});

export const dagsterConnectionSchema = z.object({
  graphql_url: z.string().url('GraphQL URL must be valid'),
  api_token: z.string().optional(),
  workspace: z.string().optional(),
  timeout_seconds: z.coerce.number().int().positive().default(30),
});

export const doltConnectionSchema = z.object({
  host: z.string().min(1, 'Host is required'),
  port: z.coerce.number().int().min(1).max(65535).default(3306),
  database: z.string().min(1, 'Database is required'),
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
  branch: z.string().default('main'),
  use_tls: z.boolean().default(false),
});

export const sourceConfigureSchema = z.object({
  name: z.string().min(2, 'Source name is required').max(255),
  description: z.string().max(2000).optional(),
  tags: z.array(z.string().min(1)).max(20).default([]),
  sync_frequency: z.string().nullable().default(null),
});

export const testSourceConfigSchema = z.object({
  type: sourceTypeSchema,
  connection_config: z.record(z.string(), z.unknown()),
});

export const qualityRuleFormSchema = z.object({
  model_id: z.string().uuid('Model is required'),
  name: z.string().min(2, 'Rule name is required'),
  description: z.string().optional(),
  rule_type: z.enum([
    'not_null',
    'unique',
    'range',
    'regex',
    'referential',
    'enum',
    'freshness',
    'row_count',
    'custom_sql',
    'statistical',
  ]),
  severity: z.enum(['critical', 'high', 'medium', 'low']),
  column_name: z.string().optional(),
  config: z.record(z.string(), z.unknown()).default({}),
  schedule: z.string().optional(),
  enabled: z.boolean().default(true),
  tags: z.array(z.string()).default([]),
});

export const contradictionResolutionSchema = z.object({
  resolution_action: z.enum([
    'source_a_corrected',
    'source_b_corrected',
    'both_corrected',
    'data_reconciled',
    'accepted_as_is',
    'false_positive',
  ]),
  resolution_notes: z.string().min(3, 'Resolution notes are required'),
});

export const darkDataGovernSchema = z.object({
  model_name: z.string().min(2, 'Model name is required'),
  assign_quality_rules: z.boolean().default(true),
});

export const saveQuerySchema = z.object({
  name: z.string().min(2, 'Name is required'),
  description: z.string().optional(),
  visibility: z.enum(['private', 'team', 'organization']).default('private'),
});

export type SourceTypeValue = z.infer<typeof sourceTypeSchema>;
export type PostgresConnectionValues = z.infer<typeof postgresConnectionSchema>;
export type MySQLConnectionValues = z.infer<typeof mysqlConnectionSchema>;
export type APIConnectionValues = z.infer<typeof apiConnectionSchema>;
export type CSVConnectionValues = z.infer<typeof csvConnectionSchema>;
export type S3ConnectionValues = z.infer<typeof s3ConnectionSchema>;
export type KerberosConfigValues = z.infer<typeof kerberosConfigSchema>;
export type ClickHouseConnectionValues = z.infer<typeof clickhouseConnectionSchema>;
export type ImpalaConnectionValues = z.infer<typeof impalaConnectionSchema>;
export type HiveConnectionValues = z.infer<typeof hiveConnectionSchema>;
export type HDFSConnectionValues = z.infer<typeof hdfsConnectionSchema>;
export type SparkConnectionValues = z.infer<typeof sparkConnectionSchema>;
export type DagsterConnectionValues = z.infer<typeof dagsterConnectionSchema>;
export type DoltConnectionValues = z.infer<typeof doltConnectionSchema>;
export type SourceConfigureValues = z.infer<typeof sourceConfigureSchema>;
export type TestSourceConfigValues = z.infer<typeof testSourceConfigSchema>;
export type QualityRuleFormValues = z.infer<typeof qualityRuleFormSchema>;
export type ContradictionResolutionValues = z.infer<typeof contradictionResolutionSchema>;
export type DarkDataGovernValues = z.infer<typeof darkDataGovernSchema>;
export type SaveQueryValues = z.infer<typeof saveQuerySchema>;
