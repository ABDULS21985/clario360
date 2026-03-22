'use client';

import { useEffect } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
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
import { FormField } from '@/components/shared/forms/form-field';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { meetingSchema, type MeetingFormValues } from '@/lib/enterprise';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaMeeting } from '@/types/suites';

interface EditMeetingDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  meeting: ActaMeeting;
  onSuccess?: () => Promise<void>;
}

/** Convert an ISO 8601 date string to the `datetime-local` input format. */
function toDatetimeLocal(iso: string | null | undefined): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function EditMeetingDialog({
  open,
  onOpenChange,
  meeting,
  onSuccess,
}: EditMeetingDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm<MeetingFormValues>({
    resolver: zodResolver(meetingSchema),
    defaultValues: meetingToFormValues(meeting),
  });

  useEffect(() => {
    if (open) {
      form.reset(meetingToFormValues(meeting));
    }
  }, [form, open, meeting]);

  const updateMutation = useMutation({
    mutationFn: (payload: MeetingFormValues) =>
      enterpriseApi.acta.updateMeeting(meeting.id, {
        ...payload,
        scheduled_at: new Date(payload.scheduled_at).toISOString(),
        scheduled_end_at: payload.scheduled_end_at
          ? new Date(payload.scheduled_end_at).toISOString()
          : null,
      }),
    onSuccess: async () => {
      showSuccess('Meeting updated.', 'Changes have been saved.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-meeting', meeting.id] }),
        queryClient.invalidateQueries({ queryKey: ['acta-meetings'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-calendar'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
      if (onSuccess) await onSuccess();
      onOpenChange(false);
    },
    onError: showApiError,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit Meeting</DialogTitle>
          <DialogDescription>
            Update meeting details. Committee assignment cannot be changed.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => updateMutation.mutate(values))}
          >
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
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

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
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

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
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

function meetingToFormValues(meeting: ActaMeeting): MeetingFormValues {
  return {
    committee_id: meeting.committee_id,
    title: meeting.title,
    description: meeting.description ?? '',
    scheduled_at: toDatetimeLocal(meeting.scheduled_at),
    scheduled_end_at: toDatetimeLocal(meeting.scheduled_end_at) || '',
    duration_minutes: meeting.duration_minutes,
    location: meeting.location ?? '',
    location_type: (meeting.location_type as MeetingFormValues['location_type']) ?? 'physical',
    virtual_link: meeting.virtual_link ?? '',
    virtual_platform: meeting.virtual_platform ?? '',
    tags: meeting.tags ?? [],
    metadata: meeting.metadata ?? {},
  };
}
