import { File } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function LexDocumentsPage() {
  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Documents" description="Legal document repository" />
        <EmptyState
          icon={File}
          title="Documents"
          description="Legal document management will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
