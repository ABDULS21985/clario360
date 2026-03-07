'use client';

import { useEffect, useMemo, useState } from 'react';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { X } from 'lucide-react';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import type { DataModel, DataSource, QualityRule, QualityRuleType } from '@/lib/data-suite';

const qualityRuleSchema = z
  .object({
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
  })
  .superRefine((value, context) => {
    const config = value.config;
    const ruleType = value.rule_type;
    const requiresColumn = !['row_count', 'custom_sql'].includes(ruleType);

    if (requiresColumn && !value.column_name?.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['column_name'],
        message: 'Column is required',
      });
    }

    if (ruleType === 'range') {
      if (typeof config.min !== 'number') {
        context.addIssue({ code: z.ZodIssueCode.custom, path: ['config.min'], message: 'Minimum value is required' });
      }
      if (typeof config.max !== 'number') {
        context.addIssue({ code: z.ZodIssueCode.custom, path: ['config.max'], message: 'Maximum value is required' });
      }
    }

    if (ruleType === 'regex' && typeof config.pattern !== 'string') {
      context.addIssue({ code: z.ZodIssueCode.custom, path: ['config.pattern'], message: 'Regex pattern is required' });
    }

    if (ruleType === 'referential') {
      if (typeof config.reference_source_id !== 'string' || !config.reference_source_id) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['config.reference_source_id'],
          message: 'Reference source is required',
        });
      }
      if (typeof config.reference_table !== 'string' || !config.reference_table) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['config.reference_table'],
          message: 'Reference table is required',
        });
      }
      if (typeof config.reference_column !== 'string' || !config.reference_column) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['config.reference_column'],
          message: 'Reference column is required',
        });
      }
    }

    if (ruleType === 'enum') {
      const raw = typeof config.allowed_values_input === 'string' ? config.allowed_values_input : '';
      if (!raw.trim()) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['config.allowed_values_input'],
          message: 'Provide at least one allowed value',
        });
      }
    }

    if (ruleType === 'freshness' && typeof config.max_age_hours !== 'number') {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['config.max_age_hours'],
        message: 'Max age in hours is required',
      });
    }

    if (ruleType === 'row_count') {
      if (typeof config.min_count !== 'number' && typeof config.max_change_percent !== 'number') {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['config.min_count'],
          message: 'Set a minimum count or maximum change percent',
        });
      }
    }

    if (ruleType === 'custom_sql' && typeof config.sql !== 'string') {
      context.addIssue({ code: z.ZodIssueCode.custom, path: ['config.sql'], message: 'SQL is required' });
    }

    if (ruleType === 'statistical' && typeof config.z_score_threshold !== 'number') {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['config.z_score_threshold'],
        message: 'Z-score threshold is required',
      });
    }
  });

type QualityRuleFormValues = z.infer<typeof qualityRuleSchema>;

interface QualityRuleFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  models: DataModel[];
  sources: DataSource[];
  rule: QualityRule | null;
  submitting: boolean;
  onSubmit: (values: QualityRuleFormValues) => void;
}

export function QualityRuleForm({
  open,
  onOpenChange,
  models,
  sources,
  rule,
  submitting,
  onSubmit,
}: QualityRuleFormProps) {
  const [tagInput, setTagInput] = useState('');
  const form = useForm<QualityRuleFormValues>({
    resolver: zodResolver(qualityRuleSchema),
    mode: 'onChange',
    defaultValues: createDefaultValues(null),
  });

  useEffect(() => {
    if (!open) {
      return;
    }
    form.reset(createDefaultValues(rule));
    setTagInput('');
  }, [form, open, rule]);

  const selectedModel = useMemo(
    () => models.find((model) => model.id === form.watch('model_id')) ?? null,
    [form, models],
  );
  const columns = selectedModel?.schema_definition.map((field) => field.name) ?? [];
  const config = form.watch('config');
  const tags = form.watch('tags');
  const isEditing = Boolean(rule);

  const setConfigValue = (key: string, value: unknown) => {
    form.setValue('config', { ...form.getValues('config'), [key]: value }, { shouldValidate: true });
  };

  const ruleType = form.watch('rule_type');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? `Edit Rule: ${rule?.name}` : 'Create Quality Rule'}</DialogTitle>
        </DialogHeader>

        <FormProvider {...form}>
          <form className="space-y-6" onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="model_id" label="Model" required>
                <Select
                  value={form.watch('model_id')}
                  onValueChange={(next) => form.setValue('model_id', next, { shouldValidate: true })}
                  disabled={isEditing}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select model" />
                  </SelectTrigger>
                  <SelectContent>
                    {models.map((model) => (
                      <SelectItem key={model.id} value={model.id}>
                        {model.display_name || model.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="rule_type" label="Rule type" required>
                <Select
                  value={ruleType}
                  onValueChange={(next) => {
                    form.setValue('rule_type', next as QualityRuleType, { shouldValidate: true });
                    form.setValue('config', defaultConfigForRule(next as QualityRuleType), { shouldValidate: true });
                  }}
                  disabled={isEditing}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="not_null">Not Null</SelectItem>
                    <SelectItem value="unique">Unique</SelectItem>
                    <SelectItem value="range">Range</SelectItem>
                    <SelectItem value="regex">Regex</SelectItem>
                    <SelectItem value="referential">Referential</SelectItem>
                    <SelectItem value="enum">Enum</SelectItem>
                    <SelectItem value="freshness">Freshness</SelectItem>
                    <SelectItem value="row_count">Row Count</SelectItem>
                    <SelectItem value="custom_sql">Custom SQL</SelectItem>
                    <SelectItem value="statistical">Statistical</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="name" label="Rule name" required>
                <Input {...form.register('name')} placeholder="customer_email_present" />
              </FormField>

              <FormField name="severity" label="Severity" required>
                <Select
                  value={form.watch('severity')}
                  onValueChange={(next) =>
                    form.setValue('severity', next as QualityRuleFormValues['severity'], {
                      shouldValidate: true,
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="critical">Critical</SelectItem>
                    <SelectItem value="high">High</SelectItem>
                    <SelectItem value="medium">Medium</SelectItem>
                    <SelectItem value="low">Low</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField name="description" label="Description">
              <Textarea {...form.register('description')} rows={3} placeholder="What this rule validates and why it matters." />
            </FormField>

            {!['row_count', 'custom_sql'].includes(ruleType) ? (
              <FormField name="column_name" label="Column" required>
                <Select
                  value={form.watch('column_name') || ''}
                  onValueChange={(next) => form.setValue('column_name', next, { shouldValidate: true })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select column" />
                  </SelectTrigger>
                  <SelectContent>
                    {columns.map((column) => (
                      <SelectItem key={column} value={column}>
                        {column}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            ) : null}

            <div className="rounded-xl border bg-muted/10 p-4">
              <div className="mb-4 font-medium">Rule configuration</div>
              <div className="grid gap-4 md:grid-cols-2">
                {ruleType === 'range' ? (
                  <>
                    <FormField name="config.min" label="Minimum" required>
                      <Input
                        type="number"
                        value={numberValue(config.min)}
                        onChange={(event) => setConfigValue('min', parseNumberValue(event.target.value))}
                      />
                    </FormField>
                    <FormField name="config.max" label="Maximum" required>
                      <Input
                        type="number"
                        value={numberValue(config.max)}
                        onChange={(event) => setConfigValue('max', parseNumberValue(event.target.value))}
                      />
                    </FormField>
                  </>
                ) : null}

                {ruleType === 'regex' ? (
                  <FormField name="config.pattern" label="Regex pattern" required className="md:col-span-2">
                    <Input
                      value={stringValue(config.pattern)}
                      onChange={(event) => setConfigValue('pattern', event.target.value)}
                      placeholder="^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$"
                    />
                  </FormField>
                ) : null}

                {ruleType === 'referential' ? (
                  <>
                    <FormField name="config.reference_source_id" label="Reference source" required>
                      <Select
                        value={stringValue(config.reference_source_id)}
                        onValueChange={(next) => setConfigValue('reference_source_id', next)}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select source" />
                        </SelectTrigger>
                        <SelectContent>
                          {sources.map((source) => (
                            <SelectItem key={source.id} value={source.id}>
                              {source.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </FormField>
                    <FormField name="config.reference_table" label="Reference table" required>
                      <Input
                        value={stringValue(config.reference_table)}
                        onChange={(event) => setConfigValue('reference_table', event.target.value)}
                        placeholder="public.customers"
                      />
                    </FormField>
                    <FormField name="config.reference_column" label="Reference column" required className="md:col-span-2">
                      <Input
                        value={stringValue(config.reference_column)}
                        onChange={(event) => setConfigValue('reference_column', event.target.value)}
                        placeholder="customer_id"
                      />
                    </FormField>
                  </>
                ) : null}

                {ruleType === 'enum' ? (
                  <FormField name="config.allowed_values_input" label="Allowed values" required className="md:col-span-2">
                    <Input
                      value={stringValue(config.allowed_values_input)}
                      onChange={(event) => setConfigValue('allowed_values_input', event.target.value)}
                      placeholder="active, inactive, pending"
                    />
                  </FormField>
                ) : null}

                {ruleType === 'freshness' ? (
                  <FormField name="config.max_age_hours" label="Max age (hours)" required>
                    <Input
                      type="number"
                      value={numberValue(config.max_age_hours)}
                      onChange={(event) => setConfigValue('max_age_hours', parseNumberValue(event.target.value))}
                    />
                  </FormField>
                ) : null}

                {ruleType === 'row_count' ? (
                  <>
                    <FormField name="config.min_count" label="Minimum row count">
                      <Input
                        type="number"
                        value={numberValue(config.min_count)}
                        onChange={(event) => setConfigValue('min_count', parseNumberValue(event.target.value))}
                      />
                    </FormField>
                    <FormField name="config.max_change_percent" label="Max change percent">
                      <Input
                        type="number"
                        value={numberValue(config.max_change_percent)}
                        onChange={(event) =>
                          setConfigValue('max_change_percent', parseNumberValue(event.target.value))
                        }
                      />
                    </FormField>
                  </>
                ) : null}

                {ruleType === 'custom_sql' ? (
                  <FormField name="config.sql" label="SQL" required className="md:col-span-2">
                    <Textarea
                      rows={5}
                      value={stringValue(config.sql)}
                      onChange={(event) => setConfigValue('sql', event.target.value)}
                      placeholder="SELECT COUNT(*) FROM public.customers WHERE email IS NULL"
                    />
                  </FormField>
                ) : null}

                {ruleType === 'statistical' ? (
                  <FormField name="config.z_score_threshold" label="Z-score threshold" required>
                    <Input
                      type="number"
                      value={numberValue(config.z_score_threshold)}
                      onChange={(event) =>
                        setConfigValue('z_score_threshold', parseNumberValue(event.target.value))
                      }
                    />
                  </FormField>
                ) : null}
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="schedule" label="Schedule">
                <Input {...form.register('schedule')} placeholder="0 2 * * *" />
              </FormField>

              <FormField name="tags" label="Tags">
                <div className="space-y-3">
                  <div className="flex flex-wrap gap-2">
                    {tags.map((tag) => (
                      <span
                        key={tag}
                        className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs"
                      >
                        {tag}
                        <button
                          type="button"
                          onClick={() =>
                            form.setValue(
                              'tags',
                              tags.filter((item) => item !== tag),
                              { shouldValidate: true },
                            )
                          }
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </span>
                    ))}
                  </div>
                  <Input
                    value={tagInput}
                    onChange={(event) => setTagInput(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key !== 'Enter') {
                        return;
                      }
                      event.preventDefault();
                      const next = tagInput.trim();
                      if (!next || tags.includes(next)) {
                        return;
                      }
                      form.setValue('tags', [...tags, next], { shouldValidate: true });
                      setTagInput('');
                    }}
                    placeholder="critical, finance, nightly"
                  />
                </div>
              </FormField>
            </div>

            <div className="flex items-center gap-3 rounded-xl border p-4">
              <Switch
                checked={form.watch('enabled')}
                onCheckedChange={(checked) => form.setValue('enabled', checked, { shouldValidate: true })}
              />
              <div>
                <div className="font-medium">Enabled</div>
                <div className="text-sm text-muted-foreground">
                  Disabled rules remain in the catalog but are not executed.
                </div>
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={!form.formState.isValid || submitting}>
                {submitting ? 'Saving…' : isEditing ? 'Save changes' : 'Create rule'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function createDefaultValues(rule: QualityRule | null): QualityRuleFormValues {
  if (!rule) {
    return {
      model_id: '',
      name: '',
      description: '',
      rule_type: 'not_null',
      severity: 'medium',
      column_name: '',
      config: defaultConfigForRule('not_null'),
      schedule: '',
      enabled: true,
      tags: [],
    };
  }

  const baseConfig = { ...rule.config };
  if (rule.rule_type === 'enum') {
    baseConfig.allowed_values_input = Array.isArray(rule.config.allowed_values)
      ? (rule.config.allowed_values as string[]).join(', ')
      : '';
  }

  return {
    model_id: rule.model_id,
    name: rule.name,
    description: rule.description,
    rule_type: rule.rule_type,
    severity: rule.severity,
    column_name: rule.column_name ?? '',
    config: baseConfig,
    schedule: rule.schedule ?? '',
    enabled: rule.enabled,
    tags: rule.tags,
  };
}

function defaultConfigForRule(ruleType: QualityRuleType): Record<string, unknown> {
  switch (ruleType) {
    case 'range':
      return { min: 0, max: 100 };
    case 'regex':
      return { pattern: '' };
    case 'referential':
      return { reference_source_id: '', reference_table: '', reference_column: '' };
    case 'enum':
      return { allowed_values_input: '' };
    case 'freshness':
      return { max_age_hours: 24 };
    case 'row_count':
      return { min_count: 0, max_change_percent: 20 };
    case 'custom_sql':
      return { sql: '' };
    case 'statistical':
      return { z_score_threshold: 3 };
    default:
      return {};
  }
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function numberValue(value: unknown): string {
  return typeof value === 'number' ? `${value}` : '';
}

function parseNumberValue(value: string): number | undefined {
  if (!value.trim()) {
    return undefined;
  }
  const parsed = Number(value);
  return Number.isNaN(parsed) ? undefined : parsed;
}

export function buildQualityRulePayload(values: QualityRuleFormValues): Omit<QualityRuleFormValues, 'config'> & { config: Record<string, unknown> } {
  const config = { ...values.config };
  if (values.rule_type === 'enum') {
    config.allowed_values = stringValue(config.allowed_values_input)
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean);
    delete config.allowed_values_input;
  }
  if (values.rule_type === 'freshness' || values.rule_type === 'statistical') {
    config.column = values.column_name;
  }
  return {
    ...values,
    config,
  };
}

