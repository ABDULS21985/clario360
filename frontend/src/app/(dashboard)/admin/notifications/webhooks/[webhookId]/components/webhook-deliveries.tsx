'use client';

import { useState } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { DataTable } from '@/components/shared/data-table/data-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { RelativeTime } from '@/components/shared/relative-time';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { showSuccess, showApiError } from '@/lib/toast';
import { RotateCw, Inbox } from 'lucide-react';
import type { WebhookDelivery } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, RowAction, FilterConfig } from '@/types/table';

const statusVariants: Record<string, 'success' | 'secondary' | 'destructive' | 'warning'> = {
  success: 'success',
  failed: 'destructive',
  pending: 'secondary',
  retrying: 'warning',
};

const columns: ColumnDef<WebhookDelivery>[] = [
  {
    accessorKey: 'event_type',
    header: 'Event Type',
    cell: ({ row }) => (
      <Badge variant="outline" className="text-xs font-mono">
        {row.original.event_type}
      </Badge>
    ),
    enableSorting: true,
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => (
      <Badge variant={statusVariants[row.original.status] ?? 'secondary'}>
        {row.original.status}
      </Badge>
    ),
    enableSorting: true,
  },
  {
    accessorKey: 'response_status',
    header: 'HTTP Status',
    cell: ({ row }) => {
      const status = row.original.response_status;
      if (status === null) return <span className="text-xs text-muted-foreground">—</span>;
      return (
        <span className={status >= 200 && status < 300 ? 'text-green-600' : 'text-destructive'}>
          {status}
        </span>
      );
    },
  },
  {
    accessorKey: 'duration_ms',
    header: 'Duration',
    cell: ({ row }) => {
      const ms = row.original.duration_ms;
      if (ms === null) return <span className="text-xs text-muted-foreground">—</span>;
      return <span className="text-xs">{ms}ms</span>;
    },
  },
  {
    accessorKey: 'attempt_count',
    header: 'Attempts',
    cell: ({ row }) => <span className="text-xs">{row.original.attempt_count}</span>,
  },
  {
    accessorKey: 'created_at',
    header: 'Created',
    cell: ({ row }) => <RelativeTime date={row.original.created_at} className="text-xs" />,
    enableSorting: true,
  },
];

const filters: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Success', value: 'success' },
      { label: 'Failed', value: 'failed' },
      { label: 'Pending', value: 'pending' },
      { label: 'Retrying', value: 'retrying' },
    ],
  },
];

function buildParams(params: FetchParams): Record<string, unknown> {
  const result: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
  };
  if (params.sort) result.sort = params.sort;
  if (params.order) result.order = params.order;
  if (params.search) result.search = params.search;
  if (params.filters) {
    for (const [key, value] of Object.entries(params.filters)) {
      result[key] = value;
    }
  }
  return result;
}

interface WebhookDeliveriesProps {
  webhookId: string;
}

export function WebhookDeliveries({ webhookId }: WebhookDeliveriesProps) {
  const [detailDelivery, setDetailDelivery] = useState<WebhookDelivery | null>(null);

  const { tableProps, refetch } = useDataTable<WebhookDelivery>({
    fetchFn: (params: FetchParams) =>
      apiGet<PaginatedResponse<WebhookDelivery>>(
        `${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}/deliveries`,
        buildParams(params),
      ),
    queryKey: `webhook-deliveries-${webhookId}`,
    defaultSort: { column: 'created_at', direction: 'desc' },
  });

  const handleRetryDelivery = async (deliveryId: string) => {
    try {
      await apiPost(`${API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS}/${webhookId}/deliveries/${deliveryId}/retry`);
      showSuccess('Delivery retry queued');
      refetch();
    } catch (error) {
      showApiError(error);
    }
  };

  const rowActions: RowAction<WebhookDelivery>[] = [
    {
      label: 'View Detail',
      onClick: (row) => setDetailDelivery(row),
    },
    {
      label: 'Retry',
      onClick: (row) => handleRetryDelivery(row.id),
      hidden: (row) => row.status !== 'failed',
    },
  ];

  return (
    <div className="space-y-4">
      <DataTable
        columns={columns}
        {...tableProps}
        filters={filters}
        rowActions={rowActions}
        onRowClick={(row) => setDetailDelivery(row)}
        searchPlaceholder="Search deliveries..."
        emptyState={{
          icon: Inbox,
          title: 'No deliveries yet',
          description: 'Deliveries will appear here once events are triggered.',
        }}
      />

      {/* Delivery Detail Dialog */}
      <Dialog open={Boolean(detailDelivery)} onOpenChange={() => setDetailDelivery(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Delivery Detail</DialogTitle>
          </DialogHeader>
          {detailDelivery && (
            <div className="space-y-4 text-sm max-h-[60vh] overflow-y-auto">
              <div className="grid grid-cols-[100px_1fr] gap-2">
                <span className="text-muted-foreground">Event</span>
                <span className="font-mono text-xs">{detailDelivery.event_type}</span>
              </div>
              <div className="grid grid-cols-[100px_1fr] gap-2">
                <span className="text-muted-foreground">Status</span>
                <Badge variant={statusVariants[detailDelivery.status] ?? 'secondary'}>
                  {detailDelivery.status}
                </Badge>
              </div>
              <div className="grid grid-cols-[100px_1fr] gap-2">
                <span className="text-muted-foreground">HTTP Status</span>
                <span>{detailDelivery.response_status ?? '—'}</span>
              </div>
              <div className="grid grid-cols-[100px_1fr] gap-2">
                <span className="text-muted-foreground">Duration</span>
                <span>{detailDelivery.duration_ms !== null ? `${detailDelivery.duration_ms}ms` : '—'}</span>
              </div>
              <div className="grid grid-cols-[100px_1fr] gap-2">
                <span className="text-muted-foreground">Attempts</span>
                <span>{detailDelivery.attempt_count}</span>
              </div>
              {detailDelivery.next_retry_at && (
                <div className="grid grid-cols-[100px_1fr] gap-2">
                  <span className="text-muted-foreground">Next Retry</span>
                  <RelativeTime date={detailDelivery.next_retry_at} />
                </div>
              )}
              <div>
                <p className="mb-1 text-muted-foreground">Request Body</p>
                <pre className="overflow-auto rounded-md border bg-muted/50 p-3 text-xs">
                  {JSON.stringify(detailDelivery.request_body, null, 2)}
                </pre>
              </div>
              {detailDelivery.response_body && (
                <div>
                  <p className="mb-1 text-muted-foreground">Response Body</p>
                  <pre className="overflow-auto rounded-md border bg-muted/50 p-3 text-xs">
                    {detailDelivery.response_body}
                  </pre>
                </div>
              )}
              {detailDelivery.status === 'failed' && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    handleRetryDelivery(detailDelivery.id);
                    setDetailDelivery(null);
                  }}
                >
                  <RotateCw className="mr-2 h-3.5 w-3.5" />
                  Retry This Delivery
                </Button>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
