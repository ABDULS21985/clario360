'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { BookOpen, ClipboardList, FileText, Link2, MapPin, Paperclip, Pencil, Users } from 'lucide-react';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { StatusBadge } from '@/components/shared/status-badge';
import { SectionCard } from '@/components/suites/section-card';
import { meetingStatusConfig } from '@/lib/status-configs';
import { enterpriseApi, canApproveMinutes } from '@/lib/enterprise';
import { formatDateTime } from '@/lib/utils';
import { useAuth } from '@/hooks/use-auth';
import { AgendaTab } from './_components/agenda-tab';
import { AgendaVoteDialog } from './_components/agenda-vote-dialog';
import { AttendanceTab } from './_components/attendance-tab';
import { MinutesTab } from './_components/minutes-tab';
import { ActionItemsTab } from './_components/action-items-tab';
import { AttachmentsTab } from './_components/attachments-tab';
import { EditMeetingDialog } from './_components/edit-meeting-dialog';
import { MeetingStatusControls } from './_components/meeting-status-controls';
import type {
  ActaAgendaItem,
  ActaActionItem,
  ActaCommittee,
  ActaMeeting,
  ActaMeetingMinutes,
} from '@/types/suites';
import { showApiError, showSuccess } from '@/lib/toast';

export default function ActaMeetingDetailPage() {
  const queryClient = useQueryClient();
  const params = useParams<{ id: string }>();
  const id = params?.id ?? '';
  const { user } = useAuth();
  const [voteItem, setVoteItem] = useState<ActaAgendaItem | null>(null);
  const [editOpen, setEditOpen] = useState(false);

  const meetingQuery = useQuery({
    queryKey: ['acta-meeting', id],
    queryFn: () => enterpriseApi.acta.getMeeting(id),
    enabled: Boolean(id),
  });
  const committeeQuery = useQuery({
    queryKey: ['acta-meeting-committee', id, meetingQuery.data?.committee_id],
    queryFn: () => enterpriseApi.acta.getCommittee(meetingQuery.data!.committee_id),
    enabled: Boolean(meetingQuery.data?.committee_id),
  });
  const attendanceQuery = useQuery({
    queryKey: ['acta-meeting-attendance', id],
    queryFn: () => enterpriseApi.acta.getAttendance(id),
    enabled: Boolean(id),
  });
  const agendaQuery = useQuery({
    queryKey: ['acta-meeting-agenda', id],
    queryFn: () => enterpriseApi.acta.listAgenda(id),
    enabled: Boolean(id),
  });
  const minutesQuery = useQuery({
    queryKey: ['acta-meeting-minutes', id],
    queryFn: async () => {
      try {
        return await enterpriseApi.acta.getMinutes(id);
      } catch (error) {
        if (isMissing(error)) {
          return null;
        }
        throw error;
      }
    },
    enabled: Boolean(id),
  });
  const versionsQuery = useQuery({
    queryKey: ['acta-meeting-minutes-versions', id],
    queryFn: async () => {
      try {
        return await enterpriseApi.acta.listMinutesVersions(id);
      } catch (error) {
        if (isMissing(error)) {
          return [];
        }
        throw error;
      }
    },
    enabled: Boolean(id),
  });
  const actionsQuery = useQuery({
    queryKey: ['acta-meeting-actions', id],
    queryFn: async () => {
      const response = await enterpriseApi.acta.listActionItems({
        page: 1,
        per_page: 100,
        order: 'desc',
        filters: { meeting_id: id },
      });
      return response.data;
    },
    enabled: Boolean(id),
  });
  const attachmentsQuery = useQuery({
    queryKey: ['acta-meeting-attachments', id],
    queryFn: () => enterpriseApi.acta.listAttachments(id),
    enabled: Boolean(id),
  });
  const usersQuery = useQuery({
    queryKey: ['acta-meeting-users', id],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, order: 'asc' }),
    enabled: Boolean(id),
  });

  const refreshMeeting = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['acta-meeting', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meeting-committee', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meeting-attendance', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meeting-agenda', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meeting-minutes', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meeting-minutes-versions', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meeting-actions', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meeting-attachments', id] }),
      queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      queryClient.invalidateQueries({ queryKey: ['acta-meetings'] }),
    ]);
  };

  const startMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.startMeeting(id),
    onSuccess: async () => {
      showSuccess('Meeting started.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const endMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.endMeeting(id),
    onSuccess: async () => {
      showSuccess('Meeting completed.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const cancelMutation = useMutation({
    mutationFn: (values: { reason: string }) => enterpriseApi.acta.cancelMeeting(id, values),
    onSuccess: async () => {
      showSuccess('Meeting cancelled.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const postponeMutation = useMutation({
    mutationFn: (values: { new_scheduled_at: string; new_scheduled_end_at?: string | null; reason: string }) =>
      enterpriseApi.acta.postponeMeeting(id, {
        ...values,
        new_scheduled_at: new Date(values.new_scheduled_at).toISOString(),
        new_scheduled_end_at: values.new_scheduled_end_at
          ? new Date(values.new_scheduled_end_at).toISOString()
          : null,
      }),
    onSuccess: async () => {
      showSuccess('Meeting postponed.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const attendanceMutation = useMutation({
    mutationFn: (values: {
      user_id: string;
      status: 'present' | 'absent' | 'proxy' | 'excused';
      proxy_user_name?: string | null;
      proxy_authorized_by?: string | null;
    }) => enterpriseApi.acta.recordAttendance(id, values),
    onSuccess: async () => refreshMeeting(),
    onError: showApiError,
  });
  const bulkAttendanceMutation = useMutation({
    mutationFn: (values: Array<{ user_id: string; status: 'present' | 'absent' | 'proxy' | 'excused'; proxy_user_name?: string | null; proxy_authorized_by?: string | null }>) =>
      enterpriseApi.acta.bulkRecordAttendance(id, { attendance: values }),
    onSuccess: async () => refreshMeeting(),
    onError: showApiError,
  });
  const agendaCreateMutation = useMutation({
    mutationFn: (values: any) => enterpriseApi.acta.createAgendaItem(id, values),
    onSuccess: async () => {
      showSuccess('Agenda item created.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const agendaDeleteMutation = useMutation({
    mutationFn: (item: ActaAgendaItem) => enterpriseApi.acta.deleteAgendaItem(id, item.id),
    onSuccess: async () => refreshMeeting(),
    onError: showApiError,
  });
  const agendaReorderMutation = useMutation({
    mutationFn: (itemIds: string[]) => enterpriseApi.acta.reorderAgenda(id, itemIds),
    onSuccess: async () => refreshMeeting(),
    onError: showApiError,
  });
  const agendaNotesMutation = useMutation({
    mutationFn: ({ itemId, notes }: { itemId: string; notes: string }) =>
      enterpriseApi.acta.updateAgendaNotes(id, itemId, notes),
    onError: showApiError,
  });
  const voteMutation = useMutation({
    mutationFn: ({ item, values }: { item: ActaAgendaItem; values: any }) =>
      enterpriseApi.acta.voteAgendaItem(id, item.id, values),
    onSuccess: async () => {
      setVoteItem(null);
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const minutesGenerateMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.generateMinutes(id),
    onSuccess: async () => {
      showSuccess('Minutes generated.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const minutesCreateMutation = useMutation({
    mutationFn: (content: string) => enterpriseApi.acta.createMinutes(id, content),
    onSuccess: async () => {
      showSuccess('Minutes created.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const minutesSaveMutation = useMutation({
    mutationFn: (content: string) => enterpriseApi.acta.updateMinutes(id, content),
    onSuccess: async () => refreshMeeting(),
    onError: showApiError,
  });
  const minutesSubmitMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.submitMinutes(id),
    onSuccess: async () => {
      showSuccess('Minutes submitted for review.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const minutesRevisionMutation = useMutation({
    mutationFn: (notes: string) => enterpriseApi.acta.requestMinutesRevision(id, notes),
    onSuccess: async () => {
      showSuccess('Revision requested.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const minutesApproveMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.approveMinutes(id),
    onSuccess: async () => {
      showSuccess('Minutes approved.');
      await refreshMeeting();
    },
    onError: showApiError,
  });
  const minutesPublishMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.publishMinutes(id),
    onSuccess: async () => {
      showSuccess('Minutes published.');
      await refreshMeeting();
    },
    onError: showApiError,
  });

  if (meetingQuery.isLoading) {
    return (
      <PermissionRedirect permission="acta:read">
        <div className="space-y-6">
          <PageHeader title="Meeting Details" description={`Meeting ID: ${id}`} />
          <LoadingSkeleton variant="card" count={3} />
        </div>
      </PermissionRedirect>
    );
  }

  if (meetingQuery.error || !meetingQuery.data) {
    return (
      <PermissionRedirect permission="acta:read">
        <ErrorState message="Failed to load meeting details." onRetry={() => void meetingQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const meeting = meetingQuery.data;
  const committee = committeeQuery.data as ActaCommittee | undefined;
  const attendance = attendanceQuery.data ?? meeting.attendance ?? [];
  const agenda = agendaQuery.data ?? meeting.agenda ?? [];
  const actionItems = actionsQuery.data ?? [];
  const minutes = minutesQuery.data as ActaMeetingMinutes | null;
  const versions = versionsQuery.data ?? [];
  const attachments = attachmentsQuery.data ?? meeting.attachments ?? [];
  const canManage =
    Boolean(committee && user?.id && (committee.chair_user_id === user.id || committee.secretary_user_id === user.id));
  const canEdit = canManage && (meeting.status === 'draft' || meeting.status === 'scheduled');
  const canApprove = canApproveMinutes(minutes, committee, user?.id);

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title={meeting.title}
          description={`Committee: ${meeting.committee_name}`}
          actions={
            <div className="flex items-center gap-2">
              {canEdit ? (
                <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                  <Pencil className="mr-1.5 h-4 w-4" />
                  Edit
                </Button>
              ) : null}
            <MeetingStatusControls
              meeting={meeting}
              canManage={canManage}
              onStart={() => startMutation.mutate()}
              onEnd={() => endMutation.mutate()}
              onCancel={(values) => cancelMutation.mutate(values)}
              onPostpone={(values) => postponeMutation.mutate(values)}
              pending={
                startMutation.isPending ||
                endMutation.isPending ||
                cancelMutation.isPending ||
                postponeMutation.isPending
              }
            />
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Status</p>
            <div className="mt-2">
              <StatusBadge status={meeting.status} config={meetingStatusConfig} />
            </div>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Scheduled</p>
            <p className="mt-2 text-lg font-semibold">{formatDateTime(meeting.scheduled_at)}</p>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Quorum</p>
            <p className="mt-2 text-lg font-semibold">
              {meeting.present_count}/{meeting.attendee_count} present
            </p>
            <p className="text-xs text-muted-foreground">
              {meeting.quorum_met ? 'Quorum met' : 'Quorum pending / not met'}
            </p>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Action Items</p>
            <p className="mt-2 text-lg font-semibold">{actionItems.length}</p>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.05fr_0.95fr]">
          <SectionCard title="Meeting Context" description="Core scheduling and participation context.">
            <div className="space-y-4">
              <div className="rounded-lg border px-4 py-3">
                <div className="flex items-center gap-2 text-sm font-medium">
                  <BookOpen className="h-4 w-4 text-muted-foreground" />
                  {meeting.committee_name}
                </div>
                {meeting.description ? <p className="mt-2 text-sm text-muted-foreground">{meeting.description}</p> : null}
              </div>
              {meeting.location ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <MapPin className="h-4 w-4" />
                  {meeting.location}
                </div>
              ) : null}
              {meeting.virtual_link ? (
                <div className="flex items-center gap-2 text-sm">
                  <Link2 className="h-4 w-4 text-muted-foreground" />
                  <a className="text-primary hover:underline" href={meeting.virtual_link} target="_blank" rel="noreferrer">
                    Join virtual session
                  </a>
                </div>
              ) : null}
              <div className="flex flex-wrap gap-2">
                {meeting.duration_minutes ? <Badge variant="outline">{meeting.duration_minutes} minutes</Badge> : null}
                <Badge variant="outline">{meeting.action_item_count} linked actions</Badge>
              </div>
            </div>
          </SectionCard>

          <SectionCard title="Meeting Workspace" description="Conduct the session across agenda, attendance, minutes, and follow-ups.">
            <div className="grid gap-2 text-sm text-muted-foreground">
              <div className="inline-flex items-center gap-2">
                <BookOpen className="h-4 w-4" />
                Agenda items: {agenda.length}
              </div>
              <div className="inline-flex items-center gap-2">
                <Users className="h-4 w-4" />
                Attendance records: {attendance.length}
              </div>
              <div className="inline-flex items-center gap-2">
                <FileText className="h-4 w-4" />
                Minutes: {minutes ? `v${minutes.version}` : 'Not started'}
              </div>
              <div className="inline-flex items-center gap-2">
                <Paperclip className="h-4 w-4" />
                Attachments: {attachments.length}
              </div>
            </div>
          </SectionCard>
        </div>

        <Tabs defaultValue="agenda" className="space-y-4">
          <TabsList className="w-full justify-start">
            <TabsTrigger value="agenda">Agenda</TabsTrigger>
            <TabsTrigger value="attendance">Attendance</TabsTrigger>
            <TabsTrigger value="minutes">Minutes</TabsTrigger>
            <TabsTrigger value="actions">Action Items</TabsTrigger>
            <TabsTrigger value="attachments">Attachments</TabsTrigger>
          </TabsList>

          <TabsContent value="agenda">
            <AgendaTab
              meeting={meeting}
              items={agenda}
              presentCount={meeting.present_count}
              users={usersQuery.data?.data ?? []}
              onCreate={(values) => agendaCreateMutation.mutate(values)}
              onDelete={(item) => agendaDeleteMutation.mutate(item)}
              onReorder={(itemIds) => agendaReorderMutation.mutate(itemIds)}
              onRecordVote={(item) => setVoteItem(item)}
              onSaveNotes={(itemId, notes) => agendaNotesMutation.mutate({ itemId, notes })}
            />
          </TabsContent>
          <TabsContent value="attendance">
            <AttendanceTab
              meeting={meeting}
              attendance={attendance}
              currentUserId={user?.id}
              onSaveAttendance={(values) => attendanceMutation.mutate(values)}
              onBulkAbsent={(values) => bulkAttendanceMutation.mutate(values)}
            />
          </TabsContent>
          <TabsContent value="minutes">
            {committee ? (
              <MinutesTab
                meeting={meeting}
                committee={committee}
                minutes={minutes}
                versions={versions}
                actionItems={actionItems}
                canApprove={canApprove}
                pending={
                  minutesGenerateMutation.isPending ||
                  minutesCreateMutation.isPending ||
                  minutesSaveMutation.isPending ||
                  minutesSubmitMutation.isPending ||
                  minutesRevisionMutation.isPending ||
                  minutesApproveMutation.isPending ||
                  minutesPublishMutation.isPending
                }
                onGenerate={() => minutesGenerateMutation.mutate()}
                onCreate={(content) => minutesCreateMutation.mutate(content)}
                onSave={(content) => minutesSaveMutation.mutate(content)}
                onSubmitReview={() => minutesSubmitMutation.mutate()}
                onRequestRevision={(notes) => minutesRevisionMutation.mutate(notes)}
                onApprove={() => minutesApproveMutation.mutate()}
                onPublish={() => minutesPublishMutation.mutate()}
              />
            ) : (
              <LoadingSkeleton variant="card" count={2} />
            )}
          </TabsContent>
          <TabsContent value="actions">
            {committee ? (
              <ActionItemsTab meeting={meeting} committee={committee} items={actionItems} />
            ) : null}
          </TabsContent>
          <TabsContent value="attachments">
            <AttachmentsTab
              meetingId={meeting.id}
              attachments={attachments}
              currentUserId={user?.id}
              onRefresh={refreshMeeting}
            />
          </TabsContent>
        </Tabs>

        <AgendaVoteDialog
          open={Boolean(voteItem)}
          onOpenChange={(open) => !open && setVoteItem(null)}
          item={voteItem}
          presentCount={meeting.present_count}
          onSubmit={(values) => voteItem && voteMutation.mutate({ item: voteItem, values })}
          pending={voteMutation.isPending}
        />

        <EditMeetingDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          meeting={meeting}
          onSuccess={refreshMeeting}
        />
      </div>
    </PermissionRedirect>
  );
}

function isMissing(error: unknown) {
  return typeof error === 'object' && error !== null && 'status' in error && (error as { status?: number }).status === 404;
}
