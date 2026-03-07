'use client';

import Link from 'next/link';
import type { ColumnDef } from '@tanstack/react-table';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { meetingStatusConfig } from '@/lib/status-configs';
import type { ActaMeeting } from '@/types/suites';

export function meetingColumns(): ColumnDef<ActaMeeting>[] {
  return [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Meeting',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <Link href={`/acta/meetings/${row.original.id}`} className="font-medium hover:underline">
            {row.original.title}
          </Link>
          <p className="text-xs text-muted-foreground">{row.original.committee_name}</p>
        </div>
      ),
    },
    {
      id: 'scheduled_at',
      accessorKey: 'scheduled_at',
      header: 'Scheduled',
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.scheduled_at} />,
    },
    {
      id: 'attendance',
      header: 'Attendance',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.present_count}/{row.original.attendee_count}
        </span>
      ),
    },
    {
      id: 'quorum',
      header: 'Quorum',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.quorum_required} required
        </span>
      ),
    },
    {
      id: 'minutes_status',
      accessorKey: 'minutes_status',
      header: 'Minutes',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground capitalize">
          {row.original.minutes_status?.replace(/_/g, ' ') ?? 'Not started'}
        </span>
      ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={meetingStatusConfig} size="sm" />,
    },
  ];
}
