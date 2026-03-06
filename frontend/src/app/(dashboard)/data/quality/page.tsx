import { CheckCircle } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function DataQualityPage() {
  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader title="Data Quality" description="Monitor and improve data quality metrics" />
        <EmptyState
          icon={CheckCircle}
          title="Data Quality"
          description="Data quality monitoring will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
