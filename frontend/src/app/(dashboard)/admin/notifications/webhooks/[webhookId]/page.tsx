'use client';

import { use } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { useNotificationWebhook } from '@/hooks/use-webhooks';
import { ArrowLeft } from 'lucide-react';
import { WebhookOverview } from './components/webhook-overview';
import { WebhookDeliveries } from './components/webhook-deliveries';
import { WebhookSettingsForm } from './components/webhook-settings-form';

interface WebhookDetailPageProps {
  params: Promise<{ webhookId: string }>;
}

export default function WebhookDetailPage({ params }: WebhookDetailPageProps) {
  const { webhookId } = use(params);
  const router = useRouter();
  const searchParams = useSearchParams();
  const activeTab = searchParams?.get('tab') ?? 'overview';

  const { data: webhook, isLoading, isError, refetch } = useNotificationWebhook(webhookId);

  const handleTabChange = (tab: string) => {
    const next = new URLSearchParams(searchParams?.toString() ?? '');
    if (tab === 'overview') {
      next.delete('tab');
    } else {
      next.set('tab', tab);
    }
    const path = `/admin/notifications/webhooks/${webhookId}`;
    router.push(next.toString() ? `${path}?${next.toString()}` : path);
  };

  if (isLoading) return <LoadingSkeleton variant="card" count={2} />;
  if (isError || !webhook) {
    return <ErrorState message="Failed to load webhook details" onRetry={() => refetch()} />;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => router.push('/admin/notifications/webhooks')}
          aria-label="Back to webhooks"
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <PageHeader
          title={webhook.name}
          description={webhook.url}
        />
      </div>

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="deliveries">Deliveries</TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-6">
          <WebhookOverview webhook={webhook} onRefresh={refetch} />
        </TabsContent>

        <TabsContent value="deliveries" className="mt-6">
          <WebhookDeliveries webhookId={webhookId} />
        </TabsContent>

        <TabsContent value="settings" className="mt-6">
          <WebhookSettingsForm webhook={webhook} onSaved={refetch} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
