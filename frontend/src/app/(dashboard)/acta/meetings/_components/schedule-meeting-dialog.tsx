'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { CalendarPlus } from 'lucide-react';
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
import { enterpriseApi, meetingSchema, type MeetingFormValues } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaCommittee } from '@/types/suites';

interface ScheduleMeetingDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  committees: ActaCommittee[];
}

export function ScheduleMeetingDialog({
  open,
  onOpenChange,
  committees,
}: ScheduleMeetingDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm<MeetingFormValues>({
    resolver: zodResolver(meetingSchema),
    defaultValues: {
      committee_id: '',
      title: '',
      description: '',
      scheduled_at: '',
      scheduled_end_at: '',
      duration_minutes: 60,
      location: '',
      location_type: 'physical',
      virtual_link: '',
      virtual_platform: '',
      tags: [],
      metadata: {},
    },
  });

  const createMutation = useMutation({
    mutationFn: (payload: MeetingFormValues) =>
      enterpriseApi.acta.createMeeting({
        ...payload,
        scheduled_at: new Date(payload.scheduled_at).toISOString(),
        scheduled_end_at: payload.scheduled_end_at
          ? new Date(payload.scheduled_end_at).toISOString()
          : null,
      }),
    onSuccess: async () => {
      showSuccess('Meeting scheduled.', 'Calendar and attendee records have been created.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-meetings'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-calendar'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
      onOpenChange(false);
      form.reset();
    },
    onError: showApiError,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Schedule Meeting</DialogTitle>
          <DialogDescription>
            Create a meeting and initialize attendance from the committee roster.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => createMutation.mutate(values))}
          >
            <FormField name="committee_id" label="Committee" required>
              <Select
                value={form.watch('committee_id')}
                onValueChange={(value) => form.setValue('committee_id', value, { shouldValidate: true })}
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

            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="title" label="Meeting title" required>
                <Input {...form.register('title')} placeholder="Q2 Board Meeting" />
              </FormField>
              <FormField name="location_type" label="Location type" required>
                <Select
                  value={form.watch('location_type')}
                  onValueChange={(value) =>
                    form.setValue('location_type', value as MeetingFormValues['location_type'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="physical">Physical</SelectItem>
                    <SelectItem value="virtual">Virtual</SelectItem>
                    <SelectItem value="hybrid">Hybrid</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField name="description" label="Description" required>
              <Textarea {...form.register('description')} rows={3} />
            </FormField>

            <div className="grid gap-4 md:grid-cols-3">
              <FormField name="scheduled_at" label="Start" required>
                <Input type="datetime-local" {...form.register('scheduled_at')} />
              </FormField>
              <FormField name="scheduled_end_at" label="End">
                <Input type="datetime-local" {...form.register('scheduled_end_at')} />
              </FormField>
              <FormField name="duration_minutes" label="Duration (minutes)" required>
                <Input
                  type="number"
                  min={15}
                  max={480}
                  value={form.watch('duration_minutes')}
                  onChange={(event) =>
                    form.setValue('duration_minutes', Number(event.target.value), { shouldValidate: true })
                  }
                />
              </FormField>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="location" label="Location">
                <Input {...form.register('location')} placeholder="Boardroom 4A or Teams Room" />
              </FormField>
              <FormField name="virtual_link" label="Virtual link">
                <Input {...form.register('virtual_link')} placeholder="https://meet.example.com/..." />
              </FormField>
            </div>

            <FormField name="virtual_platform" label="Virtual platform">
              <Input {...form.register('virtual_platform')} placeholder="Teams, Zoom, Webex" />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                <CalendarPlus className="mr-1.5 h-4 w-4" />
                {createMutation.isPending ? 'Scheduling…' : 'Schedule meeting'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
