import { FileText } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function LexContractsPage() {
  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Contracts" description="Manage and track legal contracts" />
        <EmptyState
          icon={FileText}
          title="Contracts"
          description="Contract management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
