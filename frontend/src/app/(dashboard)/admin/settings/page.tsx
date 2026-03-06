import { Settings } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function AdminSettingsPage() {
  return (
    <PermissionRedirect permission="tenant:write">
      <div className="space-y-6">
        <PageHeader title="Platform Settings" description="Configure tenant-wide settings" />
        <EmptyState
          icon={Settings}
          title="Settings"
          description="Platform configuration will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
