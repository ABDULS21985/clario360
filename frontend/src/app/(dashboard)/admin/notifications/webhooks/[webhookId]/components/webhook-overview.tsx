'use client';

import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { RelativeTime } from '@/components/shared/relative-time';
import { useTestWebhook, useRotateWebhookSecret } from '@/hooks/use-webhooks';
import { WebhookSecretDialog } from '../../components/webhook-secret-dialog';
import { showSuccess, showError, showApiError } from '@/lib/toast';
import { Send, RotateCw, KeyRound, CheckCircle, XCircle, Clock } from 'lucide-react';
import type { NotificationWebhook } from '@/types/models';

const statusVariants: Record<string, 'success' | 'secondary' | 'destructive'> = {
  active: 'success',
  inactive: 'secondary',
  failing: 'destructive',
};

interface WebhookOverviewProps {
  webhook: NotificationWebhook;
  onRefresh: () => void;
}

export function WebhookOverview({ webhook, onRefresh }: WebhookOverviewProps) {
  const [testOpen, setTestOpen] = useState(false);
  const [rotateOpen, setRotateOpen] = useState(false);
  const [secretData, setSecretData] = useState<{ name: string; secret: string } | null>(null);

  const testMutation = useTestWebhook();
  const rotateMutation = useRotateWebhookSecret();

  const handleTest = async () => {
    try {
      const result = await testMutation.mutateAsync(webhook.id);
      if (result.success) {
        showSuccess('Test delivery successful', `HTTP ${result.response_status}`);
      } else {
        showError('Test delivery failed', result.response_body);
      }
    } catch (error) {
      showApiError(error);
    }
    setTestOpen(false);
  };

  const handleRotate = async () => {
    try {
      const result = await rotateMutation.mutateAsync(webhook.id);
      setRotateOpen(false);
      setSecretData({ name: webhook.name, secret: result.secret });
      onRefresh();
    } catch (error) {
      showApiError(error);
    }
  };

  const successRate =
    webhook.success_count + webhook.failure_count > 0
      ? ((webhook.success_count / (webhook.success_count + webhook.failure_count)) * 100).toFixed(1)
      : '—';

  return (
    <div className="space-y-6">
      {/* Actions */}
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" size="sm" onClick={() => setTestOpen(true)}>
          <Send className="mr-2 h-3.5 w-3.5" />
          Test
        </Button>
        <Button variant="outline" size="sm" onClick={() => setRotateOpen(true)}>
          <KeyRound className="mr-2 h-3.5 w-3.5" />
          Rotate Secret
        </Button>
      </div>

      {/* Status and Config Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Status</CardTitle>
          </CardHeader>
          <CardContent>
            <Badge variant={statusVariants[webhook.status] ?? 'secondary'} className="text-sm">
              {webhook.status}
            </Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Success</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{webhook.success_count}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Failures</CardTitle>
            <XCircle className="h-4 w-4 text-destructive" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{webhook.failure_count}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{successRate}%</div>
          </CardContent>
        </Card>
      </div>

      {/* Details */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <div className="grid grid-cols-[120px_1fr] gap-2">
            <span className="text-muted-foreground">URL</span>
            <span className="break-all font-mono text-xs">{webhook.url}</span>
          </div>
          <div className="grid grid-cols-[120px_1fr] gap-2">
            <span className="text-muted-foreground">Events</span>
            <div className="flex flex-wrap gap-1">
              {webhook.events.map((event) => (
                <Badge key={event} variant="outline" className="text-xs">
                  {event}
                </Badge>
              ))}
            </div>
          </div>
          <div className="grid grid-cols-[120px_1fr] gap-2">
            <span className="text-muted-foreground">Retry Policy</span>
            <span>
              {webhook.retry_policy.max_retries} retries, {webhook.retry_policy.backoff_type} backoff,{' '}
              {webhook.retry_policy.initial_delay_seconds}s delay
            </span>
          </div>
          <div className="grid grid-cols-[120px_1fr] gap-2">
            <span className="text-muted-foreground">Last Triggered</span>
            <span>
              {webhook.last_triggered_at ? (
                <RelativeTime date={webhook.last_triggered_at} />
              ) : (
                'Never'
              )}
            </span>
          </div>
          {Object.keys(webhook.headers).length > 0 && (
            <div className="grid grid-cols-[120px_1fr] gap-2">
              <span className="text-muted-foreground">Headers</span>
              <div className="space-y-1">
                {Object.entries(webhook.headers).map(([key, value]) => (
                  <p key={key} className="font-mono text-xs">
                    {key}: {value}
                  </p>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Dialogs */}
      <ConfirmDialog
        open={testOpen}
        onOpenChange={setTestOpen}
        title="Test Webhook"
        description={`Send a test payload to "${webhook.name}"?`}
        confirmLabel="Send Test"
        onConfirm={handleTest}
        loading={testMutation.isPending}
      />

      <ConfirmDialog
        open={rotateOpen}
        onOpenChange={setRotateOpen}
        title="Rotate Secret"
        description="This will generate a new webhook secret. The old secret will stop working immediately. Continue?"
        confirmLabel="Rotate"
        variant="destructive"
        onConfirm={handleRotate}
        loading={rotateMutation.isPending}
      />

      <WebhookSecretDialog
        open={Boolean(secretData)}
        onOpenChange={() => setSecretData(null)}
        webhookName={secretData?.name ?? ''}
        secret={secretData?.secret ?? ''}
      />
    </div>
  );
}
