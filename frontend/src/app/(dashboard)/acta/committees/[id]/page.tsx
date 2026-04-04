'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Calendar, FileText, Pencil, Trash2 } from 'lucide-react';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { SectionCard } from '@/components/suites/section-card';
import { StatusBadge } from '@/components/shared/status-badge';
import { committeeStatusConfig, meetingStatusConfig, actionItemStatusConfig } from '@/lib/status-configs';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import { formatDate } from '@/lib/utils';
import { CommitteeStats } from '../_components/committee-stats';
import { MemberManagement } from '../_components/member-management';
import { EditCommitteeDialog } from './_components/edit-committee-dialog';

export default function CommitteeDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();
  const committeeId = params?.id ?? '';
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const committeeQuery = useQuery({
    queryKey: ['acta-committee', committeeId],
    queryFn: () => enterpriseApi.acta.getCommittee(committeeId),
    enabled: Boolean(committeeId),
  });
  const meetingsQuery = useQuery({
    queryKey: ['acta-committee-meetings', committeeId],
    queryFn: () =>
      enterpriseApi.acta.listMeetings({
        page: 1,
        per_page: 8,
        order: 'desc',
        filters: { committee_id: committeeId },
      }),
    enabled: Boolean(committeeId),
  });
  const actionsQuery = useQuery({
    queryKey: ['acta-committee-actions', committeeId],
    queryFn: () =>
      enterpriseApi.acta.listActionItems({
        page: 1,
        per_page: 8,
        order: 'desc',
        filters: { committee_id: committeeId },
      }),
    enabled: Boolean(committeeId),
  });

  const deleteMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.deleteCommittee(committeeId),
    onSuccess: async () => {
      showSuccess('Committee deleted.', 'The committee has been permanently removed.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-committees'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
      router.push('/acta/committees');
    },
    onError: showApiError,
  });

  if (committeeQuery.isLoading) {
    return (
      <PermissionRedirect permission="acta:read">
        <LoadingSkeleton variant="card" count={4} />
      </PermissionRedirect>
    );
  }

  if (committeeQuery.error || !committeeQuery.data) {
    return (
      <PermissionRedirect permission="acta:read">
        <ErrorState
          message="Failed to load committee details."
          onRetry={() => void committeeQuery.refetch()}
        />
      </PermissionRedirect>
    );
  }

  const committee = committeeQuery.data;
  const memberCount = committee.members?.filter((member) => member.active).length ?? 0;
  const meetings = meetingsQuery.data?.data ?? [];
  const actions = actionsQuery.data?.data ?? [];

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title={committee.name}
          description={committee.description}
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                <Pencil className="mr-1.5 h-4 w-4" />
                Edit
              </Button>
              <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="mr-1.5 h-4 w-4" />
                Delete
              </Button>
              <StatusBadge status={committee.status} config={committeeStatusConfig} />
            </div>
          }
        />

        <CommitteeStats stats={committee.stats} memberCount={memberCount} />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.2fr_0.8fr]">
          <SectionCard title="Committee Profile" description="Governance mandate and operating model.">
            <div className="space-y-4">
              <div className="flex flex-wrap gap-2">
                <Badge variant="outline" className="capitalize">
                  {committee.type.replace(/_/g, ' ')}
                </Badge>
                <Badge variant="outline" className="capitalize">
                  {committee.meeting_frequency.replace(/_/g, ' ')}
                </Badge>
                <Badge variant="outline">
                  Quorum{' '}
                  {committee.quorum_type === 'fixed_count'
                    ? committee.quorum_fixed_count ?? 0
                    : `${committee.quorum_percentage}%`}
                </Badge>
              </div>
              <div className="rounded-xl border px-4 py-3">
                <p className="text-sm font-medium">Charter</p>
                <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
                  {committee.charter || 'No charter text is currently recorded.'}
                </p>
              </div>
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                <div className="rounded-xl border px-4 py-3">
                  <p className="text-sm font-medium">Established</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {committee.established_date ? formatDate(committee.established_date) : 'Not recorded'}
                  </p>
                </div>
                <div className="rounded-xl border px-4 py-3">
                  <p className="text-sm font-medium">Tags</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {committee.tags.length === 0 ? (
                      <span className="text-sm text-muted-foreground">No tags</span>
                    ) : (
                      committee.tags.map((tag) => (
                        <Badge key={tag} variant="outline">
                          {tag}
                        </Badge>
                      ))
                    )}
                  </div>
                </div>
              </div>
            </div>
          </SectionCard>

          <MemberManagement committee={committee} />
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <SectionCard
            title="Recent Meetings"
            description="Latest scheduled and completed sessions for this committee."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href={`/acta/meetings?committee_id=${committee.id}`}>All meetings</Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {meetings.length === 0 ? (
                <p className="text-sm text-muted-foreground">No meetings found for this committee.</p>
              ) : (
                meetings.map((meeting) => (
                  <Link
                    key={meeting.id}
                    href={`/acta/meetings/${meeting.id}`}
                    className="block rounded-xl border px-4 py-3 transition hover:border-primary"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="truncate font-medium">{meeting.title}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatDate(meeting.scheduled_at)} • {meeting.duration_minutes} min
                        </p>
                      </div>
                      <StatusBadge status={meeting.status} config={meetingStatusConfig} size="sm" />
                    </div>
                    <div className="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground">
                      <span>{meeting.location ?? 'Location TBD'}</span>
                      <span>{meeting.present_count}/{meeting.attendee_count} present</span>
                    </div>
                  </Link>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard
            title="Open Actions"
            description="Current committee follow-ups and due dates."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href={`/acta/action-items?committee_id=${committee.id}`}>Open tracker</Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {actions.length === 0 ? (
                <p className="text-sm text-muted-foreground">No action items found for this committee.</p>
              ) : (
                actions.map((action) => (
                  <div key={action.id} className="rounded-xl border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="truncate font-medium">{action.title}</p>
                        <p className="text-xs text-muted-foreground">
                          {action.assignee_name} • due {formatDate(action.due_date)}
                        </p>
                      </div>
                      <StatusBadge status={action.status} config={actionItemStatusConfig} size="sm" />
                    </div>
                    <div className="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground">
                      <span className="capitalize">{action.priority} priority</span>
                      {action.meeting_id ? (
                        <span className="inline-flex items-center gap-1">
                          <Calendar className="h-3.5 w-3.5" />
                          Meeting linked
                        </span>
                      ) : null}
                      {action.completed_at ? (
                        <span className="inline-flex items-center gap-1">
                          <FileText className="h-3.5 w-3.5" />
                          Completed
                        </span>
                      ) : null}
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>
        </div>

        <EditCommitteeDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          committee={committee}
        />

        <ConfirmDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          title="Delete Committee"
          description={`Are you sure you want to delete "${committee.name}"? This action cannot be undone and will remove all associated records.`}
          confirmLabel="Delete committee"
          variant="destructive"
          typeToConfirm={committee.name}
          onConfirm={() => deleteMutation.mutateAsync()}
          loading={deleteMutation.isPending}
        />
      </div>
    </PermissionRedirect>
  );
}
