'use client';

import { useEffect, useState } from 'react';
import { CronExpressionParser } from 'cron-parser';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, FormProvider, useForm } from 'react-hook-form';
import { X } from 'lucide-react';
import { z } from 'zod';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { FormField } from '@/components/shared/forms/form-field';
import { CronSchedulePicker } from '@/app/(dashboard)/data/pipelines/_components/cron-schedule-picker';
import { type DataSource, dataSuiteApi } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';
import { formatDateTime } from '@/lib/format';

// Credential fields stripped by the backend sanitiser, keyed by connector type.
const CREDENTIAL_FIELDS: Record<string, { key: string; label: string }[]> = {
  postgresql:  [{ key: 'password',      label: 'Password' }],
  mysql:       [{ key: 'password',      label: 'Password' }],
  clickhouse:  [{ key: 'password',      label: 'Password' }],
  dolt:        [{ key: 'password',      label: 'Password' }],
  impala:      [{ key: 'password',      label: 'Password' }, { key: 'keytab', label: 'Keytab (base64)' }],
  hive:        [{ key: 'password',      label: 'Password' }, { key: 'keytab', label: 'Keytab (base64)' }],
  hdfs:        [{ key: 'keytab',        label: 'Keytab (base64)' }],
  spark:       [{ key: 'password',      label: 'Password' }],
  dagster:     [{ key: 'token',         label: 'API Token' }],
  api:         [{ key: 'api_key',       label: 'API Key' }, { key: 'token', label: 'Bearer Token' }, { key: 'client_secret', label: 'Client Secret' }],
  s3:          [{ key: 'access_key',    label: 'Access Key ID' }, { key: 'secret_key', label: 'Secret Access Key' }],
  stream:      [{ key: 'token',         label: 'Auth Token' }],
};

const PRESET_FREQUENCIES = [
  { label: 'Manual only',    value: 'manual' },
  { label: 'Every hour',     value: '0 * * * *' },
  { label: 'Every 6 hours',  value: '0 */6 * * *' },
  { label: 'Every 12 hours', value: '0 */12 * * *' },
  { label: 'Daily',          value: '0 0 * * *' },
  { label: 'Weekly',         value: '0 0 * * 0' },
  { label: 'Custom',         value: 'custom' },
] as const;

type FrequencyMode = (typeof PRESET_FREQUENCIES)[number]['value'];

function isCronValid(value: string): boolean {
  try { CronExpressionParser.parse(value); return true; } catch { return false; }
}

function cronNextRun(expr: string): string | null {
  try {
    const next = CronExpressionParser.parse(expr).next().toDate();
    return formatDateTime(next);
  } catch { return null; }
}

const schema = z.object({
  name: z.string().min(2, 'Name is required').max(255),
  description: z.string().optional(),
  // Empty string → clear the schedule; non-empty → validated cron expression.
  sync_frequency: z
    .string()
    .default('')
    .refine(
      (v) => v === '' || isCronValid(v),
      'Invalid cron expression (e.g. 0 * * * *)',
    ),
  tags: z.array(z.string().min(1)).max(20).default([]),
  connection_config_json: z
    .string()
    .min(2, 'Connection config is required')
    .refine((v) => { try { JSON.parse(v); return true; } catch { return false; } }, 'Must be valid JSON'),
});

type FormValues = z.infer<typeof schema>;

interface EditSourceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  source: DataSource | null;
  onUpdated?: () => void;
}

export function EditSourceDialog({ open, onOpenChange, source, onUpdated }: EditSourceDialogProps) {
  const [tagInput, setTagInput] = useState('');
  const [frequencyMode, setFrequencyMode] = useState<FrequencyMode>('manual');
  // Per-connector credential overrides: only sent when the user types a value.
  const [credentials, setCredentials] = useState<Record<string, string>>({});

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', description: '', sync_frequency: '', tags: [], connection_config_json: '{}' },
  });

  useEffect(() => {
    if (!source || !open) return;
    const freq = source.sync_frequency ?? '';
    const matchedPreset = PRESET_FREQUENCIES.find((p) => p.value !== 'manual' && p.value !== 'custom' && p.value === freq);
    setFrequencyMode(freq === '' ? 'manual' : matchedPreset ? matchedPreset.value : 'custom');
    form.reset({
      name: source.name,
      description: source.description,
      sync_frequency: freq,
      tags: source.tags ?? [],
      connection_config_json: JSON.stringify(source.connection_config ?? {}, null, 2),
    });
    setTagInput('');
    setCredentials({});
  }, [form, open, source]);

  const addTag = () => {
    const next = tagInput.trim();
    if (!next) return;
    const current = form.getValues('tags') ?? [];
    if (!current.includes(next)) {
      form.setValue('tags', [...current, next], { shouldDirty: true, shouldValidate: true });
    }
    setTagInput('');
  };

  const setFrequency = (mode: FrequencyMode) => {
    setFrequencyMode(mode);
    if (mode === 'manual') {
      form.setValue('sync_frequency', '', { shouldValidate: true, shouldDirty: true });
    } else if (mode !== 'custom') {
      form.setValue('sync_frequency', mode, { shouldValidate: true, shouldDirty: true });
    }
  };

  const onSubmit = async (values: FormValues) => {
    if (!source) return;
    try {
      // Merge any credential overrides the user typed into the config JSON.
      const baseConfig = JSON.parse(values.connection_config_json) as Record<string, unknown>;
      const credFields = CREDENTIAL_FIELDS[source.type] ?? [];
      for (const { key } of credFields) {
        const override = credentials[key]?.trim();
        if (override) {
          baseConfig[key] = override;
        }
        // If override is blank, the key is omitted — backend credential
        // preservation will reinstate the existing stored value.
      }
      await dataSuiteApi.updateSource(source.id, {
        name: values.name,
        description: values.description,
        // Always send sync_frequency: empty string explicitly clears schedule.
        sync_frequency: values.sync_frequency,
        tags: values.tags,
        connection_config: baseConfig,
      });
      showSuccess('Source updated successfully.');
      onUpdated?.();
      onOpenChange(false);
    } catch (error) {
      showApiError(error);
    }
  };

  const credentialFields = source ? (CREDENTIAL_FIELDS[source.type] ?? []) : [];
  const watchedFreq = form.watch('sync_frequency');
  const nextRun = watchedFreq ? cronNextRun(watchedFreq) : null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit source{source ? `: ${source.name}` : ''}</DialogTitle>
        </DialogHeader>

        <FormProvider {...form}>
          <form className="space-y-5" onSubmit={form.handleSubmit(onSubmit)}>
            {/* ── Name ── */}
            <FormField name="name" label="Name" required>
              <Input {...form.register('name')} />
            </FormField>

            {/* ── Description ── */}
            <FormField name="description" label="Description">
              <Textarea rows={3} {...form.register('description')} />
            </FormField>

            {/* ── Tags ── */}
            <FormField name="tags" label="Tags">
              <div className="space-y-2">
                <div className="flex gap-2">
                  <Input
                    value={tagInput}
                    onChange={(e) => setTagInput(e.target.value)}
                    placeholder="Add tag and press Enter"
                    onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addTag(); } }}
                  />
                  <Button type="button" variant="outline" onClick={addTag}>Add</Button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {(form.watch('tags') ?? []).map((tag) => (
                    <Badge key={tag} variant="outline" className="gap-1">
                      {tag}
                      <button
                        type="button"
                        onClick={() => form.setValue('tags', (form.getValues('tags') ?? []).filter((t) => t !== tag), { shouldDirty: true })}
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              </div>
            </FormField>

            {/* ── Sync frequency ── */}
            <div className="space-y-3">
              <FormField name="sync_frequency" label="Sync frequency">
                <Controller
                  name="sync_frequency"
                  control={form.control}
                  render={() => (
                    <Select value={frequencyMode} onValueChange={(v) => setFrequency(v as FrequencyMode)}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        {PRESET_FREQUENCIES.map((p) => (
                          <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
              </FormField>

              {frequencyMode === 'custom' && (
                <CronSchedulePicker
                  value={watchedFreq}
                  onChange={(v) => form.setValue('sync_frequency', v, { shouldValidate: true, shouldDirty: true })}
                  name="sync_frequency"
                  label="Custom cron schedule"
                />
              )}

              {nextRun && frequencyMode !== 'custom' && (
                <p className="text-xs text-muted-foreground">Next run: {nextRun}</p>
              )}
            </div>

            {/* ── Credential fields ── */}
            {credentialFields.length > 0 && (
              <div className="space-y-3 rounded-lg border p-4">
                <div>
                  <p className="text-sm font-medium">Credentials</p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Leave blank to preserve the existing stored credential. Only fill in a field to rotate it.
                  </p>
                </div>
                {credentialFields.map(({ key, label }) => (
                  <div key={key} className="space-y-1">
                    <label className="text-sm font-medium">{label}</label>
                    <Input
                      type="password"
                      autoComplete="new-password"
                      placeholder="Leave blank to keep existing"
                      value={credentials[key] ?? ''}
                      onChange={(e) => setCredentials((prev) => ({ ...prev, [key]: e.target.value }))}
                    />
                  </div>
                ))}
              </div>
            )}

            {/* ── Connection config (non-sensitive) ── */}
            <FormField
              name="connection_config_json"
              label="Connection config (non-sensitive fields)"
              description="Credential fields are managed above and are never included in this JSON."
              required
            >
              <Textarea rows={10} className="font-mono text-xs" {...form.register('connection_config_json')} />
            </FormField>

            {credentialFields.length > 0 && (
              <Alert>
                <AlertDescription className="text-xs">
                  Credential fields ({credentialFields.map((f) => f.key).join(', ')}) are stripped from the JSON
                  display for security. Blank credential inputs above are preserved server-side automatically.
                </AlertDescription>
              </Alert>
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={form.formState.isSubmitting}>
                {form.formState.isSubmitting ? 'Saving…' : 'Save changes'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
