'use client';

import { useParams } from 'next/navigation';
import { FileText } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';

export default function LexContractDetailPage() {
  const { id } = useParams<{ id: string }>();
  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Contract Details" description={`Contract ID: ${id}`} />
        <EmptyState
          icon={FileText}
          title="Contract Details"
          description="Contract details will be available in a future module."
        />
      </div>
    </PermissionRedirect>
  );
}
