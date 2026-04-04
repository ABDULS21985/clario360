'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { RelativeTime } from '@/components/shared/relative-time';
import type { NotificationWebhook } from '@/types/models';

const statusVariants: Record<string, 'success' | 'secondary' | 'destructive'> = {
  active: 'success',
  inactive: 'secondary',
  failing: 'destructive',
};

export const webhookColumns: ColumnDef<NotificationWebhook>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => (
      <span className="font-medium">{row.original.name}</span>
    ),
    enableSorting: true,
  },
  {
    accessorKey: 'url',
    header: 'URL',
    cell: ({ row }) => {
      const url = row.original.url;
      const truncated = url.length > 50 ? `${url.slice(0, 50)}...` : url;
      return (
        <span className="font-mono text-xs text-muted-foreground" title={url}>
          {truncated}
        </span>
      );
    },
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.original.status;
      return (
        <Badge variant={statusVariants[status] ?? 'secondary'}>
          {status}
        </Badge>
      );
    },
    enableSorting: true,
  },
  {
    accessorKey: 'events',
    header: 'Events',
    cell: ({ row }) => {
      const events = row.original.events;
      const maxShow = 3;
      const visible = events.slice(0, maxShow);
      const remaining = events.length - maxShow;
      return (
        <div className="flex flex-wrap items-center gap-1">
          {visible.map((event) => (
            <Badge key={event} variant="outline" className="text-xs">
              {event}
            </Badge>
          ))}
          {remaining > 0 && (
            <Badge variant="secondary" className="text-xs">
              +{remaining} more
            </Badge>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: 'last_triggered_at',
    header: 'Last Triggered',
    cell: ({ row }) => {
      const date = row.original.last_triggered_at;
      if (!date) return <span className="text-xs text-muted-foreground">Never</span>;
      return <RelativeTime date={date} className="text-xs" />;
    },
    enableSorting: true,
  },
  {
    id: 'stats',
    header: 'Success / Failure',
    cell: ({ row }) => (
      <div className="flex items-center gap-2 text-xs">
        <span className="text-green-600">{row.original.success_count}</span>
        <span className="text-muted-foreground">/</span>
        <span className="text-destructive">{row.original.failure_count}</span>
      </div>
    ),
  },
];
