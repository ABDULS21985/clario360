import { Bot } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function CyberVcisoPage() {
  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader title="Virtual CISO" description="AI-powered security advisory" />
        <EmptyState
          icon={Bot}
          title="Virtual CISO"
          description="AI-powered CISO advisory will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
