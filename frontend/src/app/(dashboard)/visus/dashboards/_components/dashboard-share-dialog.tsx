'use client';

import { useEffect, useMemo } from 'react';
import { useForm, FormProvider } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { MultiSelect } from '@/components/shared/forms/multi-select';
import type { UserDirectoryEntry, VisusDashboard } from '@/types/suites';

const shareSchema = z.object({
  visibility: z.enum(['private', 'team', 'organization', 'public']),
  shared_with: z.array(z.string()).default([]),
});

type ShareFormValues = z.infer<typeof shareSchema>;

interface DashboardShareDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  dashboard: VisusDashboard | null;
  users: UserDirectoryEntry[];
  pending?: boolean;
  onSubmit: (payload: ShareFormValues) => Promise<void>;
}

export function DashboardShareDialog({
  open,
  onOpenChange,
  dashboard,
  users,
  pending = false,
  onSubmit,
}: DashboardShareDialogProps) {
  const form = useForm<ShareFormValues>({
    resolver: zodResolver(shareSchema),
    defaultValues: {
      visibility: 'private',
      shared_with: [],
    },
  });

  const userOptions = useMemo(
    () =>
      users.map((user) => ({
        value: user.id,
        label: `${user.first_name} ${user.last_name}`.trim() || user.email,
      })),
    [users],
  );

  useEffect(() => {
    form.reset({
      visibility: dashboard?.visibility ?? 'private',
      shared_with: dashboard?.shared_with ?? [],
    });
  }, [dashboard, form, open]);

  const handleSubmit = form.handleSubmit(async (values) => {
    await onSubmit(values);
    onOpenChange(false);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Share Dashboard</DialogTitle>
          <DialogDescription>
            Update access for {dashboard?.name ?? 'this dashboard'} using the dedicated share contract.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form className="space-y-6" onSubmit={handleSubmit}>
            <div className="space-y-1.5">
              <Label htmlFor="share-visibility">Visibility</Label>
              <Select
                value={form.watch('visibility')}
                onValueChange={(value) => form.setValue('visibility', value as ShareFormValues['visibility'], { shouldDirty: true })}
              >
                <SelectTrigger id="share-visibility">
                  <SelectValue placeholder="Select visibility" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="private">Private</SelectItem>
                  <SelectItem value="team">Team</SelectItem>
                  <SelectItem value="organization">Organization</SelectItem>
                  <SelectItem value="public">Public</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="share-users">Shared With</Label>
              <MultiSelect
                options={userOptions}
                selected={form.watch('shared_with')}
                onChange={(values) => form.setValue('shared_with', values, { shouldDirty: true })}
                placeholder="Grant dashboard access to users"
              />
            </div>

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? 'Saving...' : 'Save Access'}
              </Button>
            </div>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
