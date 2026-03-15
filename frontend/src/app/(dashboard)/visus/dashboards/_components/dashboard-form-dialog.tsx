'use client';

import { useEffect, useMemo, useState } from 'react';
import { useForm, FormProvider } from 'react-hook-form';
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
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { MultiSelect } from '@/components/shared/forms/multi-select';
import { visusDashboardSchema } from '@/lib/enterprise/schemas';
import type { UserDirectoryEntry, VisusDashboard } from '@/types/suites';
import { formatCommaSeparatedList, formatJsonInput, parseCommaSeparatedList, parseJsonInput } from '../../_components/form-utils';

type DashboardFormValues = z.infer<typeof visusDashboardSchema>;

const VISIBILITY_OPTIONS: Array<{ label: string; value: DashboardFormValues['visibility'] }> = [
  { label: 'Private', value: 'private' },
  { label: 'Team', value: 'team' },
  { label: 'Organization', value: 'organization' },
  { label: 'Public', value: 'public' },
];

interface DashboardFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  dashboard?: VisusDashboard | null;
  users: UserDirectoryEntry[];
  pending?: boolean;
  onSubmit: (payload: DashboardFormValues) => Promise<void>;
}

export function DashboardFormDialog({
  open,
  onOpenChange,
  dashboard,
  users,
  pending = false,
  onSubmit,
}: DashboardFormDialogProps) {
  const [tagsInput, setTagsInput] = useState('');
  const [metadataInput, setMetadataInput] = useState('{}');
  const userOptions = useMemo(
    () =>
      users.map((user) => ({
        value: user.id,
        label: `${user.first_name} ${user.last_name}`.trim() || user.email,
      })),
    [users],
  );

  const form = useForm<DashboardFormValues>({
    resolver: zodResolver(visusDashboardSchema),
    defaultValues: {
      name: '',
      description: '',
      grid_columns: 12,
      visibility: 'private',
      shared_with: [],
      is_default: false,
      tags: [],
      metadata: {},
    },
  });

  useEffect(() => {
    const nextValues: DashboardFormValues = {
      name: dashboard?.name ?? '',
      description: dashboard?.description ?? '',
      grid_columns: dashboard?.grid_columns ?? 12,
      visibility: dashboard?.visibility ?? 'private',
      shared_with: dashboard?.shared_with ?? [],
      is_default: dashboard?.is_default ?? false,
      tags: dashboard?.tags ?? [],
      metadata: dashboard?.metadata ?? {},
    };
    form.reset(nextValues);
    setTagsInput(formatCommaSeparatedList(nextValues.tags));
    setMetadataInput(formatJsonInput(nextValues.metadata));
  }, [dashboard, form, open]);

  const handleSubmit = form.handleSubmit(async (values) => {
    try {
      const metadata = parseJsonInput(metadataInput);
      await onSubmit({
        ...values,
        tags: parseCommaSeparatedList(tagsInput),
        metadata,
      });
      onOpenChange(false);
    } catch (error) {
      form.setError('metadata', {
        type: 'validate',
        message: error instanceof Error ? error.message : 'Invalid JSON metadata.',
      });
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{dashboard ? 'Edit Dashboard' : 'Create Dashboard'}</DialogTitle>
          <DialogDescription>
            Configure dashboard visibility, default layout, and sharing using the Visus dashboard contract.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form className="space-y-6" onSubmit={handleSubmit}>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="name" label="Name" required>
                <Input id="name" {...form.register('name')} placeholder="Executive weekly command center" />
              </FormField>
              <FormField name="grid_columns" label="Grid Columns" required>
                <Input id="grid_columns" type="number" min={1} max={12} {...form.register('grid_columns', { valueAsNumber: true })} />
              </FormField>
            </div>

            <FormField name="description" label="Description" required>
              <Textarea
                id="description"
                rows={3}
                {...form.register('description')}
                placeholder="Summarize the operating view this dashboard supports."
              />
            </FormField>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="visibility" label="Visibility" required>
                <Select
                  value={form.watch('visibility')}
                  onValueChange={(value) =>
                    form.setValue('visibility', value as DashboardFormValues['visibility'], { shouldDirty: true, shouldValidate: true })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select visibility" />
                  </SelectTrigger>
                  <SelectContent>
                    {VISIBILITY_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <div className="space-y-1.5">
                <Label htmlFor="dashboard-tags">Tags</Label>
                <Input
                  id="dashboard-tags"
                  value={tagsInput}
                  onChange={(event) => setTagsInput(event.target.value)}
                  placeholder="board, weekly, executive"
                />
                <p className="text-xs text-muted-foreground">Comma-separated tags persisted as the dashboard tag array.</p>
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="dashboard-shared-with">Shared With</Label>
              <MultiSelect
                options={userOptions}
                selected={form.watch('shared_with')}
                onChange={(values) => form.setValue('shared_with', values, { shouldDirty: true })}
                placeholder="Select users with direct access"
              />
            </div>

            <div className="flex items-center gap-2">
              <Checkbox
                id="is_default"
                checked={form.watch('is_default')}
                onCheckedChange={(checked) => form.setValue('is_default', Boolean(checked), { shouldDirty: true })}
              />
              <Label htmlFor="is_default">Set as the tenant default dashboard</Label>
            </div>

            <Card>
              <CardContent className="space-y-3 pt-6">
                <div className="space-y-1.5">
                  <Label htmlFor="dashboard-metadata">Metadata JSON</Label>
                  <Textarea
                    id="dashboard-metadata"
                    rows={10}
                    value={metadataInput}
                    onChange={(event) => setMetadataInput(event.target.value)}
                    className="font-mono text-xs"
                    placeholder='{"owner_team":"executive","cadence":"weekly"}'
                  />
                </div>
                {typeof form.formState.errors.metadata?.message === 'string' ? (
                  <p className="text-xs text-destructive" role="alert">
                    {form.formState.errors.metadata.message}
                  </p>
                ) : null}
              </CardContent>
            </Card>

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? 'Saving...' : dashboard ? 'Save Changes' : 'Create Dashboard'}
              </Button>
            </div>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
