import { Eye } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function VisusPage() {
  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader title="Executive Intelligence" description="Executive dashboards and reports" />
        <EmptyState
          icon={Eye}
          title="Visus360"
          description="Executive intelligence module will be available in a future build."
        />
      </div>
    </PermissionRedirect>
  );
}
