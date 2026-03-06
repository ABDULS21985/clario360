import { ClipboardList } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function ActaActionItemsPage() {
  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Action Items" description="Track board action items and follow-ups" />
        <EmptyState
          icon={ClipboardList}
          title="Action Items"
          description="Action item tracking will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
