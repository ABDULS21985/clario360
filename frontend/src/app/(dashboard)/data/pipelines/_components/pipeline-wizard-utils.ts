'use client';

import type {
  DiscoveredColumn,
  DiscoveredSchema,
  DiscoveredTable,
  JsonObject,
  JsonValue,
  QualityGate,
  Transformation,
} from '@/lib/data-suite';
import { humanizeCronOrFrequency } from '@/lib/data-suite/utils';
import type {
  AggregateDefinitionDraft,
  FilterConditionDraft,
  PipelineCreatePayload,
  PipelineQualityGateDraft,
  PipelineTransformDraft,
  PipelineWizardState,
} from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

function createId(prefix: string): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

export function createInitialPipelineWizardState(): PipelineWizardState {
  return {
    step: 1,
    basic: {
      name: '',
      description: '',
      type: 'etl',
      tags: [],
    },
    source: {
      source_id: '',
      read_mode: 'table',
      source_table: '',
      source_query: '',
      incremental_enabled: false,
      incremental_field: '',
      incremental_value: '',
    },
    sourceSchema: null,
    selectedSource: null,
    selectedModel: null,
    transforms: [],
    target: {
      target_id: null,
      target_table: '',
      target_model_id: null,
      load_strategy: 'append',
      merge_keys: [],
    },
    quality: {
      quality_gates: [],
      fail_on_quality_gate: true,
    },
    schedule: {
      schedule_mode: 'manual',
      schedule_preset: null,
      custom_cron: '',
      max_retries: 3,
      retry_backoff_sec: 60,
    },
    previewBeforeRows: [],
    previewAfterRows: [],
    previewError: null,
  };
}

export function createEmptyFilterCondition(): FilterConditionDraft {
  return {
    id: createId('filter'),
    column: '',
    operator: '==',
    value: '',
    secondaryValue: '',
  };
}

export function createEmptyAggregation(): AggregateDefinitionDraft {
  return {
    id: createId('agg'),
    column: '',
    function: 'count',
    alias: '',
  };
}

export function createEmptyQualityGate(): PipelineQualityGateDraft {
  return {
    id: createId('gate'),
    name: '',
    metric: 'null_percentage',
    column: '',
    operator: '<=',
    threshold: 0,
    min_value: undefined,
    max_value: undefined,
    expression: '',
    severity: 'medium',
    description: '',
  };
}

export function createEmptyTransform(type: PipelineTransformDraft['type']): PipelineTransformDraft {
  switch (type) {
    case 'rename':
      return {
        id: createId('transform'),
        type,
        config: { from: '', to: '' },
      };
    case 'cast':
      return {
        id: createId('transform'),
        type,
        config: { column: '', to_type: 'string' },
      };
    case 'filter':
      return {
        id: createId('transform'),
        type,
        config: { combinator: 'AND', conditions: [createEmptyFilterCondition()] },
      };
    case 'map_values':
      return {
        id: createId('transform'),
        type,
        config: {
          column: '',
          mappings: [{ id: createId('mapping'), key: '', value: '' }],
          default_value: '',
        },
      };
    case 'derive':
      return {
        id: createId('transform'),
        type,
        config: { name: '', expression: '' },
      };
    case 'deduplicate':
      return {
        id: createId('transform'),
        type,
        config: { key_columns: [], keep: 'latest', order_by: '' },
      };
    case 'aggregate':
      return {
        id: createId('transform'),
        type,
        config: { group_by: [], aggregations: [createEmptyAggregation()] },
      };
  }
}

export function findTable(schema: DiscoveredSchema | null, tableName?: string): DiscoveredTable | null {
  if (!schema || !tableName) {
    return null;
  }
  return schema.tables.find((table) => qualifiedTableName(table) === tableName) ?? null;
}

export function qualifiedTableName(table: DiscoveredTable): string {
  return table.schema_name ? `${table.schema_name}.${table.name}` : table.name;
}

export function tableColumnNames(table: DiscoveredTable | null): string[] {
  return table?.columns.map((column) => column.name) ?? [];
}

export function buildSampleRows(table: DiscoveredTable | null, limit = 5): Array<Record<string, JsonValue>> {
  if (!table) {
    return [];
  }
  const rowCount = Math.min(
    limit,
    Math.max(
      1,
      ...table.columns.map((column) => column.sample_values?.length ?? 0),
    ),
  );

  return Array.from({ length: rowCount }, (_, index) => {
    const row: Record<string, JsonValue> = {};
    table.columns.forEach((column) => {
      row[column.name] = column.sample_values?.[index] ?? column.sample_values?.[0] ?? null;
    });
    return row;
  });
}

export function serializePipelinePayload(state: PipelineWizardState): PipelineCreatePayload {
  const schedule = resolveScheduleValue(
    state.schedule.schedule_mode,
    state.schedule.schedule_preset,
    state.schedule.custom_cron,
  );
  return {
    name: state.basic.name,
    description: state.basic.description,
    type: state.basic.type,
    source_id: state.source.source_id,
    target_id: state.target.target_id,
    schedule,
    tags: state.basic.tags,
    config: {
      source_table: state.source.read_mode === 'table' ? state.source.source_table : undefined,
      source_query: state.source.read_mode === 'query' ? state.source.source_query : undefined,
      target_table: state.target.target_table || undefined,
      target_model_id: state.target.target_model_id,
      incremental_field: state.source.incremental_enabled ? state.source.incremental_field || undefined : undefined,
      incremental_value: state.source.incremental_enabled ? state.source.incremental_value || null : undefined,
      transformations: state.transforms.map(serializeTransform),
      quality_gates: state.quality.quality_gates.map(serializeQualityGate),
      fail_on_quality_gate: state.quality.fail_on_quality_gate,
      load_strategy: state.target.load_strategy,
      merge_keys: state.target.load_strategy === 'merge' ? state.target.merge_keys : [],
      max_retries: state.schedule.max_retries,
      retry_backoff_sec: state.schedule.retry_backoff_sec,
    },
  };
}

export function describeSchedule(mode: PipelineWizardState['schedule']['schedule_mode'], preset: string | null, customCron: string): string {
  const value = resolveScheduleValue(mode, preset, customCron);
  return humanizeCronOrFrequency(value);
}

export function serializeQualityGate(gate: PipelineQualityGateDraft): QualityGate {
  return {
    name: gate.name,
    metric: gate.metric,
    column: gate.column || undefined,
    operator: gate.operator || undefined,
    threshold: gate.metric === 'min_row_count' ? gate.min_value ?? gate.threshold : gate.threshold,
    min_value: gate.min_value,
    max_value: gate.max_value,
    expression: gate.expression || undefined,
    severity: gate.severity,
    description: gate.description || undefined,
  };
}

export function serializeTransform(transform: PipelineTransformDraft): Transformation {
  switch (transform.type) {
    case 'rename':
      return {
        type: transform.type,
        config: {
          from: transform.config.from,
          to: transform.config.to,
        },
      };
    case 'cast':
      return {
        type: transform.type,
        config: {
          column: transform.config.column,
          to_type: transform.config.to_type,
        },
      };
    case 'filter':
      return {
        type: transform.type,
        config: {
          expression: buildFilterExpression(transform.config.conditions, transform.config.combinator),
        },
      };
    case 'map_values': {
      const mapping: JsonObject = {};
      transform.config.mappings.forEach((item) => {
        if (item.key.trim()) {
          mapping[item.key] = item.value;
        }
      });
      return {
        type: transform.type,
        config: {
          column: transform.config.column,
          mapping,
          default: transform.config.default_value || null,
        },
      };
    }
    case 'derive':
      return {
        type: transform.type,
        config: {
          name: transform.config.name,
          expression: transform.config.expression,
        },
      };
    case 'deduplicate':
      return {
        type: transform.type,
        config: {
          key_columns: transform.config.key_columns,
          keep: transform.config.keep,
          order_by: transform.config.order_by,
        },
      };
    case 'aggregate':
      return {
        type: transform.type,
        config: {
          group_by: transform.config.group_by,
          aggregations: transform.config.aggregations.map((aggregation) => ({
            column: aggregation.column,
            function: aggregation.function,
            alias: aggregation.alias,
          })),
        },
      };
  }
}

export function summarizeTransform(transform: PipelineTransformDraft): string {
  switch (transform.type) {
    case 'rename':
      return transform.config.from && transform.config.to
        ? `Rename '${transform.config.from}' → '${transform.config.to}'`
        : 'Rename column';
    case 'cast':
      return transform.config.column
        ? `Cast '${transform.config.column}' to ${transform.config.to_type}`
        : 'Cast column type';
    case 'filter':
      return transform.config.conditions.length > 0
        ? `Filter: ${buildFilterExpression(transform.config.conditions, transform.config.combinator)}`
        : 'Filter rows';
    case 'map_values':
      return transform.config.column
        ? `Map '${transform.config.column}': ${transform.config.mappings.filter((item) => item.key || item.value).length} value mappings`
        : 'Map values';
    case 'derive':
      return transform.config.name && transform.config.expression
        ? `Derive '${transform.config.name}' = ${transform.config.expression}`
        : 'Derived column';
    case 'deduplicate':
      return transform.config.key_columns.length > 0
        ? `Deduplicate by [${transform.config.key_columns.join(', ')}]`
        : 'Deduplicate rows';
    case 'aggregate':
      return transform.config.aggregations.length > 0
        ? `Group by [${transform.config.group_by.join(', ')}]`
        : 'Aggregate rows';
  }
}

export function validateTransform(transform: PipelineTransformDraft): string | null {
  switch (transform.type) {
    case 'rename':
      return transform.config.from && transform.config.to ? null : 'Rename requires both source and target columns.';
    case 'cast':
      return transform.config.column ? null : 'Cast requires a column.';
    case 'filter':
      return transform.config.conditions.every((condition) => validateFilterCondition(condition))
        ? null
        : 'Every filter condition must include a column and any required values.';
    case 'map_values':
      return transform.config.column ? null : 'Map values requires a target column.';
    case 'derive':
      return transform.config.name && transform.config.expression ? null : 'Derived column requires a name and expression.';
    case 'deduplicate':
      return transform.config.key_columns.length > 0 ? null : 'Deduplicate requires at least one key column.';
    case 'aggregate':
      return transform.config.aggregations.every((aggregation) => aggregation.function && (aggregation.column || aggregation.function === 'count'))
        ? null
        : 'Aggregate requires at least one valid aggregation definition.';
  }
}

export function runPreview(rows: Array<Record<string, JsonValue>>, transforms: PipelineTransformDraft[]): {
  rows: Array<Record<string, JsonValue>>;
  error: string | null;
} {
  try {
    const output = transforms.reduce<Array<Record<string, JsonValue>>>((currentRows, transform) => {
      const validationError = validateTransform(transform);
      if (validationError) {
        throw new Error(validationError);
      }
      return applyTransform(currentRows, transform);
    }, cloneRows(rows));

    return { rows: output.slice(0, 5), error: null };
  } catch (error) {
    return {
      rows: [],
      error: error instanceof Error ? error.message : 'Failed to preview transformations.',
    };
  }
}

function resolveScheduleValue(
  mode: PipelineWizardState['schedule']['schedule_mode'],
  preset: string | null,
  customCron?: string,
): string | null {
  if (mode === 'preset') {
    return preset;
  }
  if (mode === 'custom') {
    return customCron?.trim() || null;
  }
  return null;
}

function buildFilterExpression(conditions: FilterConditionDraft[], combinator: 'AND' | 'OR'): string {
  return conditions
    .filter((condition) => condition.column.trim())
    .map((condition) => conditionToExpression(condition))
    .filter((value) => value.length > 0)
    .join(` ${combinator} `);
}

function conditionToExpression(condition: FilterConditionDraft): string {
  const column = condition.column.trim();
  if (!column) {
    return '';
  }
  switch (condition.operator) {
    case 'is_null':
      return `${column} == null`;
    case 'is_not_null':
      return `${column} != null`;
    case 'in':
    case 'not_in': {
      const values = condition.value
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean)
        .map((item) => `'${escapeExpressionString(item)}'`)
        .join(', ');
      return `${column} ${condition.operator.toUpperCase()} (${values})`;
    }
    case 'like':
      return `${column} LIKE '${escapeExpressionString(condition.value)}'`;
    default:
      return `${column} ${condition.operator} ${formatExpressionValue(condition.value)}`;
  }
}

function formatExpressionValue(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "''";
  }
  if (!Number.isNaN(Number(trimmed))) {
    return trimmed;
  }
  if (trimmed === 'true' || trimmed === 'false' || trimmed === 'null') {
    return trimmed;
  }
  return `'${escapeExpressionString(trimmed)}'`;
}

function escapeExpressionString(value: string): string {
  return value.replace(/'/g, "\\'");
}

function validateFilterCondition(condition: FilterConditionDraft): boolean {
  if (!condition.column.trim()) {
    return false;
  }
  if (condition.operator === 'is_null' || condition.operator === 'is_not_null') {
    return true;
  }
  if (condition.operator === 'in' || condition.operator === 'not_in') {
    return condition.value.trim().length > 0;
  }
  return condition.value.trim().length > 0;
}

function applyTransform(rows: Array<Record<string, JsonValue>>, transform: PipelineTransformDraft): Array<Record<string, JsonValue>> {
  switch (transform.type) {
    case 'rename':
      return rows.map((row) => {
        const next = { ...row };
        next[transform.config.to] = row[transform.config.from] ?? null;
        delete next[transform.config.from];
        return next;
      });
    case 'cast':
      return rows.map((row) => ({
        ...row,
        [transform.config.column]: castPreviewValue(row[transform.config.column], transform.config.to_type),
      }));
    case 'filter':
      return rows.filter((row) => evaluateFilterRow(row, transform.config.conditions, transform.config.combinator));
    case 'map_values': {
      const mapping = new Map(
        transform.config.mappings
          .filter((item) => item.key.trim())
          .map((item) => [item.key, item.value] as const),
      );
      return rows.map((row) => {
        const current = `${row[transform.config.column] ?? ''}`;
        const mapped = mapping.get(current);
        return {
          ...row,
          [transform.config.column]: mapped ?? (transform.config.default_value || row[transform.config.column]),
        };
      });
    }
    case 'derive':
      return rows.map((row) => ({
        ...row,
        [transform.config.name]: evaluateDeriveExpression(row, transform.config.expression),
      }));
    case 'deduplicate':
      return deduplicateRows(rows, transform.config.key_columns, transform.config.keep, transform.config.order_by);
    case 'aggregate':
      return aggregateRows(rows, transform.config.group_by, transform.config.aggregations);
  }
}

function castPreviewValue(value: JsonValue, toType: string): JsonValue {
  if (value === null || value === undefined) {
    return null;
  }
  switch (toType) {
    case 'string':
      return `${value}`;
    case 'integer': {
      const parsed = Number.parseInt(`${value}`, 10);
      return Number.isNaN(parsed) ? null : parsed;
    }
    case 'float': {
      const parsed = Number.parseFloat(`${value}`);
      return Number.isNaN(parsed) ? null : parsed;
    }
    case 'boolean': {
      const normalized = `${value}`.trim().toLowerCase();
      if (['true', '1', 'yes', 'y'].includes(normalized)) {
        return true;
      }
      if (['false', '0', 'no', 'n'].includes(normalized)) {
        return false;
      }
      return null;
    }
    case 'datetime': {
      const parsed = new Date(`${value}`);
      return Number.isNaN(parsed.getTime()) ? null : parsed.toISOString();
    }
    default:
      return value;
  }
}

function evaluateFilterRow(
  row: Record<string, JsonValue>,
  conditions: FilterConditionDraft[],
  combinator: 'AND' | 'OR',
): boolean {
  const evaluations = conditions
    .filter((condition) => condition.column.trim())
    .map((condition) => evaluateCondition(row, condition));
  if (evaluations.length === 0) {
    return true;
  }
  return combinator === 'AND' ? evaluations.every(Boolean) : evaluations.some(Boolean);
}

function evaluateCondition(row: Record<string, JsonValue>, condition: FilterConditionDraft): boolean {
  const left = row[condition.column];
  switch (condition.operator) {
    case '==':
      return `${left ?? ''}` === condition.value;
    case '!=':
      return `${left ?? ''}` !== condition.value;
    case '>':
      return compareValues(left, condition.value) > 0;
    case '<':
      return compareValues(left, condition.value) < 0;
    case '>=':
      return compareValues(left, condition.value) >= 0;
    case '<=':
      return compareValues(left, condition.value) <= 0;
    case 'in':
      return condition.value.split(',').map((item) => item.trim()).includes(`${left ?? ''}`);
    case 'not_in':
      return !condition.value.split(',').map((item) => item.trim()).includes(`${left ?? ''}`);
    case 'like':
      return likeValue(`${left ?? ''}`, condition.value);
    case 'is_null':
      return left === null || left === undefined || `${left}`.trim() === '';
    case 'is_not_null':
      return left !== null && left !== undefined && `${left}`.trim() !== '';
  }
}

function compareValues(left: JsonValue, right: string): number {
  const leftNumber = Number(`${left}`);
  const rightNumber = Number(right);
  if (!Number.isNaN(leftNumber) && !Number.isNaN(rightNumber)) {
    return leftNumber - rightNumber;
  }
  const leftDate = Date.parse(`${left ?? ''}`);
  const rightDate = Date.parse(right);
  if (!Number.isNaN(leftDate) && !Number.isNaN(rightDate)) {
    return leftDate - rightDate;
  }
  return `${left ?? ''}`.localeCompare(right);
}

function likeValue(value: string, pattern: string): boolean {
  const escaped = pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace(/%/g, '.*').replace(/_/g, '.');
  return new RegExp(`^${escaped}$`, 'i').test(value);
}

function evaluateDeriveExpression(row: Record<string, JsonValue>, expression: string): JsonValue {
  const helpers: PreviewHelpers = {
    TRIM: (value: JsonValue) => (value === null || value === undefined ? null : `${value}`.trim()),
    UPPER: (value: JsonValue) => (value === null || value === undefined ? null : `${value}`.toUpperCase()),
    LOWER: (value: JsonValue) => (value === null || value === undefined ? null : `${value}`.toLowerCase()),
    CONCAT: (...values: JsonValue[]) => values.filter((value) => value !== null && value !== undefined).join(''),
    COALESCE: (...values: JsonValue[]) => values.find((value) => value !== null && value !== undefined && `${value}` !== '') ?? null,
  };

  const columns = Object.keys(row).sort((left, right) => right.length - left.length);
  let script = expression;
  columns.forEach((column) => {
    script = script.replace(new RegExp(`\\b${escapeRegExp(column)}\\b`, 'g'), `row["${column}"]`);
  });

  const fn = new Function(
    'row',
    'helpers',
    `const { TRIM, UPPER, LOWER, CONCAT, COALESCE } = helpers; return (${script});`,
  ) as (row: Record<string, JsonValue>, helpers: PreviewHelpers) => JsonValue;

  return fn(row, helpers);
}

function deduplicateRows(
  rows: Array<Record<string, JsonValue>>,
  keyColumns: string[],
  keep: 'latest' | 'first',
  orderBy: string,
): Array<Record<string, JsonValue>> {
  const groups = new Map<string, Array<Record<string, JsonValue>>>();
  rows.forEach((row) => {
    const key = keyColumns.map((column) => `${row[column] ?? ''}`).join('|');
    const bucket = groups.get(key) ?? [];
    bucket.push(row);
    groups.set(key, bucket);
  });

  return Array.from(groups.values()).map((group) => {
    if (!orderBy) {
      return group[0];
    }
    const sorted = [...group].sort((left, right) => compareValues(left[orderBy], `${right[orderBy] ?? ''}`));
    return keep === 'first' ? sorted[0] : sorted[sorted.length - 1];
  });
}

function aggregateRows(
  rows: Array<Record<string, JsonValue>>,
  groupBy: string[],
  aggregations: AggregateDefinitionDraft[],
): Array<Record<string, JsonValue>> {
  const groups = new Map<string, Array<Record<string, JsonValue>>>();
  rows.forEach((row) => {
    const key = groupBy.map((column) => `${row[column] ?? ''}`).join('|');
    const bucket = groups.get(key) ?? [];
    bucket.push(row);
    groups.set(key, bucket);
  });

  return Array.from(groups.values()).map((group) => {
    const result: Record<string, JsonValue> = {};
    groupBy.forEach((column) => {
      result[column] = group[0]?.[column] ?? null;
    });
    aggregations.forEach((aggregation) => {
      const alias = aggregation.alias || `${aggregation.function}_${aggregation.column || 'rows'}`;
      result[alias] = computeAggregation(group, aggregation);
    });
    return result;
  });
}

function computeAggregation(
  rows: Array<Record<string, JsonValue>>,
  aggregation: AggregateDefinitionDraft,
): JsonValue {
  switch (aggregation.function) {
    case 'count':
      return rows.length;
    case 'count_distinct':
      return new Set(rows.map((row) => `${row[aggregation.column] ?? ''}`)).size;
    case 'sum':
    case 'avg': {
      const numbers = rows.map((row) => Number(`${row[aggregation.column] ?? ''}`)).filter((value) => !Number.isNaN(value));
      const total = numbers.reduce((sum, value) => sum + value, 0);
      return aggregation.function === 'sum' ? total : numbers.length > 0 ? total / numbers.length : null;
    }
    case 'min':
      return [...rows]
        .map((row) => row[aggregation.column])
        .sort((left, right) => compareValues(left, `${right ?? ''}`))[0] ?? null;
    case 'max':
      return [...rows]
        .map((row) => row[aggregation.column])
        .sort((left, right) => compareValues(left, `${right ?? ''}`))
        .at(-1) ?? null;
  }
}

function cloneRows(rows: Array<Record<string, JsonValue>>): Array<Record<string, JsonValue>> {
  return rows.map((row) => ({ ...row }));
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

export function columnSamplePreview(column: DiscoveredColumn): string {
  return column.sample_values?.slice(0, 3).join(', ') ?? 'No sample values';
}

interface PreviewHelpers {
  TRIM: (value: JsonValue) => string | null;
  UPPER: (value: JsonValue) => string | null;
  LOWER: (value: JsonValue) => string | null;
  CONCAT: (...values: JsonValue[]) => string;
  COALESCE: (...values: JsonValue[]) => JsonValue;
}
