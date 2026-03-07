'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { useState } from 'react';
import { AlertTriangle, CalendarClock, Play, Square } from 'lucide-react';
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
  cancelMeetingSchema,
  postponeMeetingSchema,
  type CancelMeetingFormValues,
  type PostponeMeetingFormValues,
} from '@/lib/enterprise';
import type { ActaMeeting } from '@/types/suites';

interface MeetingStatusControlsProps {
  meeting: ActaMeeting;
  canManage: boolean;
  onStart: () => void;
  onEnd: () => void;
  onCancel: (values: CancelMeetingFormValues) => void;
  onPostpone: (values: PostponeMeetingFormValues) => void;
  pending?: boolean;
}

export function MeetingStatusControls({
  meeting,
  canManage,
  onStart,
  onEnd,
  onCancel,
  onPostpone,
  pending = false,
}: MeetingStatusControlsProps) {
  const [cancelOpen, setCancelOpen] = useState(false);
  const [postponeOpen, setPostponeOpen] = useState(false);
  const [endOpen, setEndOpen] = useState(false);
  const cancelForm = useForm<CancelMeetingFormValues>({
    resolver: zodResolver(cancelMeetingSchema),
    defaultValues: { reason: '' },
  });
  const postponeForm = useForm<PostponeMeetingFormValues>({
    resolver: zodResolver(postponeMeetingSchema),
    defaultValues: {
      new_scheduled_at: '',
      new_scheduled_end_at: '',
      reason: '',
    },
  });

  if (!canManage) {
    return null;
  }

  return (
    <>
      <div className="flex flex-wrap gap-2">
        {meeting.status === 'scheduled' ? (
          <>
            <Button onClick={onStart} disabled={pending}>
              <Play className="mr-1.5 h-4 w-4" />
              Start Meeting
            </Button>
            <Button variant="outline" onClick={() => setPostponeOpen(true)}>
              <CalendarClock className="mr-1.5 h-4 w-4" />
              Postpone
            </Button>
            <Button variant="outline" className="text-rose-600" onClick={() => setCancelOpen(true)}>
              <AlertTriangle className="mr-1.5 h-4 w-4" />
              Cancel
            </Button>
          </>
        ) : null}

        {meeting.status === 'in_progress' ? (
          <Button onClick={() => setEndOpen(true)} disabled={pending}>
            <Square className="mr-1.5 h-4 w-4" />
            End Meeting
          </Button>
        ) : null}
      </div>

      <Dialog open={cancelOpen} onOpenChange={setCancelOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Cancel Meeting</DialogTitle>
            <DialogDescription>
              Record a cancellation reason for the audit trail.
            </DialogDescription>
          </DialogHeader>
          <FormProvider {...cancelForm}>
            <form
              onSubmit={cancelForm.handleSubmit((values) => onCancel(values))}
              className="space-y-4"
            >
              <FormField name="reason" label="Reason" required>
                <Textarea {...cancelForm.register('reason')} rows={4} />
              </FormField>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setCancelOpen(false)}>
                  Close
                </Button>
                <Button type="submit" className="bg-rose-600 hover:bg-rose-700">
                  Cancel meeting
                </Button>
              </DialogFooter>
            </form>
          </FormProvider>
        </DialogContent>
      </Dialog>

      <Dialog open={postponeOpen} onOpenChange={setPostponeOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Postpone Meeting</DialogTitle>
            <DialogDescription>
              Move the meeting while retaining the original schedule history.
            </DialogDescription>
          </DialogHeader>
          <FormProvider {...postponeForm}>
            <form
              onSubmit={postponeForm.handleSubmit((values) => onPostpone(values))}
              className="space-y-4"
            >
              <FormField name="new_scheduled_at" label="New start" required>
                <Input type="datetime-local" {...postponeForm.register('new_scheduled_at')} />
              </FormField>
              <FormField name="new_scheduled_end_at" label="New end">
                <Input type="datetime-local" {...postponeForm.register('new_scheduled_end_at')} />
              </FormField>
              <FormField name="reason" label="Reason" required>
                <Textarea {...postponeForm.register('reason')} rows={4} />
              </FormField>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setPostponeOpen(false)}>
                  Close
                </Button>
                <Button type="submit">Postpone</Button>
              </DialogFooter>
            </form>
          </FormProvider>
        </DialogContent>
      </Dialog>

      <Dialog open={endOpen} onOpenChange={setEndOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>End Meeting</DialogTitle>
            <DialogDescription>
              Finalize attendance, compute quorum, and close the meeting.
            </DialogDescription>
          </DialogHeader>
          <div className="rounded-xl border px-4 py-3 text-sm text-muted-foreground">
            Attendance: {meeting.present_count}/{meeting.attendee_count}. Quorum:{' '}
            {meeting.quorum_met ? 'met' : 'pending recalculation'}.
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEndOpen(false)}>
              Keep open
            </Button>
            <Button type="button" onClick={onEnd}>
              End meeting
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
