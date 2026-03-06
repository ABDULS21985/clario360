import { Package } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataDarkDataPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Dark Data" description="Discover and classify unstructured dark data" />
        <EmptyState
          icon={Package}
          title="Dark Data"
          description="Dark data discovery will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
