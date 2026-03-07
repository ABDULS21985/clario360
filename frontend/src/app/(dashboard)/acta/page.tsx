'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, BookOpen, Building2, ClipboardList, Users } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { KpiCard } from '@/components/shared/kpi-card';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SectionCard } from '@/components/suites/section-card';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { meetingStatusConfig, minuteStatusConfig } from '@/lib/status-configs';
import { summarizeNamedRecords } from '@/lib/suite-utils';
import type { ActaActionItem, ActaCommittee, ActaMeeting, ActaMeetingMinute } from '@/types/suites';

export default function ActaPage() {
  const committeesQuery = useQuery({
    queryKey: ['acta-overview', 'committees'],
    queryFn: () => fetchSuitePaginated<ActaCommittee>(API_ENDPOINTS.ACTA_COMMITTEES, { page: 1, per_page: 5, order: 'desc' }),
  });
  const meetingsQuery = useQuery({
    queryKey: ['acta-overview', 'meetings'],
    queryFn: () => fetchSuitePaginated<ActaMeeting>(API_ENDPOINTS.ACTA_MEETINGS, { page: 1, per_page: 8, order: 'desc' }),
  });
  const actionsQuery = useQuery({
    queryKey: ['acta-overview', 'actions'],
    queryFn: () => fetchSuitePaginated<ActaActionItem>(API_ENDPOINTS.ACTA_ACTION_ITEMS, { page: 1, per_page: 8, order: 'desc' }),
  });
  const documentsQuery = useQuery({
    queryKey: ['acta-overview', 'documents'],
    queryFn: () => fetchSuitePaginated<ActaMeetingMinute>(API_ENDPOINTS.ACTA_DOCUMENTS, { page: 1, per_page: 6, order: 'desc' }),
  });

  if (committeesQuery.isLoading && meetingsQuery.isLoading && actionsQuery.isLoading && documentsQuery.isLoading) {
    return (
      <PermissionRedirect permission="acta:read">
        <div className="space-y-6">
          <PageHeader title="Board Governance" description="Board committees, meetings, and action items" />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (committeesQuery.error && meetingsQuery.error && actionsQuery.error && documentsQuery.error) {
    return (
      <PermissionRedirect permission="acta:read">
        <ErrorState
          message="Failed to load board governance overview."
          onRetry={() => {
            void committeesQuery.refetch();
            void meetingsQuery.refetch();
            void actionsQuery.refetch();
            void documentsQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  const meetings = [...(meetingsQuery.data?.data ?? [])].sort(
    (left, right) => new Date(left.scheduled_at).getTime() - new Date(right.scheduled_at).getTime(),
  );
  const actionItems = actionsQuery.data?.data ?? [];
  const documents = documentsQuery.data?.data ?? [];
  const openActionItems = actionItems.filter((item) => item.status !== 'completed');
  const upcomingMeetings = meetings.filter((meeting) => new Date(meeting.scheduled_at).getTime() >= Date.now());
  const publishedMinutes = documents.filter((document) => document.status === 'published');

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title="Board Governance"
          description="Live view of committees, meetings, action tracking, and minute publication."
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" asChild>
                <Link href="/acta/committees">Committees</Link>
              </Button>
              <Button size="sm" asChild>
                <Link href="/acta/meetings">Meetings</Link>
              </Button>
            </div>
          }
        />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Committees" value={committeesQuery.data?.meta.total ?? 0} icon={Users} iconColor="text-blue-600" />
          <KpiCard title="Upcoming Meetings" value={upcomingMeetings.length} icon={BookOpen} iconColor="text-violet-600" />
          <KpiCard title="Open Actions" value={openActionItems.length} icon={ClipboardList} iconColor="text-orange-600" />
          <KpiCard title="Published Minutes" value={publishedMinutes.length} icon={Building2} iconColor="text-green-600" />
        </div>

        <div className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
          <SectionCard
            title="Upcoming Meetings"
            description="Scheduled meetings ordered by upcoming date."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/acta/meetings">
                  View all
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {upcomingMeetings.length === 0 ? (
                <p className="text-sm text-muted-foreground">No upcoming meetings are scheduled.</p>
              ) : (
                upcomingMeetings.slice(0, 6).map((meeting) => (
                  <div key={meeting.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <Link href={`/acta/meetings/${meeting.id}`} className="font-medium hover:underline">
                          {meeting.title}
                        </Link>
                        <p className="text-xs text-muted-foreground">{meeting.committee_name}</p>
                      </div>
                      <StatusBadge status={meeting.status} config={meetingStatusConfig} size="sm" />
                    </div>
                    <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
                      <span>{new Date(meeting.scheduled_at).toLocaleString()}</span>
                      <span>{meeting.action_item_count} action items</span>
                      <span>{summarizeNamedRecords(meeting.attendees)}</span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard title="Board Pack Readiness" description="Recent minutes and document publication workflow.">
            <div className="space-y-3">
              {documents.length === 0 ? (
                <p className="text-sm text-muted-foreground">No meeting minutes are available yet.</p>
              ) : (
                documents.map((document) => (
                  <div key={document.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{document.meeting_title}</p>
                        <p className="text-xs text-muted-foreground">{document.ai_summary ?? 'No AI summary available'}</p>
                      </div>
                      <StatusBadge status={document.status} config={minuteStatusConfig} size="sm" />
                    </div>
                    <div className="mt-2 text-xs text-muted-foreground">
                      <RelativeTime date={document.updated_at} />
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>
        </div>

        <SectionCard
          title="Open Action Items"
          description="Follow-ups that still require execution or tracking."
          actions={
            <Button variant="ghost" size="sm" asChild>
              <Link href="/acta/action-items">
                Open actions
                <ArrowRight className="ml-1 h-3.5 w-3.5" />
              </Link>
            </Button>
          }
        >
          <div className="space-y-3">
            {openActionItems.length === 0 ? (
              <p className="text-sm text-muted-foreground">No open board action items remain.</p>
            ) : (
              openActionItems.slice(0, 6).map((item) => (
                <div key={item.id} className="rounded-lg border px-4 py-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-medium">{item.title}</p>
                      <p className="text-xs text-muted-foreground">{item.meeting_title}</p>
                    </div>
                    <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">{item.status.replace(/_/g, ' ')}</span>
                  </div>
                  <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
                    <span>Owner: {item.assigned_to ? `${item.assigned_to.slice(0, 8)}…` : 'Unassigned'}</span>
                    <span>Due: {item.due_date ? new Date(item.due_date).toLocaleDateString() : 'No due date'}</span>
                  </div>
                </div>
              ))
            )}
          </div>
        </SectionCard>
      </div>
    </PermissionRedirect>
  );
}
