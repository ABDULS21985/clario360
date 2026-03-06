import { Wrench } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function CyberRemediationPage() {
  return (
    <PermissionRedirect permission="remediation:read">
      <div className="space-y-6">
        <PageHeader title="Remediation" description="Track and manage security remediation tasks" />
        <EmptyState
          icon={Wrench}
          title="Remediation"
          description="Remediation tracking will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
