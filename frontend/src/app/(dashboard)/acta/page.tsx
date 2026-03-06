import { Building2 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function ActaPage() {
  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Board Governance" description="Board committees, meetings, and action items" />
        <EmptyState
          icon={Building2}
          title="Board Governance"
          description="Board governance module will be available in a future build."
        />
      </div>
    </PermissionRedirect>
  );
}
