'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { DataModel, DataSource } from '@/lib/data-suite';
import {
  pipelineTargetSchema,
  type PipelineTargetValues,
} from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

interface WizardStepTargetProps {
  defaultValues: PipelineTargetValues;
  sources: DataSource[];
  models: DataModel[];
  availableColumns: string[];
  onBack: () => void;
  onContinue: (values: PipelineTargetValues) => void;
}

export function WizardStepTarget({
  defaultValues,
  sources,
  models,
  availableColumns,
  onBack,
  onContinue,
}: WizardStepTargetProps) {
  const form = useForm<PipelineTargetValues>({
    resolver: zodResolver(pipelineTargetSchema),
    mode: 'onChange',
    defaultValues,
  });

  const mergeKeys = form.watch('merge_keys');

  return (
    <FormProvider {...form}>
      <form className="space-y-6" onSubmit={form.handleSubmit(onContinue)}>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField name="target_id" label="Target source">
            <Select
              value={form.watch('target_id') ?? '__none__'}
              onValueChange={(next) =>
                form.setValue('target_id', next === '__none__' ? null : next, { shouldValidate: true })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="Optional target source" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">No target source</SelectItem>
                {sources.map((source) => (
                  <SelectItem key={source.id} value={source.id}>
                    {source.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          <FormField name="target_model_id" label="Target model">
            <Select
              value={form.watch('target_model_id') ?? '__none__'}
              onValueChange={(next) =>
                form.setValue('target_model_id', next === '__none__' ? null : next, { shouldValidate: true })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="Optional governed model" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">No model</SelectItem>
                {models.map((model) => (
                  <SelectItem key={model.id} value={model.id}>
                    {model.display_name || model.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField name="target_table" label="Target table">
            <Input {...form.register('target_table')} placeholder="analytics.customer_orders" />
          </FormField>

          <FormField name="load_strategy" label="Load strategy" required>
            <Select
              value={form.watch('load_strategy')}
              onValueChange={(next) =>
                form.setValue('load_strategy', next as PipelineTargetValues['load_strategy'], { shouldValidate: true })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="append">Append</SelectItem>
                <SelectItem value="full_replace">Full replace</SelectItem>
                <SelectItem value="incremental">Incremental</SelectItem>
                <SelectItem value="merge">Merge</SelectItem>
              </SelectContent>
            </Select>
          </FormField>
        </div>

        {form.watch('load_strategy') === 'merge' ? (
          <FormField name="merge_keys" label="Merge keys" required>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              {availableColumns.map((column) => (
                <label key={column} className="flex items-center gap-3 rounded-md border px-3 py-2 text-sm">
                  <Checkbox
                    checked={mergeKeys.includes(column)}
                    onCheckedChange={() =>
                      form.setValue(
                        'merge_keys',
                        mergeKeys.includes(column)
                          ? mergeKeys.filter((item) => item !== column)
                          : [...mergeKeys, column],
                        { shouldValidate: true },
                      )
                    }
                  />
                  <span>{column}</span>
                </label>
              ))}
            </div>
          </FormField>
        ) : null}

        <div className="flex justify-between">
          <Button type="button" variant="outline" onClick={onBack}>
            Back
          </Button>
          <Button type="submit" disabled={!form.formState.isValid}>
            Continue
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}

