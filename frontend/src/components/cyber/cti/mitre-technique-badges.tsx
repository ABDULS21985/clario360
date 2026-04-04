'use client';

import { ExternalLink } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { buildMitreTechniqueHref } from '@/lib/cti-utils';

interface MitreTechniqueBadgesProps {
  techniqueIds?: string[] | null;
  emptyLabel?: string;
}

export function MitreTechniqueBadges({
  techniqueIds,
  emptyLabel = 'No MITRE techniques mapped',
}: MitreTechniqueBadgesProps) {
  if (!techniqueIds || techniqueIds.length === 0) {
    return <p className="text-sm text-muted-foreground">{emptyLabel}</p>;
  }

  return (
    <div className="flex flex-wrap gap-2">
      {techniqueIds.map((techniqueId) => (
        <Badge key={techniqueId} variant="outline" className="gap-1 px-2.5 py-1 text-xs">
          <a
            href={buildMitreTechniqueHref(techniqueId)}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1"
          >
            {techniqueId}
            <ExternalLink className="h-3 w-3" />
          </a>
        </Badge>
      ))}
    </div>
  );
}