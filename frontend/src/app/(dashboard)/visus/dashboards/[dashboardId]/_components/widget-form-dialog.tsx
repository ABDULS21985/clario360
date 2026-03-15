'use client';

import { useEffect, useMemo, useState } from 'react';
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
import { Card, CardContent } from '@/components/ui/card';
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
import { visusWidgetSchema } from '@/lib/enterprise/schemas';
import { nextWidgetPosition } from '@/lib/enterprise/utils';
import type { VisusDashboard, VisusKPIDefinition, VisusWidget, VisusWidgetTypeDefinition } from '@/types/suites';
import { formatJsonInput, parseJsonInput, widgetSupportsKpi } from '../../../_components/form-utils';

type WidgetFormValues = z.infer<typeof visusWidgetSchema>;

interface WidgetFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  dashboard: VisusDashboard;
  widget?: VisusWidget | null;
  widgetTypes: VisusWidgetTypeDefinition[];
  kpis: VisusKPIDefinition[];
  pending?: boolean;
  onSubmit: (payload: WidgetFormValues) => Promise<void>;
}

export function WidgetFormDialog({
  open,
  onOpenChange,
  dashboard,
  widget,
  widgetTypes,
  kpis,
  pending = false,
  onSubmit,
}: WidgetFormDialogProps) {
  const [configInput, setConfigInput] = useState('{}');

  const form = useForm<WidgetFormValues>({
    resolver: zodResolver(visusWidgetSchema),
    defaultValues: {
      title: '',
      subtitle: '',
      type: 'text',
      config: {},
      position: { x: 0, y: 0, w: 4, h: 3 },
      refresh_interval_seconds: 60,
    },
  });

  const selectedType = form.watch('type');
  const typeSchema = useMemo(
    () => widgetTypes.find((item) => item.type === selectedType)?.schema ?? {},
    [selectedType, widgetTypes],
  );
  const selectedKpiId = useMemo(() => {
    try {
      const parsed = parseJsonInput(configInput);
      return typeof parsed.kpi_id === 'string' ? parsed.kpi_id : undefined;
    } catch {
      return undefined;
    }
  }, [configInput]);
  const textContent = useMemo(() => {
    try {
      const parsed = parseJsonInput(configInput);
      return typeof parsed.content === 'string' ? parsed.content : '';
    } catch {
      return '';
    }
  }, [configInput]);

  useEffect(() => {
    const defaultPosition = nextWidgetPosition(dashboard, 4, 3);
    const nextValues: WidgetFormValues = {
      title: widget?.title ?? '',
      subtitle: widget?.subtitle ?? '',
      type: widget?.type ?? 'text',
      config: widget?.config ?? {},
      position: widget?.position ?? defaultPosition,
      refresh_interval_seconds: widget?.refresh_interval_seconds ?? 60,
    };
    form.reset(nextValues);
    setConfigInput(formatJsonInput(nextValues.config));
  }, [dashboard, form, open, widget]);

  const mergeConfigPatch = (patch: Record<string, unknown>) => {
    try {
      const current = parseJsonInput(configInput);
      setConfigInput(formatJsonInput({ ...current, ...patch }));
    } catch {
      setConfigInput(formatJsonInput(patch));
    }
  };

  const handleSubmit = form.handleSubmit(async (values) => {
    try {
      const config = parseJsonInput(configInput);
      await onSubmit({
        ...values,
        config,
      });
      onOpenChange(false);
    } catch (error) {
      form.setError('config', {
        type: 'validate',
        message: error instanceof Error ? error.message : 'Invalid widget configuration JSON.',
      });
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{widget ? 'Edit Widget' : 'Add Widget'}</DialogTitle>
          <DialogDescription>
            Configure widget layout, data source hints, and refresh settings for this dashboard.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form className="space-y-6" onSubmit={handleSubmit}>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="title" label="Title" required>
                <Input id="title" {...form.register('title')} placeholder="Executive KPI" />
              </FormField>
              <FormField name="subtitle" label="Subtitle">
                <Input id="subtitle" {...form.register('subtitle')} placeholder="Optional widget context" />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="type" label="Widget Type" required description={widget ? 'Widget type is immutable after creation.' : undefined}>
                <Select
                  value={selectedType}
                  onValueChange={(value) =>
                    form.setValue('type', value as WidgetFormValues['type'], { shouldDirty: true, shouldValidate: true })
                  }
                  disabled={Boolean(widget)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select widget type" />
                  </SelectTrigger>
                  <SelectContent>
                    {widgetTypes.map((item) => (
                      <SelectItem key={item.type} value={item.type}>
                        {item.type.replace(/_/g, ' ')}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="refresh_interval_seconds" label="Refresh Interval (seconds)" required>
                <Input
                  id="refresh_interval_seconds"
                  type="number"
                  min={0}
                  {...form.register('refresh_interval_seconds', { valueAsNumber: true })}
                />
              </FormField>
            </div>

            <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
              <FormField name="position.x" label="X" required>
                <Input id="position.x" type="number" min={0} max={11} {...form.register('position.x', { valueAsNumber: true })} />
              </FormField>
              <FormField name="position.y" label="Y" required>
                <Input id="position.y" type="number" min={0} {...form.register('position.y', { valueAsNumber: true })} />
              </FormField>
              <FormField name="position.w" label="Width" required>
                <Input id="position.w" type="number" min={1} max={12} {...form.register('position.w', { valueAsNumber: true })} />
              </FormField>
              <FormField name="position.h" label="Height" required>
                <Input id="position.h" type="number" min={1} max={8} {...form.register('position.h', { valueAsNumber: true })} />
              </FormField>
            </div>

            {widgetSupportsKpi(selectedType) ? (
              <div className="space-y-1.5">
                <Label htmlFor="widget-kpi">Linked KPI</Label>
                <Select
                  value={selectedKpiId}
                  onValueChange={(value) => mergeConfigPatch({ kpi_id: value })}
                >
                  <SelectTrigger id="widget-kpi">
                    <SelectValue placeholder="Select KPI" />
                  </SelectTrigger>
                  <SelectContent>
                    {kpis.map((kpi) => (
                      <SelectItem key={kpi.id} value={kpi.id}>
                        {kpi.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}

            {selectedType === 'text' ? (
              <div className="space-y-1.5">
                <Label htmlFor="widget-content">Content</Label>
                <Textarea
                  id="widget-content"
                  rows={4}
                  value={textContent}
                  onChange={(event) => mergeConfigPatch({ content: event.target.value })}
                  placeholder="Write an executive note or operational annotation."
                />
              </div>
            ) : null}

            <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.1fr_0.9fr]">
              <Card>
                <CardContent className="space-y-3 pt-6">
                  <div className="space-y-1.5">
                    <Label htmlFor="widget-config">Config JSON</Label>
                    <Textarea
                      id="widget-config"
                      rows={12}
                      value={configInput}
                      onChange={(event) => setConfigInput(event.target.value)}
                      className="font-mono text-xs"
                      placeholder='{"kpi_id":"uuid"}'
                    />
                  </div>
                  {typeof form.formState.errors.config?.message === 'string' ? (
                    <p className="text-xs text-destructive" role="alert">
                      {form.formState.errors.config.message}
                    </p>
                  ) : null}
                </CardContent>
              </Card>

              <Card>
                <CardContent className="space-y-3 pt-6">
                  <div>
                    <p className="text-sm font-medium">Type Schema Hint</p>
                    <p className="text-xs text-muted-foreground">
                      Backend-declared schema for {selectedType.replace(/_/g, ' ')}.
                    </p>
                  </div>
                  <pre className="overflow-x-auto rounded-lg border bg-muted/40 p-3 text-xs">
                    {JSON.stringify(typeSchema, null, 2)}
                  </pre>
                </CardContent>
              </Card>
            </div>

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? 'Saving...' : widget ? 'Save Widget' : 'Create Widget'}
              </Button>
            </div>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
