import { Database } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function CyberDspmPage() {
  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader title="DSPM" description="Data security posture management" />
        <EmptyState
          icon={Database}
          title="Data Security Posture Management"
          description="DSPM capabilities will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
