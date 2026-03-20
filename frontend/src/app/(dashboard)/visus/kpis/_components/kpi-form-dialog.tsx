'use client';

import { useEffect, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { visusKpiSchema } from '@/lib/enterprise/schemas';
import type { VisusKPIDefinition } from '@/types/suites';
import { formatCommaSeparatedList, formatJsonInput, parseCommaSeparatedList, parseJsonInput } from '../../_components/form-utils';

type KpiFormValues = z.infer<typeof visusKpiSchema>;

const CATEGORIES: KpiFormValues['category'][] = ['security', 'data', 'governance', 'legal', 'operations', 'general'];
const SUITES: KpiFormValues['suite'][] = ['cyber', 'data', 'acta', 'lex', 'platform', 'custom'];
const UNITS: KpiFormValues['unit'][] = ['count', 'percentage', 'hours', 'minutes', 'score', 'currency', 'ratio', 'bytes'];
const DIRECTIONS: KpiFormValues['direction'][] = ['higher_is_better', 'lower_is_better'];
const CALCULATIONS: KpiFormValues['calculation_type'][] = ['direct', 'delta', 'percentage_change', 'average_over_period', 'sum_over_period'];
const FREQUENCIES: KpiFormValues['snapshot_frequency'][] = ['every_15m', 'hourly', 'every_4h', 'daily', 'weekly'];

function nullableNumber(value: unknown): number | null {
  if (value === '' || value == null) {
    return null;
  }
  const next = Number(value);
  return Number.isFinite(next) ? next : null;
}

function nullableString(value: string): string | null {
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

interface KpiFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kpi?: VisusKPIDefinition | null;
  pending?: boolean;
  onSubmit: (payload: KpiFormValues) => Promise<void>;
}

export function KpiFormDialog({ open, onOpenChange, kpi, pending = false, onSubmit }: KpiFormDialogProps) {
  const [tagsInput, setTagsInput] = useState('');
  const [queryParamsInput, setQueryParamsInput] = useState('{}');

  const form = useForm<KpiFormValues>({
    resolver: zodResolver(visusKpiSchema),
    defaultValues: {
      name: '',
      description: '',
      category: 'operations',
      suite: 'platform',
      icon: null,
      query_endpoint: '',
      query_params: {},
      value_path: '',
      unit: 'count',
      format_pattern: null,
      target_value: null,
      warning_threshold: null,
      critical_threshold: null,
      direction: 'higher_is_better',
      calculation_type: 'direct',
      calculation_window: null,
      snapshot_frequency: 'hourly',
      enabled: true,
      tags: [],
    },
  });

  useEffect(() => {
    const nextValues: KpiFormValues = {
      name: kpi?.name ?? '',
      description: kpi?.description ?? '',
      category: kpi?.category ?? 'operations',
      suite: kpi?.suite ?? 'platform',
      icon: kpi?.icon ?? null,
      query_endpoint: kpi?.query_endpoint ?? '',
      query_params: kpi?.query_params ?? {},
      value_path: kpi?.value_path ?? '',
      unit: kpi?.unit ?? 'count',
      format_pattern: kpi?.format_pattern ?? null,
      target_value: kpi?.target_value ?? null,
      warning_threshold: kpi?.warning_threshold ?? null,
      critical_threshold: kpi?.critical_threshold ?? null,
      direction: kpi?.direction ?? 'higher_is_better',
      calculation_type: kpi?.calculation_type ?? 'direct',
      calculation_window: kpi?.calculation_window ?? null,
      snapshot_frequency: kpi?.snapshot_frequency ?? 'hourly',
      enabled: kpi?.enabled ?? true,
      tags: kpi?.tags ?? [],
    };
    form.reset(nextValues);
    setTagsInput(formatCommaSeparatedList(nextValues.tags));
    setQueryParamsInput(formatJsonInput(nextValues.query_params));
  }, [form, kpi, open]);

  const handleSubmit = form.handleSubmit(async (values) => {
    try {
      const queryParams = parseJsonInput(queryParamsInput);
      await onSubmit({
        ...values,
        icon: nullableString(values.icon ?? ''),
        format_pattern: nullableString(values.format_pattern ?? ''),
        calculation_window: nullableString(values.calculation_window ?? ''),
        tags: parseCommaSeparatedList(tagsInput),
        query_params: queryParams,
      });
      onOpenChange(false);
    } catch (error) {
      form.setError('query_params', {
        type: 'validate',
        message: error instanceof Error ? error.message : 'Invalid query params JSON.',
      });
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{kpi ? 'Edit KPI' : 'Create KPI'}</DialogTitle>
          <DialogDescription>
            Register an executive KPI definition backed by a live endpoint and value path.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form className="space-y-6" onSubmit={handleSubmit}>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="name" label="Name" required>
                <Input id="name" {...form.register('name')} placeholder="Mean time to remediation" />
              </FormField>
              <FormField name="icon" label="Icon">
                <Input id="icon" {...form.register('icon')} placeholder="shield-check" />
              </FormField>
            </div>

            <FormField name="description" label="Description" required>
              <Textarea id="description" rows={3} {...form.register('description')} placeholder="Executive summary of what this KPI measures." />
            </FormField>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="category" label="Category" required>
                <Select value={form.watch('category')} onValueChange={(value) => form.setValue('category', value as KpiFormValues['category'], { shouldDirty: true })}>
                  <SelectTrigger>
                    <SelectValue placeholder="Category" />
                  </SelectTrigger>
                  <SelectContent>
                    {CATEGORIES.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="suite" label="Suite" required>
                <Select value={form.watch('suite')} onValueChange={(value) => form.setValue('suite', value as KpiFormValues['suite'], { shouldDirty: true })}>
                  <SelectTrigger>
                    <SelectValue placeholder="Suite" />
                  </SelectTrigger>
                  <SelectContent>
                    {SUITES.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="unit" label="Unit" required>
                <Select value={form.watch('unit')} onValueChange={(value) => form.setValue('unit', value as KpiFormValues['unit'], { shouldDirty: true })}>
                  <SelectTrigger>
                    <SelectValue placeholder="Unit" />
                  </SelectTrigger>
                  <SelectContent>
                    {UNITS.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="query_endpoint" label="Query Endpoint" required>
                <Input id="query_endpoint" {...form.register('query_endpoint')} placeholder="/api/v1/cyber/remediation/stats" />
              </FormField>
              <FormField name="value_path" label="Value Path" required>
                <Input id="value_path" {...form.register('value_path')} placeholder="summary.avg_mttr_hours" />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="direction" label="Direction" required>
                <Select value={form.watch('direction')} onValueChange={(value) => form.setValue('direction', value as KpiFormValues['direction'], { shouldDirty: true })}>
                  <SelectTrigger>
                    <SelectValue placeholder="Direction" />
                  </SelectTrigger>
                  <SelectContent>
                    {DIRECTIONS.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="calculation_type" label="Calculation" required>
                <Select value={form.watch('calculation_type')} onValueChange={(value) => form.setValue('calculation_type', value as KpiFormValues['calculation_type'], { shouldDirty: true })}>
                  <SelectTrigger>
                    <SelectValue placeholder="Calculation" />
                  </SelectTrigger>
                  <SelectContent>
                    {CALCULATIONS.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="snapshot_frequency" label="Snapshot Frequency" required>
                <Select value={form.watch('snapshot_frequency')} onValueChange={(value) => form.setValue('snapshot_frequency', value as KpiFormValues['snapshot_frequency'], { shouldDirty: true })}>
                  <SelectTrigger>
                    <SelectValue placeholder="Frequency" />
                  </SelectTrigger>
                  <SelectContent>
                    {FREQUENCIES.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
              <FormField name="target_value" label="Target Value">
                <Input id="target_value" type="number" step="any" {...form.register('target_value', { setValueAs: nullableNumber })} />
              </FormField>
              <FormField name="warning_threshold" label="Warning Threshold">
                <Input id="warning_threshold" type="number" step="any" {...form.register('warning_threshold', { setValueAs: nullableNumber })} />
              </FormField>
              <FormField name="critical_threshold" label="Critical Threshold">
                <Input id="critical_threshold" type="number" step="any" {...form.register('critical_threshold', { setValueAs: nullableNumber })} />
              </FormField>
              <FormField name="format_pattern" label="Format Pattern">
                <Input id="format_pattern" {...form.register('format_pattern')} placeholder="0.0%" />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="calculation_window" label="Calculation Window">
                <Input id="calculation_window" {...form.register('calculation_window')} placeholder="30d" />
              </FormField>
              <div className="space-y-1.5">
                <Label htmlFor="kpi-tags">Tags</Label>
                <Input id="kpi-tags" value={tagsInput} onChange={(event) => setTagsInput(event.target.value)} placeholder="executive, remediation, cyber" />
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Checkbox
                id="enabled"
                checked={form.watch('enabled')}
                onCheckedChange={(checked) => form.setValue('enabled', Boolean(checked), { shouldDirty: true })}
              />
              <Label htmlFor="enabled">Enabled</Label>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="query_params">Query Params JSON</Label>
              <Textarea
                id="query_params"
                rows={10}
                value={queryParamsInput}
                onChange={(event) => setQueryParamsInput(event.target.value)}
                className="font-mono text-xs"
                placeholder='{"status":"open","window":"30d"}'
              />
              {typeof form.formState.errors.query_params?.message === 'string' ? (
                <p className="text-xs text-destructive">{form.formState.errors.query_params.message}</p>
              ) : null}
            </div>

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? 'Saving...' : kpi ? 'Save KPI' : 'Create KPI'}
              </Button>
            </div>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
