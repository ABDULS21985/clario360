'use client';

import { useMemo } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useForm, FormProvider } from 'react-hook-form';
import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
import { enterpriseApi, committeeSchema, type CommitteeFormValues, userDisplayName } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import { FormField } from '@/components/shared/forms/form-field';
import type { UserDirectoryEntry } from '@/types/suites';

interface CreateCommitteeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const COMMITTEE_TYPES = [
  'board',
  'audit',
  'risk',
  'compensation',
  'nomination',
  'executive',
  'governance',
  'ad_hoc',
] as const;

const MEETING_FREQUENCIES = [
  'weekly',
  'bi_weekly',
  'monthly',
  'quarterly',
  'semi_annual',
  'annual',
  'ad_hoc',
] as const;

export function CreateCommitteeDialog({
  open,
  onOpenChange,
}: CreateCommitteeDialogProps) {
  const queryClient = useQueryClient();
  const usersQuery = useQuery({
    queryKey: ['enterprise-users', 'committee-dialog'],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, order: 'asc' }),
  });

  const form = useForm<CommitteeFormValues>({
    resolver: zodResolver(committeeSchema),
    defaultValues: {
      name: '',
      type: 'board',
      description: '',
      chair_user_id: '',
      vice_chair_user_id: null,
      secretary_user_id: null,
      meeting_frequency: 'monthly',
      quorum_percentage: 51,
      quorum_type: 'percentage',
      quorum_fixed_count: null,
      charter: '',
      established_date: '',
      dissolution_date: '',
      tags: [],
      metadata: {},
      chair_name: '',
      chair_email: '',
      vice_chair_name: '',
      vice_chair_email: '',
      secretary_name: '',
      secretary_email: '',
    },
  });

  const users = usersQuery.data?.data ?? [];
  const usersById = useMemo(
    () => new Map(users.map((user) => [user.id, user])),
    [users],
  );

  const createMutation = useMutation({
    mutationFn: (payload: CommitteeFormValues) => enterpriseApi.acta.createCommittee(payload),
    onSuccess: () => {
      showSuccess('Committee created.', 'The governance committee has been added.');
      void queryClient.invalidateQueries({ queryKey: ['acta-committees'] });
      onOpenChange(false);
      form.reset();
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
      <DialogTrigger asChild>
        <Button>
          <Plus className="mr-1.5 h-4 w-4" />
          Create Committee
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Committee</DialogTitle>
          <DialogDescription>
            Define committee charter, cadence, and leadership with live directory data.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-5"
            onSubmit={form.handleSubmit((values) => createMutation.mutate(values))}
          >
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="name" label="Committee name" required>
                <Input {...form.register('name')} placeholder="Board of Directors" />
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
            </div>

            <FormField name="description" label="Description" required>
              <Textarea
                {...form.register('description')}
                rows={3}
                placeholder="Mandate, remit, and oversight scope."
              />
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
                    {MEETING_FREQUENCIES.map((frequency) => (
                      <SelectItem key={frequency} value={frequency}>
                        {frequency.replace(/_/g, ' ')}
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
              <Textarea
                {...form.register('charter')}
                rows={6}
                placeholder="Paste the committee charter or mandate text."
              />
            </FormField>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? 'Creating…' : 'Create committee'}
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

function RoleSelect({
  label,
  name,
  value,
  users,
  onChange,
  required = false,
}: RoleSelectProps) {
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
