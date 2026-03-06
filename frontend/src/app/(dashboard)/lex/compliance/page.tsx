import { Scale } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function LexCompliancePage() {
  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Compliance" description="Regulatory compliance tracking" />
        <EmptyState
          icon={Scale}
          title="Compliance"
          description="Compliance management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
