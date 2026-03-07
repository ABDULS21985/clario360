'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { ClipboardList } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { shortId } from '@/lib/suite-utils';
import type { ActaActionItem } from '@/types/suites';

const ACTION_FILTERS = [
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Open', value: 'open' },
      { label: 'In Progress', value: 'in_progress' },
      { label: 'Completed', value: 'completed' },
      { label: 'Blocked', value: 'blocked' },
    ],
  },
];

export default function ActaActionItemsPage() {
  const { tableProps } = useDataTable<ActaActionItem>({
    queryKey: 'acta-action-items',
    fetchFn: (params) => fetchSuitePaginated<ActaActionItem>(API_ENDPOINTS.ACTA_ACTION_ITEMS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const columns: ColumnDef<ActaActionItem>[] = [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Action Item',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.title}</p>
          <p className="text-xs text-muted-foreground">{row.original.meeting_title}</p>
        </div>
      ),
    },
    {
      id: 'assigned_to',
      accessorKey: 'assigned_to',
      header: 'Owner',
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{shortId(row.original.assigned_to)}</span>,
    },
    {
      id: 'due_date',
      accessorKey: 'due_date',
      header: 'Due',
      enableSorting: true,
      cell: ({ row }) =>
        row.original.due_date ? (
          <RelativeTime date={row.original.due_date} />
        ) : (
          <span className="text-sm text-muted-foreground">No due date</span>
        ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <span className="capitalize text-sm text-muted-foreground">{row.original.status.replace(/_/g, ' ')}</span>,
    },
    {
      id: 'updated_at',
      accessorKey: 'updated_at',
      header: 'Updated',
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.updated_at} />,
    },
  ];

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Action Items" description="Track board follow-ups, ownership, and due-date pressure." />
        <DataTable
          {...tableProps}
          columns={columns}
          filters={ACTION_FILTERS}
          emptyState={{
            icon: ClipboardList,
            title: 'No action items found',
            description: 'No board action items matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
