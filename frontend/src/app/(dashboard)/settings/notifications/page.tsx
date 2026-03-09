'use client';

import { useEffect, useState } from 'react';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { showSuccess, showApiError } from '@/lib/toast';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';

const NOTIFICATION_TYPE_LABELS: Record<string, string> = {
  'alert.created': 'Security Alerts',
  'task.assigned': 'Task Assignments',
  'pipeline.failed': 'Pipeline Events',
  'data_quality.issue': 'Data Quality Issues',
  'contract.expiring': 'Contract Expirations',
  'meeting.reminder': 'Meeting Reminders',
  'system.maintenance': 'System Maintenance',
};

const NOTIFICATION_TYPES = Object.keys(NOTIFICATION_TYPE_LABELS);

const TIMEZONES = [
  'UTC',
  'Asia/Riyadh',
  'Asia/Dubai',
  'Europe/London',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
];

const DAYS_OF_WEEK = [
  { value: 0, label: 'Sun' },
  { value: 1, label: 'Mon' },
  { value: 2, label: 'Tue' },
  { value: 3, label: 'Wed' },
  { value: 4, label: 'Thu' },
  { value: 5, label: 'Fri' },
  { value: 6, label: 'Sat' },
];

const channelSchema = z.object({
  in_app: z.boolean(),
  email: z.boolean(),
  push: z.boolean(),
  sms: z.boolean(),
});

const quietHoursSchema = z.object({
  enabled: z.boolean(),
  start: z.string(),
  end: z.string(),
  timezone: z.string(),
  days: z.array(z.number()),
});

const digestSchema = z.object({
  enabled: z.boolean(),
  frequency: z.enum(['daily', 'weekly', 'never']),
  delivery_time: z.string(),
});

const typeOverrideSchema = z.object({
  key: z.string(),
  enabled: z.boolean(),
  email: z.boolean(),
  push: z.boolean(),
  sms: z.boolean(),
  priority: z.enum(['low', 'normal', 'high', 'urgent']),
});

const preferencesSchema = z.object({
  channels: channelSchema,
  quiet_hours: quietHoursSchema,
  digest: digestSchema,
  overrides: z.array(typeOverrideSchema),
});

type PreferencesFormData = z.infer<typeof preferencesSchema>;

interface ApiPreferences {
  channels: {
    in_app: boolean;
    email: boolean;
    push: boolean;
    sms?: boolean;
  };
  quiet_hours: {
    enabled: boolean;
    start: string;
    end: string;
    timezone: string;
    days?: number[];
  };
  digest: {
    daily?: boolean;
    weekly?: boolean;
    enabled?: boolean;
    frequency?: 'daily' | 'weekly' | 'never';
    delivery_time?: string;
  };
  overrides: Record<string, { email: boolean; push: boolean; sms?: boolean; enabled?: boolean; priority?: string }>;
}

function toFormData(api: ApiPreferences): PreferencesFormData {
  let frequency: 'daily' | 'weekly' | 'never' = 'never';
  if (api.digest.frequency) {
    frequency = api.digest.frequency;
  } else if (api.digest.daily) {
    frequency = 'daily';
  } else if (api.digest.weekly) {
    frequency = 'weekly';
  }

  return {
    channels: {
      in_app: true,
      email: api.channels.email ?? true,
      push: api.channels.push ?? true,
      sms: api.channels.sms ?? false,
    },
    quiet_hours: {
      enabled: api.quiet_hours.enabled,
      start: api.quiet_hours.start ?? '22:00',
      end: api.quiet_hours.end ?? '07:00',
      timezone: api.quiet_hours.timezone ?? 'Asia/Riyadh',
      days: api.quiet_hours.days ?? [0, 1, 2, 3, 4, 5, 6],
    },
    digest: {
      enabled: api.digest.enabled ?? frequency !== 'never',
      frequency,
      delivery_time: api.digest.delivery_time ?? '09:00',
    },
    overrides: NOTIFICATION_TYPES.map((key) => {
      const override = api.overrides[key];
      return {
        key,
        enabled: override?.enabled ?? true,
        email: override?.email ?? true,
        push: override?.push ?? true,
        sms: override?.sms ?? false,
        priority: (override?.priority as 'low' | 'normal' | 'high' | 'urgent') ?? 'normal',
      };
    }),
  };
}

function toApiPayload(formData: PreferencesFormData): ApiPreferences {
  const overrides: ApiPreferences['overrides'] = {};
  for (const o of formData.overrides) {
    overrides[o.key] = {
      enabled: o.enabled,
      email: o.email,
      push: o.push,
      sms: o.sms,
      priority: o.priority,
    };
  }
  return {
    channels: formData.channels,
    quiet_hours: formData.quiet_hours,
    digest: {
      enabled: formData.digest.enabled,
      frequency: formData.digest.frequency,
      delivery_time: formData.digest.delivery_time,
      daily: formData.digest.frequency === 'daily',
      weekly: formData.digest.frequency === 'weekly',
    },
    overrides,
  };
}

const DEFAULT_FORM: PreferencesFormData = {
  channels: { in_app: true, email: true, push: true, sms: false },
  quiet_hours: { enabled: false, start: '22:00', end: '07:00', timezone: 'Asia/Riyadh', days: [0, 1, 2, 3, 4, 5, 6] },
  digest: { enabled: false, frequency: 'never', delivery_time: '09:00' },
  overrides: NOTIFICATION_TYPES.map((key) => ({
    key,
    enabled: true,
    email: true,
    push: true,
    sms: false,
    priority: 'normal' as const,
  })),
};

export default function NotificationPreferencesPage() {
  const [resetOpen, setResetOpen] = useState(false);
  const queryClient = useQueryClient();

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['notification-preferences'],
    queryFn: () => apiGet<ApiPreferences>(API_ENDPOINTS.NOTIFICATIONS_PREFERENCES),
  });

  const form = useForm<PreferencesFormData>({
    resolver: zodResolver(preferencesSchema),
    defaultValues: DEFAULT_FORM,
  });

  const { fields } = useFieldArray({
    control: form.control,
    name: 'overrides',
  });

  useEffect(() => {
    if (data) {
      form.reset(toFormData(data));
    }
  }, [data, form]);

  const saveMutation = useMutation({
    mutationFn: (payload: PreferencesFormData) =>
      apiPut(API_ENDPOINTS.NOTIFICATIONS_PREFERENCES, toApiPayload(payload)),
    onSuccess: () => {
      showSuccess('Preferences saved');
      queryClient.invalidateQueries({ queryKey: ['notification-preferences'] });
      form.reset(form.getValues());
    },
    onError: (error) => showApiError(error),
  });

  const handleReset = async () => {
    form.reset(DEFAULT_FORM);
    setResetOpen(false);
  };

  const onSubmit = (payload: PreferencesFormData) => {
    saveMutation.mutate(payload);
  };

  if (isLoading) return <LoadingSkeleton variant="card" count={4} />;
  if (isError) return <ErrorState message="Failed to load preferences" onRetry={() => refetch()} />;

  const isDirty = form.formState.isDirty;
  const watchQuietHoursEnabled = form.watch('quiet_hours.enabled');
  const watchDigestEnabled = form.watch('digest.enabled');

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 max-w-2xl">
      <PageHeader
        title="Notification Preferences"
        description="Customize how and when you receive notifications."
        actions={
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setResetOpen(true)}
          >
            Reset to Defaults
          </Button>
        }
      />

      {/* Section 1: Global Channels */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Notification Channels</CardTitle>
          <CardDescription>Choose how you receive notifications</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {[
            { key: 'in_app' as const, label: 'In-app notifications', description: 'Always enabled', disabled: true },
            { key: 'email' as const, label: 'Email notifications', description: 'Receive notifications via email', disabled: false },
            { key: 'push' as const, label: 'Push notifications', description: 'Real-time browser push', disabled: false },
            { key: 'sms' as const, label: 'SMS notifications', description: 'Text message alerts', disabled: false },
          ].map((ch) => (
            <div key={ch.key} className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">{ch.label}</p>
                <p className="text-xs text-muted-foreground">{ch.description}</p>
              </div>
              <Switch
                checked={form.watch(`channels.${ch.key}`)}
                onCheckedChange={(v) => form.setValue(`channels.${ch.key}`, v, { shouldDirty: true })}
                disabled={ch.disabled}
                aria-label={ch.label}
              />
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Section 2: Quiet Hours */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Quiet Hours</CardTitle>
          <CardDescription>
            During quiet hours, only critical notifications are delivered immediately.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Enable quiet hours</p>
            <Switch
              checked={watchQuietHoursEnabled}
              onCheckedChange={(v) => form.setValue('quiet_hours.enabled', v, { shouldDirty: true })}
              aria-label="Enable quiet hours"
            />
          </div>
          {watchQuietHoursEnabled && (
            <div className="space-y-4 pt-2">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-xs text-muted-foreground">Start time</Label>
                  <input
                    type="time"
                    {...form.register('quiet_hours.start')}
                    className="mt-1 block w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                  />
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground">End time</Label>
                  <input
                    type="time"
                    {...form.register('quiet_hours.end')}
                    className="mt-1 block w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                  />
                </div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Timezone</Label>
                <Select
                  value={form.watch('quiet_hours.timezone')}
                  onValueChange={(v) => form.setValue('quiet_hours.timezone', v, { shouldDirty: true })}
                >
                  <SelectTrigger className="mt-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TIMEZONES.map((tz) => (
                      <SelectItem key={tz} value={tz}>{tz}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground mb-2 block">Active days</Label>
                <div className="flex flex-wrap gap-2">
                  {DAYS_OF_WEEK.map((day) => {
                    const days = form.watch('quiet_hours.days');
                    const isChecked = days.includes(day.value);
                    return (
                      <button
                        key={day.value}
                        type="button"
                        onClick={() => {
                          const next = isChecked
                            ? days.filter((d) => d !== day.value)
                            : [...days, day.value].sort();
                          form.setValue('quiet_hours.days', next, { shouldDirty: true });
                        }}
                        className={cn(
                          'h-8 w-10 rounded-md border text-xs font-medium transition-colors',
                          isChecked
                            ? 'border-primary bg-primary text-primary-foreground'
                            : 'border-border bg-background hover:bg-muted',
                        )}
                      >
                        {day.label}
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Section 3: Digest */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Digest</CardTitle>
          <CardDescription>
            Receive a summary of notifications via email.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Enable digest</p>
            <Switch
              checked={watchDigestEnabled}
              onCheckedChange={(v) => form.setValue('digest.enabled', v, { shouldDirty: true })}
              aria-label="Enable digest"
            />
          </div>
          {watchDigestEnabled && (
            <div className="space-y-4 pt-2">
              <div>
                <Label className="text-xs text-muted-foreground">Frequency</Label>
                <div className="mt-2 flex gap-3">
                  {(['daily', 'weekly'] as const).map((freq) => (
                    <label key={freq} className="flex items-center gap-2 text-sm">
                      <input
                        type="radio"
                        value={freq}
                        checked={form.watch('digest.frequency') === freq}
                        onChange={() => form.setValue('digest.frequency', freq, { shouldDirty: true })}
                        className="h-4 w-4 text-primary"
                      />
                      {freq.charAt(0).toUpperCase() + freq.slice(1)}
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Delivery time</Label>
                <input
                  type="time"
                  {...form.register('digest.delivery_time')}
                  className="mt-1 block w-full max-w-[200px] rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Section 4: Per-Type Settings */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Per-Type Settings</CardTitle>
          <CardDescription>
            Override global channel settings for specific notification types.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Accordion type="multiple" className="w-full">
            {fields.map((field, index) => {
              const label = NOTIFICATION_TYPE_LABELS[field.key] ?? field.key;
              const isEnabled = form.watch(`overrides.${index}.enabled`);
              const priority = form.watch(`overrides.${index}.priority`);
              return (
                <AccordionItem key={field.id} value={field.key}>
                  <AccordionTrigger className="py-3">
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-medium">{label}</span>
                      {!isEnabled && (
                        <Badge variant="secondary" className="text-xs">Disabled</Badge>
                      )}
                      {priority === 'urgent' && (
                        <Badge variant="destructive" className="text-xs">Urgent</Badge>
                      )}
                      {priority === 'high' && (
                        <Badge variant="warning" className="text-xs">High</Badge>
                      )}
                    </div>
                  </AccordionTrigger>
                  <AccordionContent className="space-y-4 pb-4">
                    <div className="flex items-center justify-between">
                      <Label className="text-sm">Enabled</Label>
                      <Switch
                        checked={isEnabled}
                        onCheckedChange={(v) => form.setValue(`overrides.${index}.enabled`, v, { shouldDirty: true })}
                        aria-label={`Enable ${label}`}
                      />
                    </div>
                    {isEnabled && (
                      <>
                        <div className="space-y-2">
                          <Label className="text-xs text-muted-foreground">Channels (override global)</Label>
                          <div className="flex flex-wrap gap-4">
                            {(['email', 'push', 'sms'] as const).map((ch) => (
                              <label key={ch} className="flex items-center gap-2 text-sm">
                                <Checkbox
                                  checked={form.watch(`overrides.${index}.${ch}`)}
                                  onCheckedChange={(v) => form.setValue(`overrides.${index}.${ch}`, Boolean(v), { shouldDirty: true })}
                                />
                                {ch.charAt(0).toUpperCase() + ch.slice(1)}
                              </label>
                            ))}
                          </div>
                        </div>
                        <div>
                          <Label className="text-xs text-muted-foreground">Priority</Label>
                          <Select
                            value={priority}
                            onValueChange={(v) => form.setValue(`overrides.${index}.priority`, v as 'low' | 'normal' | 'high' | 'urgent', { shouldDirty: true })}
                          >
                            <SelectTrigger className="mt-1 max-w-[200px]">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="low">Low</SelectItem>
                              <SelectItem value="normal">Normal</SelectItem>
                              <SelectItem value="high">High</SelectItem>
                              <SelectItem value="urgent">Urgent</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </>
                    )}
                  </AccordionContent>
                </AccordionItem>
              );
            })}
          </Accordion>
        </CardContent>
      </Card>

      {/* Sticky save bar */}
      <div className="sticky bottom-0 flex items-center justify-end gap-3 border-t bg-background py-4">
        <Button
          type="submit"
          disabled={!isDirty || saveMutation.isPending}
        >
          {saveMutation.isPending ? 'Saving...' : 'Save Preferences'}
        </Button>
      </div>

      <ConfirmDialog
        open={resetOpen}
        onOpenChange={setResetOpen}
        title="Reset to Defaults"
        description="This will reset all notification preferences to their default values. Your current settings will be lost."
        confirmLabel="Reset"
        variant="destructive"
        onConfirm={handleReset}
      />
    </form>
  );
}
