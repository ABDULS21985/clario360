import { GitBranch } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataPipelinesPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Pipelines" description="Monitor and manage data pipelines" />
        <EmptyState
          icon={GitBranch}
          title="Pipelines"
          description="Pipeline management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
