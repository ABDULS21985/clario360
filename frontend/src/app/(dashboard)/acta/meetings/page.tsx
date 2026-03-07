'use client';

import { useState } from 'react';
import { CalendarDays, List } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { Button } from '@/components/ui/button';
import { useQuery } from '@tanstack/react-query';
import { enterpriseApi } from '@/lib/enterprise';
import { actaMonthString } from '@/lib/enterprise';
import { meetingColumns } from './_components/meeting-columns';
import { MeetingFilters } from './_components/meeting-filters';
import { ScheduleMeetingDialog } from './_components/schedule-meeting-dialog';
import { MeetingCalendar } from './_components/meeting-calendar';
import type { ActaMeeting } from '@/types/suites';

export default function ActaMeetingsPage() {
  const [view, setView] = useState<'table' | 'calendar'>('table');
  const [month, setMonth] = useState(actaMonthString(new Date()));
  const [dialogOpen, setDialogOpen] = useState(false);
  const { tableProps } = useDataTable<ActaMeeting>({
    queryKey: 'acta-meetings',
    fetchFn: (params) => enterpriseApi.acta.listMeetings(params),
    defaultPageSize: 25,
    defaultSort: { column: 'scheduled_at', direction: 'desc' },
  });
  const committeesQuery = useQuery({
    queryKey: ['acta-meeting-committees'],
    queryFn: () => enterpriseApi.acta.listCommittees({ page: 1, per_page: 100, order: 'asc' }),
  });
  const calendarQuery = useQuery({
    queryKey: ['acta-meeting-calendar', month],
    queryFn: () => enterpriseApi.acta.getCalendar(month),
  });

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title="Meetings"
          description="Schedule, track, and conduct board and committee meetings."
          actions={
            <>
              <div className="flex rounded-lg border p-1">
                <Button variant={view === 'table' ? 'default' : 'ghost'} size="sm" onClick={() => setView('table')}>
                  <List className="mr-1.5 h-4 w-4" />
                  Table
                </Button>
                <Button variant={view === 'calendar' ? 'default' : 'ghost'} size="sm" onClick={() => setView('calendar')}>
                  <CalendarDays className="mr-1.5 h-4 w-4" />
                  Calendar
                </Button>
              </div>
              <Button onClick={() => setDialogOpen(true)}>Schedule Meeting</Button>
            </>
          }
        />

        <MeetingFilters
          search={tableProps.searchValue ?? ''}
          onSearchChange={(value) => tableProps.onSearchChange?.(value)}
          committeeId={
            typeof tableProps.activeFilters?.committee_id === 'string'
              ? tableProps.activeFilters.committee_id
              : undefined
          }
          onCommitteeChange={(value) => tableProps.onFilterChange?.('committee_id', value)}
          status={
            typeof tableProps.activeFilters?.status === 'string'
              ? tableProps.activeFilters.status
              : undefined
          }
          onStatusChange={(value) => tableProps.onFilterChange?.('status', value)}
          committees={committeesQuery.data?.data ?? []}
          loading={tableProps.isLoading}
        />

        {view === 'table' ? (
          <DataTable
            {...tableProps}
            columns={meetingColumns()}
            searchSlot={null}
            emptyState={{
              icon: CalendarDays,
              title: 'No meetings found',
              description: 'No meetings matched the current filters.',
            }}
          />
        ) : (
          <MeetingCalendar
            month={month}
            days={calendarQuery.data ?? []}
            onMonthChange={setMonth}
          />
        )}

        <ScheduleMeetingDialog
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          committees={committeesQuery.data?.data ?? []}
        />
      </div>
    </PermissionRedirect>
  );
}
