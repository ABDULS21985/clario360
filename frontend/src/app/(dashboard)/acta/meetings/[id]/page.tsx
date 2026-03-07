'use client';

import { useMemo } from 'react';
import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { BookOpen, ClipboardList, FileText, Link2, MapPin } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { SectionCard } from '@/components/suites/section-card';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuiteData, fetchSuitePaginated } from '@/lib/suite-api';
import { meetingStatusConfig, minuteStatusConfig } from '@/lib/status-configs';
import { shortId, summarizeNamedRecords } from '@/lib/suite-utils';
import { formatDateTime, truncate } from '@/lib/utils';
import type { ActaActionItem, ActaMeeting, ActaMeetingMinute } from '@/types/suites';

export default function ActaMeetingDetailPage() {
  const { id } = useParams<{ id: string }>();

  const meetingQuery = useQuery({
    queryKey: ['acta-meeting', id],
    queryFn: () => fetchSuiteData<ActaMeeting>(`${API_ENDPOINTS.ACTA_MEETINGS}/${id}`),
    enabled: Boolean(id),
  });

  const actionsQuery = useQuery({
    queryKey: ['acta-meeting-actions', id],
    queryFn: () =>
      fetchSuitePaginated<ActaActionItem>(API_ENDPOINTS.ACTA_ACTION_ITEMS, {
        page: 1,
        per_page: 200,
        order: 'desc',
      }),
    enabled: Boolean(id),
  });

  const documentsQuery = useQuery({
    queryKey: ['acta-meeting-documents', id],
    queryFn: () =>
      fetchSuitePaginated<ActaMeetingMinute>(API_ENDPOINTS.ACTA_DOCUMENTS, {
        page: 1,
        per_page: 200,
        order: 'desc',
      }),
    enabled: Boolean(id),
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
  const actionItems = useMemo(
    () => (actionsQuery.data?.data ?? []).filter((item) => item.meeting_id === meeting.id),
    [actionsQuery.data?.data, meeting.id],
  );
  const documents = useMemo(
    () => (documentsQuery.data?.data ?? []).filter((document) => document.meeting_id === meeting.id),
    [documentsQuery.data?.data, meeting.id],
  );

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title={meeting.title} description={`Committee: ${meeting.committee_name}`} />

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
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
            <p className="text-sm text-muted-foreground">Attendees</p>
            <p className="mt-2 text-lg font-semibold">{summarizeNamedRecords(meeting.attendees, 3)}</p>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Action Items</p>
            <p className="mt-2 text-lg font-semibold">{actionItems.length}</p>
          </div>
        </div>

        <div className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
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

          <SectionCard title="Attendees" description="Participants resolved from the meeting payload.">
            <div className="space-y-2">
              {Array.isArray(meeting.attendees) && meeting.attendees.length > 0 ? (
                meeting.attendees.map((attendee, index) => (
                  <div key={`${index}-${JSON.stringify(attendee)}`} className="rounded-lg border px-4 py-3 text-sm">
                    {summarizeNamedRecords([attendee], 1)}
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No attendee list is available for this meeting.</p>
              )}
            </div>
          </SectionCard>
        </div>

        <div className="grid gap-4 xl:grid-cols-2">
          <SectionCard title="Action Items" description="Follow-ups tied to this meeting.">
            <div className="space-y-3">
              {actionItems.length === 0 ? (
                <p className="text-sm text-muted-foreground">No action items are linked to this meeting.</p>
              ) : (
                actionItems.map((item) => (
                  <div key={item.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{item.title}</p>
                        <p className="text-xs text-muted-foreground">{truncate(item.description, 140)}</p>
                      </div>
                      <Badge variant="outline" className="capitalize">{item.status.replace(/_/g, ' ')}</Badge>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
                      <span>Owner: {shortId(item.assigned_to)}</span>
                      <span>Due: {item.due_date ? formatDateTime(item.due_date) : 'No due date'}</span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard title="Minutes & Outcomes" description="Minute records and AI-generated summaries linked to this meeting.">
            <div className="space-y-3">
              {documents.length === 0 ? (
                <p className="text-sm text-muted-foreground">No minutes have been published for this meeting.</p>
              ) : (
                documents.map((document) => (
                  <div key={document.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <FileText className="h-4 w-4 text-muted-foreground" />
                          <p className="font-medium">{document.meeting_title}</p>
                        </div>
                        <p className="mt-2 text-sm text-muted-foreground">{document.ai_summary ?? truncate(document.content, 160)}</p>
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
      </div>
    </PermissionRedirect>
  );
}
