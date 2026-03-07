'use client';

import { useEffect, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { X } from 'lucide-react';
import { FormField } from '@/components/shared/forms/form-field';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { CronSchedulePicker } from '@/app/(dashboard)/data/pipelines/_components/cron-schedule-picker';
import { sourceConfigureSchema, type SourceConfigureValues } from '@/lib/data-suite/forms';

interface WizardStepSyncProps {
  defaultValues: SourceConfigureValues;
  formId: string;
  onSubmit: (values: SourceConfigureValues) => void;
}

const PRESET_FREQUENCIES = [
  { label: 'Manual only', value: 'manual' },
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Every 12 hours', value: '0 */12 * * *' },
  { label: 'Daily', value: '0 0 * * *' },
  { label: 'Weekly', value: '0 0 * * 0' },
  { label: 'Custom', value: 'custom' },
] as const;

export function WizardStepSync({
  defaultValues,
  formId,
  onSubmit,
}: WizardStepSyncProps) {
  const form = useForm<SourceConfigureValues>({
    resolver: zodResolver(sourceConfigureSchema),
    defaultValues,
  });
  const [tagInput, setTagInput] = useState('');
  const [frequencyMode, setFrequencyMode] = useState<(typeof PRESET_FREQUENCIES)[number]['value']>(() => {
    if (!defaultValues.sync_frequency) {
      return 'manual';
    }
    const match = PRESET_FREQUENCIES.find((item) => item.value === defaultValues.sync_frequency);
    return match?.value ?? 'custom';
  });

  useEffect(() => {
    form.reset(defaultValues);
  }, [defaultValues, form]);

  return (
    <FormProvider {...form}>
      <form id={formId} className="space-y-4" onSubmit={form.handleSubmit((values) => onSubmit(values))}>
        <FormField name="name" label="Source name" required>
          <Input {...form.register('name')} />
        </FormField>

        <FormField name="description" label="Description">
          <Textarea rows={4} {...form.register('description')} />
        </FormField>

        <div className="space-y-3">
          <FormField name="tags" label="Tags">
            <div className="space-y-3">
              <div className="flex gap-2">
                <Input
                  value={tagInput}
                  onChange={(event) => setTagInput(event.target.value)}
                  placeholder="Add tag and press Enter"
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                      event.preventDefault();
                      const next = tagInput.trim();
                      if (!next) {
                        return;
                      }
                      form.setValue('tags', Array.from(new Set([...(form.getValues('tags') ?? []), next])), { shouldDirty: true, shouldValidate: true });
                      setTagInput('');
                    }
                  }}
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    const next = tagInput.trim();
                    if (!next) {
                      return;
                    }
                    form.setValue('tags', Array.from(new Set([...(form.getValues('tags') ?? []), next])), { shouldDirty: true, shouldValidate: true });
                    setTagInput('');
                  }}
                >
                  Add
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                {(form.watch('tags') ?? []).map((tag) => (
                  <Badge key={tag} variant="outline" className="gap-1">
                    {tag}
                    <button type="button" onClick={() => form.setValue('tags', (form.getValues('tags') ?? []).filter((value) => value !== tag), { shouldDirty: true })}>
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
          </FormField>
        </div>

        <FormField name="sync_frequency" label="Sync frequency">
          <Select
            value={frequencyMode}
            onValueChange={(value) => {
              const nextValue = value as (typeof PRESET_FREQUENCIES)[number]['value'];
              setFrequencyMode(nextValue);
              if (nextValue === 'manual') {
                form.setValue('sync_frequency', null, { shouldValidate: true, shouldDirty: true });
              } else if (nextValue !== 'custom') {
                form.setValue('sync_frequency', nextValue, { shouldValidate: true, shouldDirty: true });
              }
            }}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PRESET_FREQUENCIES.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FormField>

        {frequencyMode === 'custom' ? (
          <CronSchedulePicker
            value={form.watch('sync_frequency') ?? ''}
            onChange={(value) => form.setValue('sync_frequency', value, { shouldValidate: true, shouldDirty: true })}
            name="sync_frequency"
            label="Custom cron schedule"
          />
        ) : null}
      </form>
    </FormProvider>
  );
}
