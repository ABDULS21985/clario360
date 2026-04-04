'use client';

import Link from 'next/link';
import { format, parseISO } from 'date-fns';
import { ArrowRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { SectionCard } from '@/components/suites/section-card';
import { EmptyState } from '@/components/common/empty-state';
import { Calendar } from 'lucide-react';
import type { ActaMeetingSummary } from '@/types/suites';

interface ActaUpcomingMeetingsProps {
  meetings: ActaMeetingSummary[];
}

export function ActaUpcomingMeetings({ meetings }: ActaUpcomingMeetingsProps) {
  return (
    <SectionCard
      title="Upcoming Meetings"
      description="Next scheduled committee sessions."
      actions={
        <Button variant="ghost" size="sm" asChild>
          <Link href="/acta/meetings">
            All meetings
            <ArrowRight className="ml-1 h-3.5 w-3.5" />
          </Link>
        </Button>
      }
    >
      {meetings.length === 0 ? (
        <EmptyState
          icon={Calendar}
          title="No upcoming meetings"
          description="The schedule is currently clear for the next committee sessions."
        />
      ) : (
        <div className="space-y-3">
          {meetings.slice(0, 5).map((meeting) => (
            <Link
              key={meeting.id}
              href={`/acta/meetings/${meeting.id}`}
              className="block rounded-xl border px-4 py-3 transition hover:border-primary"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="truncate font-medium">{meeting.title}</p>
                    <Badge variant="outline" className="capitalize">
                      {meeting.status.replace(/_/g, ' ')}
                    </Badge>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {meeting.committee_name}
                  </p>
                </div>
                <div className="text-right text-xs text-muted-foreground">
                  <div>{format(parseISO(meeting.scheduled_at), 'MMM d, yyyy')}</div>
                  <div>{format(parseISO(meeting.scheduled_at), 'p')}</div>
                </div>
              </div>
              <div className="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground">
                <span>{meeting.location ?? 'Location to be confirmed'}</span>
                <span>{meeting.duration_minutes} min</span>
                <span>
                  {meeting.quorum_met === null || meeting.quorum_met === undefined
                    ? 'Quorum pending'
                    : meeting.quorum_met
                      ? 'Quorum met'
                      : 'Quorum not met'}
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </SectionCard>
  );
}
