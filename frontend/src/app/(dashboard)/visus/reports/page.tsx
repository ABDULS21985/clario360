import { FileBarChart } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function VisusReportsPage() {
  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader title="Reports" description="Executive reports and analytics" />
        <EmptyState
          icon={FileBarChart}
          title="Reports"
          description="Executive reports will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
