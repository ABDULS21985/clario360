'use client';

import { useMemo } from 'react';
import { CronExpressionParser } from 'cron-parser';
import { Input } from '@/components/ui/input';
import { FormField } from '@/components/shared/forms/form-field';
import { formatDateTime } from '@/lib/format';

interface CronSchedulePickerProps {
  value: string;
  onChange: (value: string) => void;
  name?: string;
  label?: string;
  description?: string;
}

export function CronSchedulePicker({
  value,
  onChange,
  name = 'cron_expression',
  label = 'Cron expression',
  description = 'Five-field cron expression in minute hour day month weekday format.',
}: CronSchedulePickerProps) {
  const preview = useMemo(() => {
    if (!value.trim()) {
      return { error: null, nextRuns: [] as string[] };
    }

    try {
      const iterator = CronExpressionParser.parse(value);
      const nextRuns = Array.from({ length: 5 }, () => formatDateTime(iterator.next().toDate()));
      return { error: null, nextRuns };
    } catch (error) {
      return {
        error: error instanceof Error ? error.message : 'Invalid cron expression',
        nextRuns: [] as string[],
      };
    }
  }, [value]);

  return (
    <div className="space-y-2">
      <FormField name={name} label={label} description={description}>
        <Input value={value} onChange={(event) => onChange(event.target.value)} placeholder="0 * * * *" />
      </FormField>
      {preview.error ? (
        <p className="text-xs text-destructive">{preview.error}</p>
      ) : preview.nextRuns.length > 0 ? (
        <div className="rounded-lg border bg-muted/20 p-3 text-xs text-muted-foreground">
          Next 5 runs: {preview.nextRuns.join(' • ')}
        </div>
      ) : null}
    </div>
  );
}
