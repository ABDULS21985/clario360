import { Users } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function ActaCommitteesPage() {
  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Committees" description="Manage board committees and membership" />
        <EmptyState
          icon={Users}
          title="Committees"
          description="Committee management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
