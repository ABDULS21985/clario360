import { BarChart3 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Data Intelligence" description="Data pipelines, quality monitoring, and dataset management" />
        <EmptyState
          icon={BarChart3}
          title="Data Intelligence"
          description="Data intelligence capabilities will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
