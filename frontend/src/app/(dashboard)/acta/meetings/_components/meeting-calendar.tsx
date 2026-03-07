'use client';

import Link from 'next/link';
import {
  addMonths,
  eachDayOfInterval,
  endOfMonth,
  endOfWeek,
  format,
  isSameMonth,
  parseISO,
  startOfMonth,
  startOfWeek,
  subMonths,
} from 'date-fns';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import type { ActaCalendarDay } from '@/types/suites';

interface MeetingCalendarProps {
  month: string;
  days: ActaCalendarDay[];
  onMonthChange: (month: string) => void;
}

export function MeetingCalendar({ month, days, onMonthChange }: MeetingCalendarProps) {
  const monthDate = parseISO(`${month}-01`);
  const start = startOfWeek(startOfMonth(monthDate), { weekStartsOn: 1 });
  const end = endOfWeek(endOfMonth(monthDate), { weekStartsOn: 1 });
  const dayMap = new Map(days.map((day) => [day.date, day.meetings]));
  const grid = eachDayOfInterval({ start, end });

  return (
    <div className="rounded-xl border bg-card p-4">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <p className="text-lg font-semibold">{format(monthDate, 'MMMM yyyy')}</p>
          <p className="text-sm text-muted-foreground">Calendar view of meetings and board sessions.</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon" onClick={() => onMonthChange(format(subMonths(monthDate, 1), 'yyyy-MM'))}>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button variant="outline" size="icon" onClick={() => onMonthChange(format(addMonths(monthDate, 1), 'yyyy-MM'))}>
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-7 gap-3 text-center text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        {['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((day) => (
          <div key={day}>{day}</div>
        ))}
      </div>

      <div className="mt-3 grid grid-cols-7 gap-3">
        {grid.map((day) => {
          const iso = format(day, 'yyyy-MM-dd');
          const meetings = dayMap.get(iso) ?? [];
          return (
            <div
              key={iso}
              className={cn(
                'min-h-32 rounded-xl border p-3',
                !isSameMonth(day, monthDate) && 'opacity-45',
              )}
            >
              <div className="mb-2 text-sm font-medium">{format(day, 'd')}</div>
              <div className="space-y-2">
                {meetings.map((meeting) => (
                  <Link
                    key={meeting.id}
                    href={`/acta/meetings/${meeting.id}`}
                    className="block rounded-lg bg-muted px-2 py-1.5 text-xs transition hover:bg-accent"
                  >
                    <p className="truncate font-medium">{meeting.title}</p>
                    <p className="truncate text-muted-foreground">{format(parseISO(meeting.scheduled_at), 'p')}</p>
                  </Link>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
