'use client';

import { useEffect } from 'react';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { MultiSelect } from '@/components/shared/forms/multi-select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import {
  parseTagsInput,
  THREAT_FEED_AUTH_OPTIONS,
  THREAT_FEED_INTERVAL_OPTIONS,
  THREAT_FEED_TYPE_OPTIONS,
} from '@/lib/cyber-indicators';
import { API_ENDPOINTS } from '@/lib/constants';
import { INDICATOR_TYPE_OPTIONS } from '@/lib/cyber-threats';
import type { ThreatFeedConfig, ThreatFeedConfigInput } from '@/types/cyber';

const feedSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  type: z.enum(['stix', 'taxii', 'misp', 'csv_url', 'manual']),
  url: z.string().optional(),
  auth_type: z.enum(['none', 'api_key', 'basic', 'certificate']),
  api_key: z.string().optional(),
  username: z.string().optional(),
  password: z.string().optional(),
  certificate: z.string().optional(),
  private_key: z.string().optional(),
  sync_interval: z.enum(['hourly', 'every_6h', 'daily', 'weekly', 'manual']),
  default_severity: z.enum(['critical', 'high', 'medium', 'low']),
  default_confidence: z.number().min(0).max(100),
  default_tags_input: z.string().optional(),
  indicator_types: z.array(z.string()).default([]),
  enabled: z.boolean().default(true),
}).superRefine((value, ctx) => {
  if (value.type !== 'manual' && !value.url?.trim()) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['url'],
      message: 'URL is required for this feed type',
    });
  }

  if (value.url?.trim()) {
    try {
      new URL(value.url.trim());
    } catch {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['url'],
        message: 'Enter a valid URL',
      });
    }
  }

  if (value.auth_type === 'api_key' && !value.api_key?.trim()) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['api_key'],
      message: 'API key is required',
    });
  }

  if (value.auth_type === 'basic' && (!value.username?.trim() || !value.password?.trim())) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['username'],
      message: 'Username and password are required',
    });
  }
});

type ThreatFeedFormValues = z.infer<typeof feedSchema>;

interface AddFeedDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  feed?: ThreatFeedConfig | null;
  onSuccess?: (feed: ThreatFeedConfig) => void;
}

export function AddFeedDialog({
  open,
  onOpenChange,
  feed,
  onSuccess,
}: AddFeedDialogProps) {
  const isEditing = Boolean(feed);
  const methods = useForm<ThreatFeedFormValues>({
    resolver: zodResolver(feedSchema),
    defaultValues: {
      name: '',
      type: 'stix',
      url: '',
      auth_type: 'none',
      api_key: '',
      username: '',
      password: '',
      certificate: '',
      private_key: '',
      sync_interval: 'daily',
      default_severity: 'medium',
      default_confidence: 75,
      default_tags_input: '',
      indicator_types: [],
      enabled: true,
    },
  });

  useEffect(() => {
    if (!open) {
      return;
    }
    const authConfig = (feed?.auth_config ?? {}) as Record<string, unknown>;
    methods.reset({
      name: feed?.name ?? '',
      type: feed?.type ?? 'stix',
      url: feed?.url ?? '',
      auth_type: feed?.auth_type ?? 'none',
      api_key: typeof authConfig.api_key === 'string' ? authConfig.api_key : '',
      username: typeof authConfig.username === 'string' ? authConfig.username : '',
      password: typeof authConfig.password === 'string' ? authConfig.password : '',
      certificate: typeof authConfig.certificate === 'string' ? authConfig.certificate : '',
      private_key: typeof authConfig.private_key === 'string' ? authConfig.private_key : '',
      sync_interval: feed?.sync_interval ?? 'daily',
      default_severity: feed?.default_severity ?? 'medium',
      default_confidence: Math.round((feed?.default_confidence ?? 0.75) * 100),
      default_tags_input: (feed?.default_tags ?? []).join(', '),
      indicator_types: feed?.indicator_types ?? [],
      enabled: feed?.enabled ?? true,
    });
  }, [feed, methods, open]);

  const createMutation = useApiMutation<ThreatFeedConfig, ThreatFeedConfigInput>(
    'post',
    API_ENDPOINTS.CYBER_THREAT_FEEDS,
    {
      invalidateKeys: ['cyber-threat-feeds'],
      successMessage: 'Threat feed created',
      onSuccess: (response) => {
        onOpenChange(false);
        onSuccess?.(response);
      },
    },
  );

  const updateMutation = useApiMutation<ThreatFeedConfig, ThreatFeedConfigInput>(
    'put',
    () => `${API_ENDPOINTS.CYBER_THREAT_FEEDS}/${feed!.id}`,
    {
      invalidateKeys: ['cyber-threat-feeds'],
      successMessage: 'Threat feed updated',
      onSuccess: (response) => {
        onOpenChange(false);
        onSuccess?.(response);
      },
    },
  );

  const authType = methods.watch('auth_type');
  const type = methods.watch('type');
  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  const handleSubmit = methods.handleSubmit((values) => {
    const authConfig = buildAuthConfig(values);
    const payload: ThreatFeedConfigInput = {
      name: values.name.trim(),
      type: values.type,
      url: values.type === 'manual' ? undefined : values.url?.trim() || undefined,
      auth_type: values.auth_type,
      auth_config: authConfig,
      sync_interval: values.sync_interval,
      default_severity: values.default_severity,
      default_confidence: values.default_confidence / 100,
      default_tags: parseTagsInput(values.default_tags_input ?? ''),
      indicator_types: values.indicator_types,
      enabled: values.enabled,
    };

    if (isEditing) {
      updateMutation.mutate(payload);
      return;
    }
    createMutation.mutate(payload);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Threat Feed' : 'Add Threat Feed'}</DialogTitle>
          <DialogDescription>
            Configure the source, authentication, and import defaults the ingestion pipeline should apply.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="name" label="Name" required className="md:col-span-2">
                <Input id="name" placeholder="MISP community feed" {...methods.register('name')} />
              </FormField>

              <FormField name="type" label="Type" required>
                <Select
                  value={type}
                  onValueChange={(value) => methods.setValue('type', value as ThreatFeedFormValues['type'])}
                >
                  <SelectTrigger id="type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {THREAT_FEED_TYPE_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="sync_interval" label="Sync Interval" required>
                <Select
                  value={methods.watch('sync_interval')}
                  onValueChange={(value) => methods.setValue('sync_interval', value as ThreatFeedFormValues['sync_interval'])}
                >
                  <SelectTrigger id="sync_interval">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {THREAT_FEED_INTERVAL_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              {type !== 'manual' && (
                <FormField name="url" label="URL" required className="md:col-span-2">
                  <Input
                    id="url"
                    placeholder="https://intel.example.com/feed.json"
                    {...methods.register('url')}
                  />
                </FormField>
              )}

              <FormField name="auth_type" label="Authentication" required>
                <Select
                  value={authType}
                  onValueChange={(value) => methods.setValue('auth_type', value as ThreatFeedFormValues['auth_type'])}
                >
                  <SelectTrigger id="auth_type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {THREAT_FEED_AUTH_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="default_severity" label="Default Severity" required>
                <Select
                  value={methods.watch('default_severity')}
                  onValueChange={(value) => methods.setValue('default_severity', value as ThreatFeedFormValues['default_severity'])}
                >
                  <SelectTrigger id="default_severity">
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

              {authType === 'api_key' && (
                <FormField name="api_key" label="API Key" required className="md:col-span-2">
                  <Input id="api_key" type="password" {...methods.register('api_key')} />
                </FormField>
              )}

              {authType === 'basic' && (
                <>
                  <FormField name="username" label="Username" required>
                    <Input id="username" {...methods.register('username')} />
                  </FormField>
                  <FormField name="password" label="Password" required>
                    <Input id="password" type="password" {...methods.register('password')} />
                  </FormField>
                </>
              )}

              {authType === 'certificate' && (
                <>
                  <FormField name="certificate" label="Certificate">
                    <Input id="certificate" placeholder="PEM certificate" {...methods.register('certificate')} />
                  </FormField>
                  <FormField name="private_key" label="Private Key">
                    <Input id="private_key" placeholder="PEM private key" {...methods.register('private_key')} />
                  </FormField>
                </>
              )}

              <FormField
                name="default_confidence"
                label={`Default Confidence (${methods.watch('default_confidence')}%)`}
                className="md:col-span-2"
              >
                <Slider
                  value={[methods.watch('default_confidence')]}
                  min={0}
                  max={100}
                  step={1}
                  onValueChange={(value) => methods.setValue('default_confidence', value[0] ?? 0)}
                />
              </FormField>

              <FormField name="default_tags_input" label="Default Tags" className="md:col-span-2">
                <Input
                  id="default_tags_input"
                  placeholder="community, inbound, watchlist"
                  {...methods.register('default_tags_input')}
                />
              </FormField>

              <FormField
                name="indicator_types"
                label="Indicator Type Filter"
                description="Leave empty to ingest all IOC types the feed publishes."
                className="md:col-span-2"
              >
                <MultiSelect
                  options={INDICATOR_TYPE_OPTIONS.map((option) => ({
                    label: option.label,
                    value: option.value,
                  }))}
                  selected={methods.watch('indicator_types')}
                  onChange={(value) => methods.setValue('indicator_types', value)}
                  placeholder="All indicator types"
                />
              </FormField>

              <div className="md:col-span-2 flex items-center justify-between rounded-2xl border border-border/70 bg-slate-50/70 px-4 py-3">
                <div>
                  <Label htmlFor="enabled" className="text-sm font-medium">Enable feed</Label>
                  <p className="text-xs text-muted-foreground">
                    Disabled feeds remain configured but do not schedule syncs.
                  </p>
                </div>
                <Switch
                  id="enabled"
                  checked={methods.watch('enabled')}
                  onCheckedChange={(checked) => methods.setValue('enabled', checked)}
                />
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Saving…' : isEditing ? 'Save Feed' : 'Create Feed'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function buildAuthConfig(values: ThreatFeedFormValues): Record<string, unknown> | undefined {
  switch (values.auth_type) {
    case 'api_key':
      return values.api_key ? { api_key: values.api_key.trim() } : undefined;
    case 'basic':
      return values.username || values.password
        ? { username: values.username?.trim(), password: values.password?.trim() }
        : undefined;
    case 'certificate':
      return values.certificate || values.private_key
        ? { certificate: values.certificate?.trim(), private_key: values.private_key?.trim() }
        : undefined;
    default:
      return undefined;
  }
}
