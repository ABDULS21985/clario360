import { Gavel } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function LexPage() {
  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Legal" description="Contract management, documents, and compliance" />
        <EmptyState
          icon={Gavel}
          title="Legal"
          description="Legal management module will be available in a future build."
        />
      </div>
    </PermissionRedirect>
  );
}
