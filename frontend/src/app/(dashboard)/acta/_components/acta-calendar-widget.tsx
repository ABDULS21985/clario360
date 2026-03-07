'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import {
  addMonths,
  eachDayOfInterval,
  endOfMonth,
  endOfWeek,
  format,
  isSameMonth,
  isToday,
  parseISO,
  startOfMonth,
  startOfWeek,
  subMonths,
} from 'date-fns';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { SectionCard } from '@/components/suites/section-card';
import type { ActaCalendarDay, ActaMeetingSummary } from '@/types/suites';

interface ActaCalendarWidgetProps {
  month: string;
  days: ActaCalendarDay[];
  onMonthChange: (month: string) => void;
}

const WEEK_DAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const PALETTE = ['bg-sky-500', 'bg-emerald-500', 'bg-amber-500', 'bg-rose-500', 'bg-violet-500', 'bg-cyan-500'];

export function ActaCalendarWidget({
  month,
  days,
  onMonthChange,
}: ActaCalendarWidgetProps) {
  const monthDate = useMemo(() => parseISO(`${month}-01`), [month]);
  const dayMap = useMemo(
    () =>
      new Map(
        days.map((day) => [
          day.date,
          [...day.meetings].sort((left, right) =>
            left.scheduled_at.localeCompare(right.scheduled_at),
          ),
        ]),
      ),
    [days],
  );
  const calendarDays = useMemo(() => {
    const start = startOfWeek(startOfMonth(monthDate), { weekStartsOn: 1 });
    const end = endOfWeek(endOfMonth(monthDate), { weekStartsOn: 1 });
    return eachDayOfInterval({ start, end });
  }, [monthDate]);
  const [selectedDate, setSelectedDate] = useState<string | null>(days[0]?.date ?? null);

  const selectedMeetings = selectedDate ? dayMap.get(selectedDate) ?? [] : [];

  return (
    <SectionCard
      title="Meeting Calendar"
      description="Monthly view of scheduled committee sessions."
      actions={
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onMonthChange(format(subMonths(monthDate, 1), 'yyyy-MM'))}
            aria-label="Previous month"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <div className="min-w-[7rem] text-center text-sm font-medium">
            {format(monthDate, 'MMMM yyyy')}
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onMonthChange(format(addMonths(monthDate, 1), 'yyyy-MM'))}
            aria-label="Next month"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      }
    >
      <div className="grid grid-cols-7 gap-2 text-center text-xs font-medium text-muted-foreground">
        {WEEK_DAYS.map((name) => (
          <div key={name}>{name}</div>
        ))}
      </div>

      <div className="mt-3 grid grid-cols-7 gap-2">
        {calendarDays.map((day) => {
          const iso = format(day, 'yyyy-MM-dd');
          const meetings = dayMap.get(iso) ?? [];
          return (
            <button
              key={iso}
              type="button"
              onClick={() => meetings.length > 0 && setSelectedDate(iso)}
              className={cn(
                'min-h-24 rounded-xl border p-2 text-left transition',
                meetings.length > 0 ? 'hover:border-primary hover:bg-accent/40' : 'cursor-default',
                selectedDate === iso && 'border-primary bg-accent/30',
                !isSameMonth(day, monthDate) && 'opacity-45',
                isToday(day) && 'ring-1 ring-primary/30',
              )}
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium">{format(day, 'd')}</span>
                {meetings.length > 0 ? (
                  <span className="text-[10px] text-muted-foreground">
                    {meetings.length}
                  </span>
                ) : null}
              </div>
              <div className="mt-3 flex flex-wrap gap-1">
                {meetings.slice(0, 4).map((meeting) => (
                  <span
                    key={meeting.id}
                    className={cn(
                      'inline-flex h-2.5 w-2.5 rounded-full',
                      PALETTE[colorIndex(meeting.committee_name)],
                    )}
                    title={`${meeting.committee_name}: ${meeting.title}`}
                  />
                ))}
              </div>
              <div className="mt-2 space-y-1">
                {meetings.slice(0, 2).map((meeting) => (
                  <div key={meeting.id} className="truncate text-[11px] text-muted-foreground">
                    {meeting.title}
                  </div>
                ))}
              </div>
            </button>
          );
        })}
      </div>

      <div className="mt-4 rounded-xl border bg-muted/20 p-4">
        <div className="mb-2 flex items-center justify-between">
          <div className="text-sm font-medium">
            {selectedDate ? format(parseISO(selectedDate), 'MMMM d, yyyy') : 'Select a day'}
          </div>
          {selectedMeetings.length > 0 && selectedDate ? (
            <span className="text-xs text-muted-foreground">
              {selectedMeetings.length} meeting{selectedMeetings.length === 1 ? '' : 's'}
            </span>
          ) : null}
        </div>

        {selectedMeetings.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No meetings scheduled for the selected day.
          </p>
        ) : (
          <div className="space-y-3">
            {selectedMeetings.map((meeting) => (
              <Link
                key={meeting.id}
                href={`/acta/meetings/${meeting.id}`}
                className="block rounded-lg border bg-background px-4 py-3 transition hover:border-primary"
              >
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate font-medium">{meeting.title}</p>
                    <p className="truncate text-xs text-muted-foreground">
                      {meeting.committee_name}
                    </p>
                  </div>
                  <div className="text-right text-xs text-muted-foreground">
                    <div>{format(parseISO(meeting.scheduled_at), 'p')}</div>
                    {meeting.location ? <div>{meeting.location}</div> : null}
                  </div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </SectionCard>
  );
}

function colorIndex(value: string) {
  return Array.from(value).reduce((sum, char) => sum + char.charCodeAt(0), 0) % PALETTE.length;
}
