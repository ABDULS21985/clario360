import { BookOpen } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function ActaMeetingsPage() {
  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Meetings" description="Schedule and manage board meetings" />
        <EmptyState
          icon={BookOpen}
          title="Meetings"
          description="Meeting management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
