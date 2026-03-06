import { Boxes } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataModelsPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Data Models" description="Define and manage your data models" />
        <EmptyState
          icon={Boxes}
          title="Data Models"
          description="Data model management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
