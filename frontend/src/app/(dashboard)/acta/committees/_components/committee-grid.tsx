'use client';

import { Building2 } from 'lucide-react';
import { EmptyState } from '@/components/common/empty-state';
import type { ActaCommittee } from '@/types/suites';
import { CommitteeCard } from './committee-card';

interface CommitteeGridProps {
  committees: ActaCommittee[];
}

export function CommitteeGrid({ committees }: CommitteeGridProps) {
  if (committees.length === 0) {
    return (
      <EmptyState
        icon={Building2}
        title="No committees found"
        description="Create the first governance committee to start managing board operations."
      />
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
      {committees.map((committee) => (
        <CommitteeCard key={committee.id} committee={committee} />
      ))}
    </div>
  );
}
