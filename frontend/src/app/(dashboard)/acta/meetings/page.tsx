'use client';

import Link from 'next/link';
import { type ColumnDef } from '@tanstack/react-table';
import { BookOpen } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { meetingStatusConfig } from '@/lib/status-configs';
import { summarizeNamedRecords } from '@/lib/suite-utils';
import type { ActaMeeting } from '@/types/suites';

const MEETING_FILTERS = [
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Scheduled', value: 'scheduled' },
      { label: 'In Progress', value: 'in_progress' },
      { label: 'Completed', value: 'completed' },
      { label: 'Cancelled', value: 'cancelled' },
    ],
  },
];

export default function ActaMeetingsPage() {
  const { tableProps } = useDataTable<ActaMeeting>({
    queryKey: 'acta-meetings',
    fetchFn: (params) => fetchSuitePaginated<ActaMeeting>(API_ENDPOINTS.ACTA_MEETINGS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'scheduled_at', direction: 'desc' },
  });

  const columns: ColumnDef<ActaMeeting>[] = [
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
      id: 'attendees',
      header: 'Attendees',
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{summarizeNamedRecords(row.original.attendees)}</span>,
    },
    {
      id: 'action_item_count',
      accessorKey: 'action_item_count',
      header: 'Actions',
      enableSorting: true,
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{row.original.action_item_count}</span>,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={meetingStatusConfig} size="sm" />,
    },
  ];

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Meetings" description="Upcoming and historical board meetings with linked follow-up activity." />
        <DataTable
          {...tableProps}
          columns={columns}
          filters={MEETING_FILTERS}
          emptyState={{
            icon: BookOpen,
            title: 'No meetings found',
            description: 'No meetings matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
