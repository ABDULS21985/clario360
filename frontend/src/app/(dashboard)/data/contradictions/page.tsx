import { Zap } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataContradictionsPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Contradictions" description="Identify and resolve data contradictions" />
        <EmptyState
          icon={Zap}
          title="Contradictions"
          description="Data contradiction detection will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
