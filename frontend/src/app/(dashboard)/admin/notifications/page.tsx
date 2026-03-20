'use client';

import { useSearchParams, useRouter, usePathname } from 'next/navigation';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { PageHeader } from '@/components/common/page-header';
import { DeliveryDashboard } from './components/delivery-dashboard';
import { TestNotificationForm } from './components/test-notification-form';

export default function AdminNotificationsPage() {
  const router = useRouter();
  const pathname = usePathname();
  const currentPath = pathname ?? '/admin/notifications';
  const searchParams = useSearchParams();
  const activeTab = searchParams?.get('tab') ?? 'dashboard';

  const handleTabChange = (tab: string) => {
    const next = new URLSearchParams(searchParams?.toString() ?? '');
    if (tab === 'dashboard') {
      next.delete('tab');
    } else {
      next.set('tab', tab);
    }
    router.push(next.toString() ? `${currentPath}?${next.toString()}` : currentPath);
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Notification Management"
        description="Monitor delivery performance, manage webhooks, and test notifications."
      />

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          <TabsTrigger value="dashboard">Dashboard</TabsTrigger>
          <TabsTrigger value="test">Test</TabsTrigger>
        </TabsList>

        <TabsContent value="dashboard" className="mt-6">
          <DeliveryDashboard />
        </TabsContent>

        <TabsContent value="test" className="mt-6">
          <TestNotificationForm />
        </TabsContent>
      </Tabs>
    </div>
  );
}
