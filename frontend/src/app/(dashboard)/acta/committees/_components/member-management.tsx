'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { SectionCard } from '@/components/suites/section-card';
import { FormField } from '@/components/shared/forms/form-field';
import {
  committeeMemberSchema,
  enterpriseApi,
  type CommitteeMemberFormValues,
  userDisplayName,
} from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaCommittee, UserDirectoryEntry } from '@/types/suites';

interface MemberManagementProps {
  committee: ActaCommittee;
}

const ROLES = ['chair', 'vice_chair', 'secretary', 'member', 'observer'] as const;

export function MemberManagement({ committee }: MemberManagementProps) {
  const queryClient = useQueryClient();
  const [selectedUser, setSelectedUser] = useState<string>('');
  const [selectedRole, setSelectedRole] = useState<CommitteeMemberFormValues['role']>('member');
  const usersQuery = useQuery({
    queryKey: ['enterprise-users', 'committee-members', committee.id],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, order: 'asc' }),
  });

  const form = useForm<CommitteeMemberFormValues>({
    resolver: zodResolver(committeeMemberSchema),
    defaultValues: {
      user_id: '',
      user_name: '',
      user_email: '',
      role: 'member',
    },
  });

  const users = usersQuery.data?.data ?? [];
  const usersById = useMemo(() => new Map(users.map((user) => [user.id, user])), [users]);

  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ['acta-committee', committee.id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-committees'] }),
    ]);

  const addMutation = useMutation({
    mutationFn: (payload: CommitteeMemberFormValues) =>
      enterpriseApi.acta.addCommitteeMember(committee.id, payload),
    onSuccess: async () => {
      showSuccess('Member added.', 'Committee roster has been updated.');
      await refresh();
      form.reset({ user_id: '', user_name: '', user_email: '', role: 'member' });
      setSelectedUser('');
      setSelectedRole('member');
    },
    onError: showApiError,
  });

  const updateMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: CommitteeMemberFormValues['role'] }) =>
      enterpriseApi.acta.updateCommitteeMember(committee.id, userId, { role }),
    onSuccess: async () => {
      showSuccess('Role updated.', 'Committee member role has been changed.');
      await refresh();
    },
    onError: showApiError,
  });

  const removeMutation = useMutation({
    mutationFn: (userId: string) => enterpriseApi.acta.removeCommitteeMember(committee.id, userId),
    onSuccess: async () => {
      showSuccess('Member removed.', 'The committee roster has been updated.');
      await refresh();
    },
    onError: showApiError,
  });

  const onUserSelected = (userId: string) => {
    const user = usersById.get(userId);
    setSelectedUser(userId);
    form.setValue('user_id', userId, { shouldValidate: true });
    form.setValue('user_name', user ? userDisplayName(user) : '', { shouldValidate: true });
    form.setValue('user_email', user?.email ?? '', { shouldValidate: true });
  };

  return (
    <SectionCard
      title="Member Management"
      description="Maintain committee membership and role assignments."
    >
      <div className="space-y-4">
        {(committee.members ?? []).filter((member) => member.active).map((member) => (
          <div
            key={member.id}
            className="flex flex-col gap-3 rounded-xl border px-4 py-3 md:flex-row md:items-center md:justify-between"
          >
            <div className="min-w-0">
              <p className="font-medium">{member.user_name}</p>
              <p className="text-xs text-muted-foreground">{member.user_email}</p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="outline">{member.role.replace(/_/g, ' ')}</Badge>
              <Select
                value={member.role}
                onValueChange={(role) =>
                  updateMutation.mutate({
                    userId: member.user_id,
                    role: role as CommitteeMemberFormValues['role'],
                  })
                }
              >
                <SelectTrigger className="w-[160px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ROLES.map((role) => (
                    <SelectItem key={role} value={role}>
                      {role.replace(/_/g, ' ')}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => removeMutation.mutate(member.user_id)}
                aria-label={`Remove ${member.user_name}`}
              >
                <Trash2 className="h-4 w-4 text-rose-600" />
              </Button>
            </div>
          </div>
        ))}

        <FormProvider {...form}>
          <form
            className="rounded-xl border border-dashed p-4"
            onSubmit={form.handleSubmit((values) => addMutation.mutate(values))}
          >
            <div className="grid gap-3 md:grid-cols-[1.4fr_0.8fr_auto]">
              <FormField name="user_id" label="Add member" required>
                <Select value={selectedUser || undefined} onValueChange={onUserSelected}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select user" />
                  </SelectTrigger>
                  <SelectContent>
                    {users.map((user: UserDirectoryEntry) => (
                      <SelectItem key={user.id} value={user.id}>
                        {userDisplayName(user)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="role" label="Role" required>
                <Select
                  value={selectedRole}
                  onValueChange={(role) => {
                    setSelectedRole(role as CommitteeMemberFormValues['role']);
                    form.setValue('role', role as CommitteeMemberFormValues['role'], {
                      shouldValidate: true,
                    });
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ROLES.map((role) => (
                      <SelectItem key={role} value={role}>
                        {role.replace(/_/g, ' ')}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <div className="flex items-end">
                <Button type="submit" className="w-full" disabled={addMutation.isPending}>
                  <Plus className="mr-1.5 h-4 w-4" />
                  Add
                </Button>
              </div>
            </div>
          </form>
        </FormProvider>
      </div>
    </SectionCard>
  );
}
