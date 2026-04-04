'use client';

import { z } from 'zod';
import type {
  DataModel,
  DataSource,
  DiscoveredSchema,
  JsonValue,
  LoadStrategy,
  PipelineType,
  QualityGate,
  Transformation,
} from '@/lib/data-suite';

export const pipelineBasicSchema = z.object({
  name: z.string().min(2, 'Pipeline name is required').max(255),
  description: z.string().max(2000).optional(),
  type: z.enum(['etl', 'elt', 'batch', 'streaming']),
  tags: z.array(z.string().min(1)).max(20).default([]),
});

export const pipelineSourceSchema = z
  .object({
    source_id: z.string().uuid('Source is required'),
    read_mode: z.enum(['table', 'query']).default('table'),
    source_table: z.string().optional(),
    source_query: z.string().optional(),
    incremental_enabled: z.boolean().default(false),
    incremental_field: z.string().optional(),
    incremental_value: z.string().optional(),
  })
  .superRefine((value, context) => {
    if (value.read_mode === 'table' && !value.source_table?.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['source_table'],
        message: 'Select a source table',
      });
    }
    if (value.read_mode === 'query' && !value.source_query?.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['source_query'],
        message: 'Source query is required',
      });
    }
    if (value.incremental_enabled && !value.incremental_field?.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['incremental_field'],
        message: 'Incremental field is required',
      });
    }
  });

export const pipelineTargetSchema = z
  .object({
    target_id: z.string().uuid().nullable().default(null),
    target_table: z.string().optional(),
    target_model_id: z.string().uuid().nullable().default(null),
    load_strategy: z.enum(['append', 'full_replace', 'incremental', 'merge']).default('append'),
    merge_keys: z.array(z.string()).default([]),
  })
  .superRefine((value, context) => {
    if (value.load_strategy === 'merge' && value.merge_keys.length === 0) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['merge_keys'],
        message: 'Choose at least one merge key',
      });
    }
  });

export const pipelineQualityGateSchema = z
  .object({
    id: z.string(),
    name: z.string().min(2, 'Gate name is required'),
    metric: z.enum(['null_percentage', 'unique_percentage', 'row_count_change', 'min_row_count', 'custom']),
    column: z.string().optional(),
    operator: z.string().optional(),
    threshold: z.coerce.number().optional(),
    min_value: z.coerce.number().optional(),
    max_value: z.coerce.number().optional(),
    expression: z.string().optional(),
    severity: z.enum(['critical', 'high', 'medium', 'low']).default('medium'),
    description: z.string().optional(),
  })
  .superRefine((value, context) => {
    if ((value.metric === 'null_percentage' || value.metric === 'unique_percentage') && !value.column?.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['column'],
        message: 'Column is required for this metric',
      });
    }
    if (value.metric === 'custom' && !value.expression?.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['expression'],
        message: 'Expression is required',
      });
    }
    if ((value.metric === 'null_percentage' || value.metric === 'unique_percentage' || value.metric === 'row_count_change') && value.threshold === undefined) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['threshold'],
        message: 'Threshold is required',
      });
    }
    if (value.metric === 'min_row_count' && value.threshold === undefined && value.min_value === undefined) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['threshold'],
        message: 'Minimum row count is required',
      });
    }
  });

export const pipelineScheduleSchema = z
  .object({
    schedule_mode: z.enum(['manual', 'preset', 'custom']).default('manual'),
    schedule_preset: z.enum(['0 * * * *', '0 */6 * * *', '0 */12 * * *', '0 0 * * *', '0 0 * * 0']).nullable().default(null),
    custom_cron: z.string().optional(),
    max_retries: z.coerce.number().int().min(0).max(10).default(3),
    retry_backoff_sec: z.coerce.number().int().min(5).max(3600).default(60),
  })
  .superRefine((value, context) => {
    if (value.schedule_mode === 'preset' && !value.schedule_preset) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['schedule_preset'],
        message: 'Choose a schedule preset',
      });
    }
    if (value.schedule_mode === 'custom' && !value.custom_cron?.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['custom_cron'],
        message: 'Cron expression is required',
      });
    }
  });

export type PipelineBasicValues = z.infer<typeof pipelineBasicSchema>;
export type PipelineSourceValues = z.infer<typeof pipelineSourceSchema>;
export type PipelineTargetValues = z.infer<typeof pipelineTargetSchema>;
export type PipelineScheduleValues = z.infer<typeof pipelineScheduleSchema>;
export type PipelineQualityGateDraft = z.infer<typeof pipelineQualityGateSchema>;

export type FilterOperator =
  | '=='
  | '!='
  | '>'
  | '<'
  | '>='
  | '<='
  | 'in'
  | 'not_in'
  | 'like'
  | 'is_null'
  | 'is_not_null';

export interface FilterConditionDraft {
  id: string;
  column: string;
  operator: FilterOperator;
  value: string;
  secondaryValue: string;
}

export interface MapValueDraft {
  id: string;
  key: string;
  value: string;
}

export interface AggregateDefinitionDraft {
  id: string;
  column: string;
  function: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'count_distinct';
  alias: string;
}

export interface RenameTransformDraft {
  id: string;
  type: 'rename';
  config: {
    from: string;
    to: string;
  };
}

export interface CastTransformDraft {
  id: string;
  type: 'cast';
  config: {
    column: string;
    to_type: 'string' | 'integer' | 'float' | 'boolean' | 'datetime';
  };
}

export interface FilterTransformDraft {
  id: string;
  type: 'filter';
  config: {
    combinator: 'AND' | 'OR';
    conditions: FilterConditionDraft[];
  };
}

export interface MapValuesTransformDraft {
  id: string;
  type: 'map_values';
  config: {
    column: string;
    mappings: MapValueDraft[];
    default_value: string;
  };
}

export interface DeriveTransformDraft {
  id: string;
  type: 'derive';
  config: {
    name: string;
    expression: string;
  };
}

export interface DeduplicateTransformDraft {
  id: string;
  type: 'deduplicate';
  config: {
    key_columns: string[];
    keep: 'latest' | 'first';
    order_by: string;
  };
}

export interface AggregateTransformDraft {
  id: string;
  type: 'aggregate';
  config: {
    group_by: string[];
    aggregations: AggregateDefinitionDraft[];
  };
}

export type PipelineTransformDraft =
  | RenameTransformDraft
  | CastTransformDraft
  | FilterTransformDraft
  | MapValuesTransformDraft
  | DeriveTransformDraft
  | DeduplicateTransformDraft
  | AggregateTransformDraft;

export interface PipelineQualityValues {
  quality_gates: PipelineQualityGateDraft[];
  fail_on_quality_gate: boolean;
}

export interface PipelineWizardState {
  step: number;
  basic: PipelineBasicValues;
  source: PipelineSourceValues;
  sourceSchema: DiscoveredSchema | null;
  selectedSource: DataSource | null;
  selectedModel: DataModel | null;
  transforms: PipelineTransformDraft[];
  target: PipelineTargetValues;
  quality: PipelineQualityValues;
  schedule: PipelineScheduleValues;
  previewBeforeRows: Array<Record<string, JsonValue>>;
  previewAfterRows: Array<Record<string, JsonValue>>;
  previewError: string | null;
}

export interface PipelineCreatePayload {
  name: string;
  description?: string;
  type: PipelineType;
  source_id: string;
  target_id?: string | null;
  schedule?: string | null;
  tags: string[];
  config: {
    source_table?: string;
    source_query?: string;
    target_table?: string;
    target_model_id?: string | null;
    incremental_field?: string;
    incremental_value?: string | null;
    transformations?: Transformation[];
    quality_gates?: QualityGate[];
    fail_on_quality_gate?: boolean;
    load_strategy?: LoadStrategy;
    merge_keys?: string[];
    max_retries?: number;
    retry_backoff_sec?: number;
  };
}

export const PIPELINE_SCHEDULE_PRESETS = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Every 12 hours', value: '0 */12 * * *' },
  { label: 'Daily', value: '0 0 * * *' },
  { label: 'Weekly', value: '0 0 * * 0' },
] as const;

