'use client';

import { useEffect } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { Plus } from 'lucide-react';
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
import { FormField } from '@/components/shared/forms/form-field';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  actionItemSchema,
  dateInputValue,
  enterpriseApi,
  type ActionItemFormValues,
  userDisplayName,
} from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaCommittee, ActaMeeting } from '@/types/suites';

interface CreateActionItemDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  committees?: ActaCommittee[];
  meetings?: ActaMeeting[];
  preset?: Partial<ActionItemFormValues>;
  triggerLabel?: string;
}

export function CreateActionItemDialog({
  open,
  onOpenChange,
  committees = [],
  meetings = [],
  preset,
}: CreateActionItemDialogProps) {
  const queryClient = useQueryClient();
  const usersQuery = useQuery({
    queryKey: ['enterprise-users', 'action-items'],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, order: 'asc' }),
  });
  const form = useForm<ActionItemFormValues>({
    resolver: zodResolver(actionItemSchema),
    defaultValues: {
      meeting_id: preset?.meeting_id ?? '',
      agenda_item_id: preset?.agenda_item_id ?? null,
      committee_id: preset?.committee_id ?? '',
      title: preset?.title ?? '',
      description: preset?.description ?? '',
      priority: preset?.priority ?? 'medium',
      assigned_to: preset?.assigned_to ?? '',
      assignee_name: preset?.assignee_name ?? '',
      due_date: dateInputValue(preset?.due_date),
      tags: preset?.tags ?? [],
      metadata: preset?.metadata ?? {},
    },
  });

  useEffect(() => {
    if (open) {
      form.reset({
        meeting_id: preset?.meeting_id ?? '',
        agenda_item_id: preset?.agenda_item_id ?? null,
        committee_id: preset?.committee_id ?? '',
        title: preset?.title ?? '',
        description: preset?.description ?? '',
        priority: preset?.priority ?? 'medium',
        assigned_to: preset?.assigned_to ?? '',
        assignee_name: preset?.assignee_name ?? '',
        due_date: dateInputValue(preset?.due_date),
        tags: preset?.tags ?? [],
        metadata: preset?.metadata ?? {},
      });
    }
  }, [form, open, preset]);

  const createMutation = useMutation({
    mutationFn: (payload: ActionItemFormValues) => enterpriseApi.acta.createActionItem(payload),
    onSuccess: async () => {
      showSuccess('Action item created.', 'The follow-up has been added to the tracker.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-action-items'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-meeting-actions'] }),
      ]);
      onOpenChange(false);
    },
    onError: showApiError,
  });

  const users = usersQuery.data?.data ?? [];
  const isMeetingLocked = Boolean(preset?.meeting_id);
  const isCommitteeLocked = Boolean(preset?.committee_id);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Action Item</DialogTitle>
          <DialogDescription>
            Assign a follow-up from meeting decisions or governance reviews.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => createMutation.mutate(values))}
          >
            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="meeting_id" label="Meeting" required>
                <Select
                  value={form.watch('meeting_id')}
                  onValueChange={(value) => form.setValue('meeting_id', value, { shouldValidate: true })}
                  disabled={isMeetingLocked}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select meeting" />
                  </SelectTrigger>
                  <SelectContent>
                    {meetings.map((meeting) => (
                      <SelectItem key={meeting.id} value={meeting.id}>
                        {meeting.title}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="committee_id" label="Committee" required>
                <Select
                  value={form.watch('committee_id')}
                  onValueChange={(value) => form.setValue('committee_id', value, { shouldValidate: true })}
                  disabled={isCommitteeLocked}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select committee" />
                  </SelectTrigger>
                  <SelectContent>
                    {committees.map((committee) => (
                      <SelectItem key={committee.id} value={committee.id}>
                        {committee.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField name="title" label="Title" required>
              <Input {...form.register('title')} placeholder="Prepare Q2 board pack" />
            </FormField>

            <FormField name="description" label="Description" required>
              <Textarea {...form.register('description')} rows={4} />
            </FormField>

            <div className="grid gap-4 md:grid-cols-3">
              <FormField name="priority" label="Priority" required>
                <Select
                  value={form.watch('priority')}
                  onValueChange={(value) =>
                    form.setValue('priority', value as ActionItemFormValues['priority'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger>
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

              <FormField name="assigned_to" label="Assigned to" required>
                <Select
                  value={form.watch('assigned_to')}
                  onValueChange={(value) => {
                    const user = users.find((entry) => entry.id === value);
                    form.setValue('assigned_to', value, { shouldValidate: true });
                    form.setValue('assignee_name', user ? userDisplayName(user) : '', { shouldValidate: true });
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select assignee" />
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

              <FormField name="due_date" label="Due date" required>
                <Input type="date" {...form.register('due_date')} />
              </FormField>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                <Plus className="mr-1.5 h-4 w-4" />
                {createMutation.isPending ? 'Creating…' : 'Create action'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
