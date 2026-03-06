'use client';

import { useParams } from 'next/navigation';
import { BookOpen } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function ActaMeetingDetailPage() {
  const { id } = useParams<{ id: string }>();
  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader title="Meeting Details" description={`Meeting ID: ${id}`} />
        <EmptyState
          icon={BookOpen}
          title="Meeting Details"
          description="Meeting details will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
