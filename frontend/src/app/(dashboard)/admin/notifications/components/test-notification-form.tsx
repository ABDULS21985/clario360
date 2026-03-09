'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useSendTestNotification } from '@/hooks/use-delivery-stats';
import { useNotificationWebhooks } from '@/hooks/use-webhooks';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { showSuccess, showApiError } from '@/lib/toast';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Send } from 'lucide-react';
import type { NotificationType } from '@/types/models';

const NOTIFICATION_TYPES: { value: NotificationType; label: string }[] = [
  { value: 'alert', label: 'Alert' },
  { value: 'task', label: 'Task' },
  { value: 'approval', label: 'Approval' },
  { value: 'system', label: 'System' },
  { value: 'mention', label: 'Mention' },
  { value: 'deadline', label: 'Deadline' },
  { value: 'completion', label: 'Completion' },
  { value: 'error', label: 'Error' },
  { value: 'report', label: 'Report' },
];

const CHANNELS = [
  { value: 'email', label: 'Email' },
  { value: 'in_app', label: 'In-App' },
  { value: 'push', label: 'Push' },
  { value: 'webhook', label: 'Webhook' },
] as const;

const testSchema = z.object({
  type: z.enum(['alert', 'task', 'approval', 'system', 'mention', 'deadline', 'completion', 'error', 'report']),
  channel: z.enum(['email', 'in_app', 'push', 'webhook']),
  webhook_id: z.string().optional(),
});

type TestFormData = z.infer<typeof testSchema>;

export function TestNotificationForm() {
  const form = useForm<TestFormData>({
    resolver: zodResolver(testSchema),
    defaultValues: {
      type: 'system',
      channel: 'in_app',
    },
  });

  const sendMutation = useSendTestNotification();
  const selectedChannel = form.watch('channel');

  const { data: webhooksData } = useNotificationWebhooks(
    selectedChannel === 'webhook' ? { page: 1, per_page: 100 } : undefined,
  );

  const onSubmit = async (data: TestFormData) => {
    try {
      const result = await sendMutation.mutateAsync({
        type: data.type,
        channel: data.channel,
        webhook_id: data.channel === 'webhook' ? data.webhook_id : undefined,
      });
      showSuccess('Test notification sent', result.message);
    } catch (error) {
      showApiError(error);
    }
  };

  return (
    <Card className="max-w-lg">
      <CardHeader>
        <CardTitle className="text-base">Send Test Notification</CardTitle>
        <CardDescription>
          Send a test notification to verify your delivery channels are working correctly.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label>Notification Type</Label>
            <Select
              value={form.watch('type')}
              onValueChange={(v) => form.setValue('type', v as NotificationType)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select type" />
              </SelectTrigger>
              <SelectContent>
                {NOTIFICATION_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>
                    {t.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {form.formState.errors.type && (
              <p className="text-xs text-destructive">{form.formState.errors.type.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label>Channel</Label>
            <div className="flex flex-wrap gap-2">
              {CHANNELS.map((ch) => (
                <Button
                  key={ch.value}
                  type="button"
                  variant={selectedChannel === ch.value ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => form.setValue('channel', ch.value)}
                >
                  {ch.label}
                </Button>
              ))}
            </div>
          </div>

          {selectedChannel === 'webhook' && (
            <div className="space-y-2">
              <Label>Webhook</Label>
              <Select
                value={form.watch('webhook_id') ?? ''}
                onValueChange={(v) => form.setValue('webhook_id', v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select webhook" />
                </SelectTrigger>
                <SelectContent>
                  {webhooksData?.data.map((wh) => (
                    <SelectItem key={wh.id} value={wh.id}>
                      {wh.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <Button
            type="submit"
            disabled={sendMutation.isPending}
            className="w-full"
          >
            <Send className="mr-2 h-4 w-4" />
            {sendMutation.isPending ? 'Sending...' : 'Send Test'}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
