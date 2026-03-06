import { Shield } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function CyberPage() {
  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader title="Cybersecurity" description="Threat detection, vulnerability management, and security monitoring" />
        <EmptyState
          icon={Shield}
          title="Cybersecurity Overview"
          description="Cybersecurity module details will be available in a future build."
        />
      </div>
    </PermissionRedirect>
  );
}
