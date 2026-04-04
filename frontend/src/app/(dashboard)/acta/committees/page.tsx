'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SearchInput } from '@/components/shared/forms/search-input';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { enterpriseApi } from '@/lib/enterprise';
import { CommitteeGrid } from './_components/committee-grid';
import { CreateCommitteeDialog } from './_components/create-committee-dialog';

export default function ActaCommitteesPage() {
  const [search, setSearch] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const committeesQuery = useQuery({
    queryKey: ['acta-committees', search],
    queryFn: () =>
      enterpriseApi.acta.listCommittees({
        page: 1,
        per_page: 100,
        order: 'desc',
        search: search || undefined,
      }),
  });

  if (committeesQuery.isLoading) {
    return (
      <PermissionRedirect permission="acta:read">
        <LoadingSkeleton variant="card" count={6} />
      </PermissionRedirect>
    );
  }

  if (committeesQuery.error) {
    return (
      <PermissionRedirect permission="acta:read">
        <ErrorState message="Failed to load committees." onRetry={() => void committeesQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title="Committees"
          description="Board and governance committee roster, cadence, and operating profile."
          actions={<CreateCommitteeDialog open={dialogOpen} onOpenChange={setDialogOpen} />}
        />
        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search committees..."
          loading={committeesQuery.isFetching}
        />
        <CommitteeGrid committees={committeesQuery.data?.data ?? []} />
      </div>
    </PermissionRedirect>
  );
}
