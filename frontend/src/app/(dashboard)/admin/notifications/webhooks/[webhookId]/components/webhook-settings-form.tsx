'use client';

import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Slider } from '@/components/ui/slider';
import { useUpdateWebhook } from '@/hooks/use-webhooks';
import { Plus, Trash2 } from 'lucide-react';
import type { NotificationWebhook } from '@/types/models';

const EVENT_GROUPS: Record<string, string[]> = {
  'Cyber Events': [
    'cyber.alert.created',
    'cyber.alert.resolved',
    'cyber.threat.detected',
    'cyber.vulnerability.found',
    'cyber.asset.compromised',
  ],
  'Data Events': [
    'data.pipeline.started',
    'data.pipeline.completed',
    'data.pipeline.failed',
    'data.quality.issue',
    'data.source.connected',
  ],
  'Acta Events': [
    'acta.meeting.scheduled',
    'acta.action.assigned',
    'acta.action.overdue',
    'acta.minutes.published',
  ],
  'Lex Events': [
    'lex.contract.expiring',
    'lex.compliance.violation',
    'lex.document.reviewed',
  ],
  'Workflow Events': [
    'workflow.task.assigned',
    'workflow.task.completed',
    'workflow.instance.failed',
    'workflow.approval.requested',
  ],
  'System Events': [
    'system.maintenance',
    'system.announcement',
    'system.user.created',
    'system.user.suspended',
  ],
};

const headerSchema = z.object({
  key: z.string().min(1, 'Required'),
  value: z.string().min(1, 'Required'),
});

const settingsSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100),
  url: z.string().url('Must be a valid URL'),
  headers: z.array(headerSchema),
  events: z.array(z.string()).min(1, 'Select at least one event'),
  max_retries: z.number().min(0).max(10),
  backoff_type: z.enum(['linear', 'exponential']),
  initial_delay_seconds: z.number().min(1).max(300),
});

type SettingsFormData = z.infer<typeof settingsSchema>;

interface WebhookSettingsFormProps {
  webhook: NotificationWebhook;
  onSaved: () => void;
}

export function WebhookSettingsForm({ webhook, onSaved }: WebhookSettingsFormProps) {
  const updateMutation = useUpdateWebhook(webhook.id);

  const form = useForm<SettingsFormData>({
    resolver: zodResolver(settingsSchema),
    defaultValues: {
      name: webhook.name,
      url: webhook.url,
      headers: Object.entries(webhook.headers).map(([key, value]) => ({ key, value })),
      events: webhook.events,
      max_retries: webhook.retry_policy.max_retries,
      backoff_type: webhook.retry_policy.backoff_type,
      initial_delay_seconds: webhook.retry_policy.initial_delay_seconds,
    },
  });

  const { fields: headerFields, append: appendHeader, remove: removeHeader } = useFieldArray({
    control: form.control,
    name: 'headers',
  });

  const onSubmit = async (data: SettingsFormData) => {
    const headers: Record<string, string> = {};
    for (const h of data.headers) {
      headers[h.key] = h.value;
    }

    await updateMutation.mutateAsync({
      name: data.name,
      url: data.url,
      events: data.events,
      headers,
      retry_policy: {
        max_retries: data.max_retries,
        backoff_type: data.backoff_type,
        initial_delay_seconds: data.initial_delay_seconds,
      },
    });
    form.reset(data);
    onSaved();
  };

  const events = form.watch('events');
  const toggleEvent = (event: string) => {
    const current = form.getValues('events');
    if (current.includes(event)) {
      form.setValue('events', current.filter((e) => e !== event), { shouldDirty: true, shouldValidate: true });
    } else {
      form.setValue('events', [...current, event], { shouldDirty: true, shouldValidate: true });
    }
  };

  const toggleGroup = (groupEvents: string[]) => {
    const current = form.getValues('events');
    const allSelected = groupEvents.every((e) => current.includes(e));
    if (allSelected) {
      form.setValue('events', current.filter((e) => !groupEvents.includes(e)), { shouldDirty: true, shouldValidate: true });
    } else {
      const merged = Array.from(new Set([...current, ...groupEvents]));
      form.setValue('events', merged, { shouldDirty: true, shouldValidate: true });
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 max-w-2xl">
      {/* Basic Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Basic Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-name">Name</Label>
            <Input id="edit-name" {...form.register('name')} />
            {form.formState.errors.name && (
              <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-url">URL</Label>
            <Input id="edit-url" {...form.register('url')} />
            {form.formState.errors.url && (
              <p className="text-xs text-destructive">{form.formState.errors.url.message}</p>
            )}
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Custom Headers</Label>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => appendHeader({ key: '', value: '' })}
              >
                <Plus className="mr-1 h-3 w-3" /> Add
              </Button>
            </div>
            {headerFields.map((field, index) => (
              <div key={field.id} className="flex items-center gap-2">
                <Input
                  placeholder="Header name"
                  {...form.register(`headers.${index}.key`)}
                  className="flex-1"
                />
                <Input
                  placeholder="Value"
                  {...form.register(`headers.${index}.value`)}
                  className="flex-1"
                />
                <Button type="button" variant="ghost" size="sm" onClick={() => removeHeader(index)}>
                  <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Events */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Events</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 max-h-[400px] overflow-y-auto">
          {Object.entries(EVENT_GROUPS).map(([group, groupEvents]) => {
            const allSelected = groupEvents.every((e) => events.includes(e));
            const someSelected = groupEvents.some((e) => events.includes(e));
            return (
              <div key={group} className="space-y-2">
                <div className="flex items-center gap-2">
                  <Checkbox
                    checked={allSelected}
                    onCheckedChange={() => toggleGroup(groupEvents)}
                    aria-label={`Select all ${group}`}
                  />
                  <span className="text-sm font-medium">{group}</span>
                  {someSelected && !allSelected && (
                    <Badge variant="secondary" className="text-xs">Partial</Badge>
                  )}
                </div>
                <div className="ml-6 grid grid-cols-1 gap-1.5">
                  {groupEvents.map((event) => (
                    <label key={event} className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Checkbox
                        checked={events.includes(event)}
                        onCheckedChange={() => toggleEvent(event)}
                      />
                      {event}
                    </label>
                  ))}
                </div>
              </div>
            );
          })}
          {form.formState.errors.events && (
            <p className="text-xs text-destructive">{form.formState.errors.events.message}</p>
          )}
        </CardContent>
      </Card>

      {/* Retry Policy */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Retry Policy</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-3">
            <Label>Max Retries: {form.watch('max_retries')}</Label>
            <Slider
              value={[form.watch('max_retries')]}
              onValueChange={([v]) => form.setValue('max_retries', v, { shouldDirty: true })}
              min={0}
              max={10}
              step={1}
            />
          </div>
          <div className="space-y-2">
            <Label>Backoff Type</Label>
            <div className="flex gap-3">
              {(['linear', 'exponential'] as const).map((type) => (
                <label key={type} className="flex items-center gap-2 text-sm">
                  <input
                    type="radio"
                    checked={form.watch('backoff_type') === type}
                    onChange={() => form.setValue('backoff_type', type, { shouldDirty: true })}
                    className="h-4 w-4"
                  />
                  {type.charAt(0).toUpperCase() + type.slice(1)}
                </label>
              ))}
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-delay">Initial Delay (seconds)</Label>
            <Input
              id="edit-delay"
              type="number"
              min={1}
              max={300}
              {...form.register('initial_delay_seconds', { valueAsNumber: true })}
            />
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button
          type="submit"
          disabled={!form.formState.isDirty || updateMutation.isPending}
        >
          {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
        </Button>
      </div>
    </form>
  );
}
