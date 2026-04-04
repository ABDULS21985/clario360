'use client';

import { useState } from 'react';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Slider } from '@/components/ui/slider';
import { useCreateWebhook } from '@/hooks/use-webhooks';
import { Plus, Trash2, ChevronLeft, ChevronRight, Check } from 'lucide-react';
import { cn } from '@/lib/utils';

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

const ALL_EVENTS = Object.values(EVENT_GROUPS).flat();

const headerSchema = z.object({
  key: z.string().min(1, 'Header name required'),
  value: z.string().min(1, 'Header value required'),
});

const createWebhookSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100),
  url: z.string().url('Must be a valid URL'),
  headers: z.array(headerSchema),
  events: z.array(z.string()).min(1, 'Select at least one event'),
  max_retries: z.number().min(0).max(10),
  backoff_type: z.enum(['linear', 'exponential']),
  initial_delay_seconds: z.number().min(1).max(300),
});

type CreateWebhookFormData = z.infer<typeof createWebhookSchema>;

interface CreateWebhookDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: (name: string, secret: string) => void;
}

const STEPS = ['Basic Info', 'Events', 'Retry Policy', 'Review'];

export function CreateWebhookDialog({ open, onOpenChange, onSuccess }: CreateWebhookDialogProps) {
  const [step, setStep] = useState(0);
  const createMutation = useCreateWebhook();

  const form = useForm<CreateWebhookFormData>({
    resolver: zodResolver(createWebhookSchema),
    defaultValues: {
      name: '',
      url: '',
      headers: [],
      events: [],
      max_retries: 3,
      backoff_type: 'exponential',
      initial_delay_seconds: 10,
    },
  });

  const { fields: headerFields, append: appendHeader, remove: removeHeader } = useFieldArray({
    control: form.control,
    name: 'headers',
  });

  const resetAndClose = () => {
    form.reset();
    setStep(0);
    onOpenChange(false);
  };

  const handleSubmit = async () => {
    const data = form.getValues();
    const headers: Record<string, string> = {};
    for (const h of data.headers) {
      headers[h.key] = h.value;
    }

    try {
      const result = await createMutation.mutateAsync({
        name: data.name,
        url: data.url,
        events: data.events,
        headers: Object.keys(headers).length > 0 ? headers : undefined,
        retry_policy: {
          max_retries: data.max_retries,
          backoff_type: data.backoff_type,
          initial_delay_seconds: data.initial_delay_seconds,
        },
      });
      resetAndClose();
      onSuccess(result.webhook.name, result.secret);
    } catch {
      // Error handled by mutation
    }
  };

  const canProceed = () => {
    switch (step) {
      case 0: {
        const name = form.watch('name');
        const url = form.watch('url');
        return name.trim().length > 0 && url.trim().length > 0;
      }
      case 1:
        return form.watch('events').length > 0;
      case 2:
        return true;
      default:
        return true;
    }
  };

  const events = form.watch('events');
  const toggleEvent = (event: string) => {
    const current = form.getValues('events');
    if (current.includes(event)) {
      form.setValue('events', current.filter((e) => e !== event), { shouldValidate: true });
    } else {
      form.setValue('events', [...current, event], { shouldValidate: true });
    }
  };

  const toggleGroup = (groupEvents: string[]) => {
    const current = form.getValues('events');
    const allSelected = groupEvents.every((e) => current.includes(e));
    if (allSelected) {
      form.setValue('events', current.filter((e) => !groupEvents.includes(e)), { shouldValidate: true });
    } else {
      const merged = Array.from(new Set([...current, ...groupEvents]));
      form.setValue('events', merged, { shouldValidate: true });
    }
  };

  return (
    <Dialog open={open} onOpenChange={resetAndClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Create Webhook</DialogTitle>
          <DialogDescription>
            Step {step + 1} of {STEPS.length}: {STEPS[step]}
          </DialogDescription>
        </DialogHeader>

        {/* Step indicators */}
        <div className="flex items-center gap-2 py-2">
          {STEPS.map((label, i) => (
            <div key={label} className="flex items-center gap-2">
              <div
                className={cn(
                  'flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium',
                  i < step
                    ? 'bg-primary text-primary-foreground'
                    : i === step
                    ? 'border-2 border-primary text-primary'
                    : 'border border-border text-muted-foreground',
                )}
              >
                {i < step ? <Check className="h-3.5 w-3.5" /> : i + 1}
              </div>
              {i < STEPS.length - 1 && (
                <div className={cn('h-0.5 w-6', i < step ? 'bg-primary' : 'bg-border')} />
              )}
            </div>
          ))}
        </div>

        <div className="min-h-[280px]">
          {/* Step 1: Basic Info */}
          {step === 0 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="webhook-name">Name</Label>
                <Input
                  id="webhook-name"
                  placeholder="My webhook"
                  {...form.register('name')}
                />
                {form.formState.errors.name && (
                  <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="webhook-url">URL</Label>
                <Input
                  id="webhook-url"
                  placeholder="https://example.com/webhook"
                  {...form.register('url')}
                />
                {form.formState.errors.url && (
                  <p className="text-xs text-destructive">{form.formState.errors.url.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label>Custom Headers (optional)</Label>
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
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeHeader(index)}
                    >
                      <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Step 2: Events */}
          {step === 1 && (
            <div className="space-y-4 max-h-[350px] overflow-y-auto pr-2">
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  {events.length} of {ALL_EVENTS.length} events selected
                </p>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    if (events.length === ALL_EVENTS.length) {
                      form.setValue('events', [], { shouldValidate: true });
                    } else {
                      form.setValue('events', [...ALL_EVENTS], { shouldValidate: true });
                    }
                  }}
                >
                  {events.length === ALL_EVENTS.length ? 'Deselect All' : 'Select All'}
                </Button>
              </div>
              {Object.entries(EVENT_GROUPS).map(([group, groupEvents]) => {
                const allSelected = groupEvents.every((e) => events.includes(e));
                const someSelected = groupEvents.some((e) => events.includes(e));
                return (
                  <div key={group} className="space-y-2">
                    <div className="flex items-center gap-2">
                      <Checkbox
                        checked={allSelected}
                        ref={undefined}
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
            </div>
          )}

          {/* Step 3: Retry Policy */}
          {step === 2 && (
            <div className="space-y-6">
              <div className="space-y-3">
                <Label>Max Retries: {form.watch('max_retries')}</Label>
                <Slider
                  value={[form.watch('max_retries')]}
                  onValueChange={([v]) => form.setValue('max_retries', v)}
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
                        onChange={() => form.setValue('backoff_type', type)}
                        className="h-4 w-4"
                      />
                      {type.charAt(0).toUpperCase() + type.slice(1)}
                    </label>
                  ))}
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="initial-delay">Initial Delay (seconds)</Label>
                <Input
                  id="initial-delay"
                  type="number"
                  min={1}
                  max={300}
                  {...form.register('initial_delay_seconds', { valueAsNumber: true })}
                />
              </div>
            </div>
          )}

          {/* Step 4: Review */}
          {step === 3 && (
            <div className="space-y-4 text-sm">
              <div>
                <p className="font-medium text-muted-foreground">Name</p>
                <p>{form.watch('name')}</p>
              </div>
              <div>
                <p className="font-medium text-muted-foreground">URL</p>
                <p className="break-all font-mono text-xs">{form.watch('url')}</p>
              </div>
              {headerFields.length > 0 && (
                <div>
                  <p className="font-medium text-muted-foreground">Custom Headers</p>
                  <div className="space-y-1">
                    {form.watch('headers').map((h, i) => (
                      <p key={i} className="font-mono text-xs">{h.key}: {h.value}</p>
                    ))}
                  </div>
                </div>
              )}
              <div>
                <p className="font-medium text-muted-foreground">Events ({events.length})</p>
                <div className="flex flex-wrap gap-1 mt-1">
                  {events.slice(0, 6).map((e) => (
                    <Badge key={e} variant="outline" className="text-xs">{e}</Badge>
                  ))}
                  {events.length > 6 && (
                    <Badge variant="secondary" className="text-xs">+{events.length - 6} more</Badge>
                  )}
                </div>
              </div>
              <div>
                <p className="font-medium text-muted-foreground">Retry Policy</p>
                <p className="text-xs">
                  {form.watch('max_retries')} retries, {form.watch('backoff_type')} backoff,{' '}
                  {form.watch('initial_delay_seconds')}s initial delay
                </p>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="flex items-center justify-between gap-2 sm:justify-between">
          <Button
            type="button"
            variant="outline"
            onClick={() => (step > 0 ? setStep(step - 1) : resetAndClose())}
            disabled={createMutation.isPending}
          >
            <ChevronLeft className="mr-1 h-4 w-4" />
            {step > 0 ? 'Back' : 'Cancel'}
          </Button>
          {step < STEPS.length - 1 ? (
            <Button
              type="button"
              onClick={() => setStep(step + 1)}
              disabled={!canProceed()}
            >
              Next
              <ChevronRight className="ml-1 h-4 w-4" />
            </Button>
          ) : (
            <Button
              type="button"
              onClick={handleSubmit}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? 'Creating...' : 'Create Webhook'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
