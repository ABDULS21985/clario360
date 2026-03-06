import { Search } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function CyberThreatsPage() {
  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader title="Threat Hunting" description="Proactively hunt for threats in your environment" />
        <EmptyState
          icon={Search}
          title="Threat Hunting"
          description="Threat hunting capabilities will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
