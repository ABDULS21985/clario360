import { FolderOpen } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataSourcesPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Data Sources" description="Manage your data source connections" />
        <EmptyState
          icon={FolderOpen}
          title="Data Sources"
          description="Data source management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
