import { Monitor } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function CyberAssetsPage() {
  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader title="Assets" description="Manage and monitor your asset inventory" />
        <EmptyState
          icon={Monitor}
          title="Asset Inventory"
          description="Asset discovery and management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
