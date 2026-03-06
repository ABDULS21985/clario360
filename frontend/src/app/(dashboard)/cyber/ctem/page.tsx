import { Target } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function CyberCtemPage() {
  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader title="CTEM Assessments" description="Continuous threat exposure management" />
        <EmptyState
          icon={Target}
          title="CTEM Assessments"
          description="Continuous threat exposure management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
