import { z } from 'zod';

export const sourceTypeSchema = z.enum([
  'postgresql',
  'mysql',
  'api',
  'csv',
  's3',
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

export const sourceConfigureSchema = z.object({
  name: z.string().min(2, 'Source name is required').max(255),
  description: z.string().max(2000).optional(),
  tags: z.array(z.string().min(1)).max(20).default([]),
  sync_frequency: z.string().nullable().default(null),
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
export type SourceConfigureValues = z.infer<typeof sourceConfigureSchema>;
export type QualityRuleFormValues = z.infer<typeof qualityRuleFormSchema>;
export type ContradictionResolutionValues = z.infer<typeof contradictionResolutionSchema>;
export type DarkDataGovernValues = z.infer<typeof darkDataGovernSchema>;
export type SaveQueryValues = z.infer<typeof saveQuerySchema>;
