'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { DataTable } from '@/components/shared/data-table/data-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Plus, Webhook } from 'lucide-react';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { useDeleteWebhook, useTestWebhook } from '@/hooks/use-webhooks';
import { showSuccess, showError } from '@/lib/toast';
import { CreateWebhookDialog } from './components/create-webhook-dialog';
import { WebhookSecretDialog } from './components/webhook-secret-dialog';
import { webhookColumns } from './components/webhook-columns';
import type { NotificationWebhook } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, RowAction, FilterConfig } from '@/types/table';

function buildApiParams(params: FetchParams): Record<string, unknown> {
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

const filters: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Inactive', value: 'inactive' },
      { label: 'Failing', value: 'failing' },
    ],
  },
];

export default function WebhooksPage() {
  const router = useRouter();
  const [createOpen, setCreateOpen] = useState(false);
  const [secretDialogData, setSecretDialogData] = useState<{ name: string; secret: string } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<NotificationWebhook | null>(null);
  const [testTarget, setTestTarget] = useState<NotificationWebhook | null>(null);

  const deleteMutation = useDeleteWebhook();
  const testMutation = useTestWebhook();

  const { tableProps, refetch } = useDataTable<NotificationWebhook>({
    fetchFn: (params: FetchParams) =>
      apiGet<PaginatedResponse<NotificationWebhook>>(
        API_ENDPOINTS.NOTIFICATIONS_WEBHOOKS,
        buildApiParams(params),
      ),
    queryKey: 'notification-webhooks',
    defaultSort: { column: 'created_at', direction: 'desc' },
  });

  const rowActions: RowAction<NotificationWebhook>[] = [
    {
      label: 'View Details',
      onClick: (row) => router.push(`/admin/notifications/webhooks/${row.id}`),
    },
    {
      label: 'Test',
      onClick: (row) => setTestTarget(row),
    },
    {
      label: 'Edit',
      onClick: (row) => router.push(`/admin/notifications/webhooks/${row.id}?tab=settings`),
    },
    {
      label: 'Delete',
      variant: 'destructive',
      onClick: (row) => setDeleteTarget(row),
    },
  ];

  const handleDelete = async () => {
    if (!deleteTarget) return;
    await deleteMutation.mutateAsync(deleteTarget.id);
    setDeleteTarget(null);
    refetch();
  };

  const handleTest = async () => {
    if (!testTarget) return;
    try {
      const result = await testMutation.mutateAsync(testTarget.id);
      if (result.success) {
        showSuccess('Test delivery successful', `Status: ${result.response_status}`);
      } else {
        showError('Test delivery failed', result.response_body);
      }
    } catch {
      // Error handled by mutation
    }
    setTestTarget(null);
  };

  const handleCreateSuccess = (name: string, secret: string) => {
    setCreateOpen(false);
    setSecretDialogData({ name, secret });
    refetch();
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Webhooks"
        description="Manage webhook endpoints for notification delivery."
        actions={
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create Webhook
          </Button>
        }
      />

      <DataTable
        columns={webhookColumns}
        {...tableProps}
        filters={filters}
        rowActions={rowActions}
        onRowClick={(row) => router.push(`/admin/notifications/webhooks/${row.id}`)}
        searchPlaceholder="Search webhooks..."
        emptyState={{
          icon: Webhook,
          title: 'No webhooks configured',
          description: 'Create a webhook to forward notifications to external services.',
          action: {
            label: 'Create Webhook',
            onClick: () => setCreateOpen(true),
            icon: Plus,
          },
        }}
      />

      <CreateWebhookDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={handleCreateSuccess}
      />

      <WebhookSecretDialog
        open={Boolean(secretDialogData)}
        onOpenChange={() => setSecretDialogData(null)}
        webhookName={secretDialogData?.name ?? ''}
        secret={secretDialogData?.secret ?? ''}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Webhook"
        description={`Are you sure you want to delete the webhook "${deleteTarget?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteMutation.isPending}
      />

      <ConfirmDialog
        open={Boolean(testTarget)}
        onOpenChange={() => setTestTarget(null)}
        title="Test Webhook"
        description={`Send a test payload to "${testTarget?.name}"?`}
        confirmLabel="Send Test"
        onConfirm={handleTest}
        loading={testMutation.isPending}
      />
    </div>
  );
}
