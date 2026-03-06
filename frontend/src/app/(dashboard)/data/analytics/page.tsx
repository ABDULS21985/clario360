import { TrendingUp } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataAnalyticsPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Analytics" description="Advanced analytics and insights" />
        <EmptyState
          icon={TrendingUp}
          title="Analytics"
          description="Advanced analytics will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
