'use client';

import { useEffect, useMemo } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useForm, FormProvider } from 'react-hook-form';
import { Pencil } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { enterpriseApi, committeeSchema, type CommitteeFormValues, userDisplayName, dateInputValue } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import { FormField } from '@/components/shared/forms/form-field';
import type { ActaCommittee, UserDirectoryEntry } from '@/types/suites';

interface EditCommitteeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  committee: ActaCommittee;
}

const COMMITTEE_TYPES = [
  'board', 'audit', 'risk', 'compensation', 'nomination', 'executive', 'governance', 'ad_hoc',
] as const;

const MEETING_FREQUENCIES = [
  'weekly', 'bi_weekly', 'monthly', 'quarterly', 'semi_annual', 'annual', 'ad_hoc',
] as const;

const COMMITTEE_STATUSES = ['active', 'inactive', 'dissolved'] as const;

function committeeToFormValues(committee: ActaCommittee): CommitteeFormValues {
  return {
    name: committee.name,
    type: committee.type as CommitteeFormValues['type'],
    status: (committee.status as CommitteeFormValues['status']) ?? 'active',
    description: committee.description ?? '',
    chair_user_id: committee.chair_user_id ?? '',
    vice_chair_user_id: committee.vice_chair_user_id ?? null,
    secretary_user_id: committee.secretary_user_id ?? null,
    meeting_frequency: committee.meeting_frequency as CommitteeFormValues['meeting_frequency'],
    quorum_percentage: committee.quorum_percentage ?? 51,
    quorum_type: (committee.quorum_type as CommitteeFormValues['quorum_type']) ?? 'percentage',
    quorum_fixed_count: committee.quorum_fixed_count ?? null,
    charter: committee.charter ?? '',
    established_date: dateInputValue(committee.established_date),
    dissolution_date: dateInputValue(committee.dissolution_date),
    tags: committee.tags ?? [],
    metadata: committee.metadata ?? {},
    chair_name: committee.chair_name ?? '',
    chair_email: committee.chair_email ?? '',
    vice_chair_name: committee.vice_chair_name ?? '',
    vice_chair_email: committee.vice_chair_email ?? '',
    secretary_name: committee.secretary_name ?? '',
    secretary_email: committee.secretary_email ?? '',
  };
}

export function EditCommitteeDialog({
  open,
  onOpenChange,
  committee,
}: EditCommitteeDialogProps) {
  const queryClient = useQueryClient();
  const usersQuery = useQuery({
    queryKey: ['enterprise-users', 'edit-committee-dialog'],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, order: 'asc' }),
    enabled: open,
  });

  const form = useForm<CommitteeFormValues>({
    resolver: zodResolver(committeeSchema),
    defaultValues: committeeToFormValues(committee),
  });

  useEffect(() => {
    if (open) {
      form.reset(committeeToFormValues(committee));
    }
  }, [form, open, committee]);

  const users = usersQuery.data?.data ?? [];
  const usersById = useMemo(
    () => new Map(users.map((user) => [user.id, user])),
    [users],
  );

  const updateMutation = useMutation({
    mutationFn: (payload: CommitteeFormValues) =>
      enterpriseApi.acta.updateCommittee(committee.id, payload),
    onSuccess: () => {
      showSuccess('Committee updated.', 'Changes have been saved.');
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-committee', committee.id] }),
        queryClient.invalidateQueries({ queryKey: ['acta-committees'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
      onOpenChange(false);
    },
    onError: showApiError,
  });

  const applyUser = (
    prefix: 'chair' | 'vice_chair' | 'secretary',
    userId: string | null,
  ) => {
    const user = userId ? usersById.get(userId) : undefined;
    form.setValue(
      `${prefix}_user_id` as keyof CommitteeFormValues,
      userId as never,
      { shouldValidate: true },
    );
    form.setValue(`${prefix}_name` as keyof CommitteeFormValues, user ? userDisplayName(user) : '' as never, { shouldValidate: true });
    form.setValue(`${prefix}_email` as keyof CommitteeFormValues, user?.email ?? '' as never, { shouldValidate: true });
  };

  const quorumType = form.watch('quorum_type');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit Committee</DialogTitle>
          <DialogDescription>
            Update committee charter, cadence, and leadership.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-5"
            onSubmit={form.handleSubmit((values) => updateMutation.mutate(values))}
          >
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="name" label="Committee name" required>
                <Input {...form.register('name')} />
              </FormField>
              <FormField name="type" label="Committee type" required>
                <Select
                  value={form.watch('type')}
                  onValueChange={(value) =>
                    form.setValue('type', value as CommitteeFormValues['type'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {COMMITTEE_TYPES.map((type) => (
                      <SelectItem key={type} value={type}>
                        {type.replace(/_/g, ' ')}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="status" label="Status" required>
                <Select
                  value={form.watch('status') ?? 'active'}
                  onValueChange={(value) =>
                    form.setValue('status', value as CommitteeFormValues['status'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {COMMITTEE_STATUSES.map((status) => (
                      <SelectItem key={status} value={status} className="capitalize">
                        {status}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField name="description" label="Description" required>
              <Textarea {...form.register('description')} rows={3} />
            </FormField>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <RoleSelect
                label="Chair"
                name="chair_user_id"
                value={form.watch('chair_user_id')}
                users={users}
                onChange={(userId) => applyUser('chair', userId)}
                required
              />
              <RoleSelect
                label="Vice chair"
                name="vice_chair_user_id"
                value={form.watch('vice_chair_user_id') ?? ''}
                users={users}
                onChange={(userId) => applyUser('vice_chair', userId || null)}
              />
              <RoleSelect
                label="Secretary"
                name="secretary_user_id"
                value={form.watch('secretary_user_id') ?? ''}
                users={users}
                onChange={(userId) => applyUser('secretary', userId || null)}
              />
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="meeting_frequency" label="Meeting frequency" required>
                <Select
                  value={form.watch('meeting_frequency')}
                  onValueChange={(value) =>
                    form.setValue(
                      'meeting_frequency',
                      value as CommitteeFormValues['meeting_frequency'],
                      { shouldValidate: true },
                    )
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {MEETING_FREQUENCIES.map((freq) => (
                      <SelectItem key={freq} value={freq}>
                        {freq.replace(/_/g, ' ')}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <div className="rounded-lg border px-4 py-3">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">Use fixed quorum count</p>
                    <p className="text-xs text-muted-foreground">
                      Switch from percentage-based quorum to absolute count.
                    </p>
                  </div>
                  <Switch
                    checked={quorumType === 'fixed_count'}
                    onCheckedChange={(checked) =>
                      form.setValue('quorum_type', checked ? 'fixed_count' : 'percentage', {
                        shouldValidate: true,
                      })
                    }
                  />
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {quorumType === 'fixed_count' ? (
                <FormField name="quorum_fixed_count" label="Fixed quorum count" required>
                  <Input
                    type="number"
                    min={1}
                    value={form.watch('quorum_fixed_count') ?? ''}
                    onChange={(event) =>
                      form.setValue(
                        'quorum_fixed_count',
                        event.target.value ? Number(event.target.value) : null,
                        { shouldValidate: true },
                      )
                    }
                  />
                </FormField>
              ) : (
                <FormField name="quorum_percentage" label="Quorum percentage" required>
                  <Input
                    type="number"
                    min={1}
                    max={100}
                    value={form.watch('quorum_percentage')}
                    onChange={(event) =>
                      form.setValue('quorum_percentage', Number(event.target.value), {
                        shouldValidate: true,
                      })
                    }
                  />
                </FormField>
              )}
              <FormField name="established_date" label="Established date">
                <Input type="date" {...form.register('established_date')} />
              </FormField>
            </div>

            <FormField name="charter" label="Charter">
              <Textarea {...form.register('charter')} rows={6} />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending}>
                <Pencil className="mr-1.5 h-4 w-4" />
                {updateMutation.isPending ? 'Saving...' : 'Save changes'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

interface RoleSelectProps {
  label: string;
  name: string;
  value: string;
  users: UserDirectoryEntry[];
  onChange: (userId: string) => void;
  required?: boolean;
}

function RoleSelect({ label, name, value, users, onChange, required = false }: RoleSelectProps) {
  return (
    <FormField name={name} label={label} required={required}>
      <Select value={value || undefined} onValueChange={onChange}>
        <SelectTrigger>
          <SelectValue placeholder={`Select ${label.toLowerCase()}`} />
        </SelectTrigger>
        <SelectContent>
          {users.map((user) => (
            <SelectItem key={user.id} value={user.id}>
              {userDisplayName(user)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </FormField>
  );
}
