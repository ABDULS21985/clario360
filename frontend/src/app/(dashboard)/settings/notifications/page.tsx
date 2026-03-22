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

/* ------------------------------------------------------------------ */
/* Notification type labels — keys MUST match backend NotificationType */
/* ------------------------------------------------------------------ */
const NOTIFICATION_TYPE_LABELS: Record<string, string> = {
  // Security & Alerts
  'alert.created': 'Security Alerts',
  'alert.escalated': 'Alert Escalations',
  'security.incident': 'Security Incidents',
  'login.anomaly': 'Login Anomalies',
  'malware.detected': 'Malware Detection',
  // Remediation
  'remediation.approval_required': 'Remediation Approvals',
  'remediation.completed': 'Remediation Completed',
  'remediation.failed': 'Remediation Failures',
  // Tasks & Workflows
  'task.assigned': 'Task Assignments',
  'task.overdue': 'Task Overdue',
  'task.escalated': 'Task Escalations',
  'workflow.failed': 'Workflow Failures',
  'workflow.completed': 'Workflow Completions',
  // Data & Pipelines
  'pipeline.failed': 'Pipeline Failures',
  'pipeline.completed': 'Pipeline Completions',
  'data_quality.issue_detected': 'Data Quality Issues',
  'contradiction.detected': 'Contradiction Detected',
  // Contracts & Legal
  'contract.expiring': 'Contract Expirations',
  'contract.created': 'Contract Created',
  'analysis.ready': 'Analysis Ready',
  'clause.risk_flagged': 'Clause Risk Flagged',
  // Meetings & Governance
  'meeting.scheduled': 'Meeting Scheduled',
  'meeting.reminder': 'Meeting Reminders',
  'action_item.assigned': 'Action Item Assigned',
  'action_item.overdue': 'Action Item Overdue',
  'minutes.approved': 'Minutes Approved',
  // KPI & Reporting
  'kpi.threshold_breached': 'KPI Threshold Breached',
  // Account & System
  'password.expiring': 'Password Expiring',
  'system.maintenance': 'System Maintenance',
  'welcome': 'Welcome',
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

/* ------------------------------------------------------------------ */
/* Zod schemas — form-level model (frontend-friendly)                 */
/* ------------------------------------------------------------------ */

/** Channels use backend names: in_app, email, websocket, webhook */
const channelSchema = z.object({
  in_app: z.boolean(),
  email: z.boolean(),
  websocket: z.boolean(),
  webhook: z.boolean(),
});

const quietHoursSchema = z.object({
  enabled: z.boolean(),
  start_time: z.string(),
  end_time: z.string(),
  timezone: z.string(),
});

const digestSchema = z.object({
  daily: z.boolean(),
  weekly: z.boolean(),
});

const typeOverrideSchema = z.object({
  key: z.string(),
  in_app: z.boolean(),
  email: z.boolean(),
  websocket: z.boolean(),
  webhook: z.boolean(),
});

const preferencesSchema = z.object({
  channels: channelSchema,
  quiet_hours: quietHoursSchema,
  digest: digestSchema,
  overrides: z.array(typeOverrideSchema),
});

type PreferencesFormData = z.infer<typeof preferencesSchema>;

/* ------------------------------------------------------------------ */
/* Backend API shape — matches NotificationPreference model exactly    */
/* ------------------------------------------------------------------ */
interface ChannelPreference {
  in_app: boolean;
  email: boolean;
  websocket: boolean;
  webhook: boolean;
}

interface BackendQuietHours {
  enabled: boolean;
  start_time: string;
  end_time: string;
  timezone: string;
}

interface BackendDigestConfig {
  daily: boolean;
  weekly: boolean;
}

interface ApiPreferences {
  user_id: string;
  tenant_id: string;
  global_prefs: ChannelPreference;
  per_type_prefs: Record<string, ChannelPreference>;
  quiet_hours: BackendQuietHours | null;
  digest_config: BackendDigestConfig;
  updated_at: string;
}

interface ApiPreferenceUpdatePayload {
  global_prefs?: ChannelPreference;
  per_type_prefs?: Record<string, ChannelPreference>;
  quiet_hours?: BackendQuietHours;
  digest_config?: BackendDigestConfig;
}

/* ------------------------------------------------------------------ */
/* Mapping: API response → form data                                  */
/* ------------------------------------------------------------------ */
function toFormData(api: ApiPreferences): PreferencesFormData {
  return {
    channels: {
      in_app: api.global_prefs.in_app,
      email: api.global_prefs.email,
      websocket: api.global_prefs.websocket,
      webhook: api.global_prefs.webhook,
    },
    quiet_hours: {
      enabled: api.quiet_hours?.enabled ?? false,
      start_time: api.quiet_hours?.start_time ?? '22:00',
      end_time: api.quiet_hours?.end_time ?? '07:00',
      timezone: api.quiet_hours?.timezone ?? 'Asia/Riyadh',
    },
    digest: {
      daily: api.digest_config.daily,
      weekly: api.digest_config.weekly,
    },
    overrides: NOTIFICATION_TYPES.map((key) => {
      const override = api.per_type_prefs[key];
      return {
        key,
        in_app: override?.in_app ?? true,
        email: override?.email ?? true,
        websocket: override?.websocket ?? true,
        webhook: override?.webhook ?? false,
      };
    }),
  };
}

/* ------------------------------------------------------------------ */
/* Mapping: form data → API update payload                            */
/* ------------------------------------------------------------------ */
function toApiPayload(formData: PreferencesFormData): ApiPreferenceUpdatePayload {
  const perTypePrefs: Record<string, ChannelPreference> = {};
  for (const o of formData.overrides) {
    perTypePrefs[o.key] = {
      in_app: o.in_app,
      email: o.email,
      websocket: o.websocket,
      webhook: o.webhook,
    };
  }

  return {
    global_prefs: formData.channels,
    quiet_hours: {
      enabled: formData.quiet_hours.enabled,
      start_time: formData.quiet_hours.start_time,
      end_time: formData.quiet_hours.end_time,
      timezone: formData.quiet_hours.timezone,
    },
    digest_config: {
      daily: formData.digest.daily,
      weekly: formData.digest.weekly,
    },
    per_type_prefs: perTypePrefs,
  };
}

/* ------------------------------------------------------------------ */
/* Default form values                                                */
/* ------------------------------------------------------------------ */
const DEFAULT_FORM: PreferencesFormData = {
  channels: { in_app: true, email: true, websocket: true, webhook: false },
  quiet_hours: { enabled: false, start_time: '22:00', end_time: '07:00', timezone: 'Asia/Riyadh' },
  digest: { daily: false, weekly: true },
  overrides: NOTIFICATION_TYPES.map((key) => ({
    key,
    in_app: true,
    email: true,
    websocket: true,
    webhook: false,
  })),
};

/* ------------------------------------------------------------------ */
/* Page component                                                     */
/* ------------------------------------------------------------------ */
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

  // Reset to defaults AND immediately persist so the dialog confirms
  // the user's intent — clicking "Reset" should take effect right away,
  // not leave the form dirty waiting for a separate Save click.
  const handleReset = async () => {
    setResetOpen(false);
    saveMutation.mutate(DEFAULT_FORM);
    form.reset(DEFAULT_FORM);
  };

  const onSubmit = (payload: PreferencesFormData) => {
    saveMutation.mutate(payload);
  };

  if (isLoading) return <LoadingSkeleton variant="card" count={4} />;
  if (isError) return <ErrorState message="Failed to load preferences" onRetry={() => refetch()} />;

  const isDirty = form.formState.isDirty;
  const watchQuietHoursEnabled = form.watch('quiet_hours.enabled');

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="w-full space-y-6">
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
            { key: 'websocket' as const, label: 'Real-time notifications', description: 'Live updates via WebSocket connection', disabled: false },
            { key: 'webhook' as const, label: 'Webhook notifications', description: 'Deliver to registered webhook endpoints', disabled: false },
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
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <Label className="text-xs text-muted-foreground">Start time</Label>
                  <input
                    type="time"
                    {...form.register('quiet_hours.start_time')}
                    className="mt-1 block w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                  />
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground">End time</Label>
                  <input
                    type="time"
                    {...form.register('quiet_hours.end_time')}
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
            <div>
              <p className="text-sm font-medium">Daily digest</p>
              <p className="text-xs text-muted-foreground">Receive a daily summary each morning</p>
            </div>
            <Switch
              checked={form.watch('digest.daily')}
              onCheckedChange={(v) => form.setValue('digest.daily', v, { shouldDirty: true })}
              aria-label="Enable daily digest"
            />
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Weekly digest</p>
              <p className="text-xs text-muted-foreground">Receive a weekly summary each Monday</p>
            </div>
            <Switch
              checked={form.watch('digest.weekly')}
              onCheckedChange={(v) => form.setValue('digest.weekly', v, { shouldDirty: true })}
              aria-label="Enable weekly digest"
            />
          </div>
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
              return (
                <AccordionItem key={field.id} value={field.key}>
                  <AccordionTrigger className="py-3">
                    <span className="text-sm font-medium">{label}</span>
                  </AccordionTrigger>
                  <AccordionContent className="space-y-4 pb-4">
                    <div className="space-y-2">
                      <Label className="text-xs text-muted-foreground">Channel overrides</Label>
                      <div className="flex flex-wrap gap-4">
                        {([
                          { key: 'in_app' as const, label: 'In-app' },
                          { key: 'email' as const, label: 'Email' },
                          { key: 'websocket' as const, label: 'Real-time' },
                          { key: 'webhook' as const, label: 'Webhook' },
                        ]).map((ch) => (
                          <label key={ch.key} className="flex items-center gap-2 text-sm">
                            <Checkbox
                              checked={form.watch(`overrides.${index}.${ch.key}`)}
                              onCheckedChange={(v) => form.setValue(`overrides.${index}.${ch.key}`, Boolean(v), { shouldDirty: true })}
                              disabled={ch.key === 'in_app'}
                            />
                            {ch.label}
                          </label>
                        ))}
                      </div>
                    </div>
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
