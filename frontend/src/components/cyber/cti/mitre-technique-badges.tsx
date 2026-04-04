'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ExternalLink } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { buildMitreTechniqueHref } from '@/lib/cti-utils';
import type { MITRETechniqueItem } from '@/types/cyber';

interface MitreTechniqueBadgesProps {
  techniqueIds?: string[] | null;
  maxVisible?: number;
  emptyLabel?: string;
  onTechniqueClick?: (id: string) => void;
}

export function MitreTechniqueBadges({
  techniqueIds,
  maxVisible = 6,
  emptyLabel = 'No MITRE techniques mapped',
  onTechniqueClick,
}: MitreTechniqueBadgesProps) {
  const router = useRouter();
  const ids = techniqueIds ?? [];

  const techniquesQuery = useQuery({
    queryKey: ['cti-mitre-techniques'],
    queryFn: () => apiGet<{ data: MITRETechniqueItem[] }>(API_ENDPOINTS.CYBER_MITRE_TECHNIQUES),
    staleTime: 5 * 60_000,
  });

  const techniqueNameMap = useMemo(
    () => new Map((techniquesQuery.data?.data ?? []).map((item) => [item.id, item.name])),
    [techniquesQuery.data],
  );

  if (ids.length === 0) {
    return <p className="text-sm text-muted-foreground">{emptyLabel}</p>;
  }

  const visible = ids.slice(0, maxVisible);
  const hiddenCount = Math.max(ids.length - visible.length, 0);

  const handleTechniqueClick = (techniqueId: string) => {
    if (onTechniqueClick) {
      onTechniqueClick(techniqueId);
      return;
    }

    router.push(`/cyber/mitre-attack?technique=${encodeURIComponent(techniqueId)}`);
  };

  return (
    <div className="flex flex-wrap gap-2">
      {visible.map((techniqueId) => {
        const techniqueName = techniqueNameMap.get(techniqueId);

        return (
          <Badge key={techniqueId} variant="outline" className="gap-1 px-2.5 py-1 text-xs">
            <button
              type="button"
              className="inline-flex items-center gap-1"
              onClick={() => handleTechniqueClick(techniqueId)}
              title={techniqueName ? `${techniqueId} · ${techniqueName}` : techniqueId}
              aria-label={techniqueName ? `${techniqueId} ${techniqueName}` : techniqueId}
            >
              {techniqueId}
            </button>
            <a
              href={buildMitreTechniqueHref(techniqueId)}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center"
              aria-label={`Open ${techniqueId} on MITRE ATT&CK`}
              onClick={(event) => event.stopPropagation()}
            >
              <ExternalLink className="h-3 w-3" />
            </a>
          </Badge>
        );
      })}
      {hiddenCount > 0 && (
        <Badge variant="outline" className="px-2.5 py-1 text-xs text-muted-foreground">
          +{hiddenCount} more
        </Badge>
      )}
    </div>
  );
}
