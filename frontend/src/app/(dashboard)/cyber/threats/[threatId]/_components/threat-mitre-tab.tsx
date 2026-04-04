'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Badge } from '@/components/ui/badge';
import { Shield } from 'lucide-react';
import type { MITRETacticItem, MITRETechniqueItem, Threat } from '@/types/cyber';

interface ThreatMitreTabProps {
  threat: Threat;
}

export function ThreatMitreTab({ threat }: ThreatMitreTabProps) {
  const tacticsQuery = useQuery({
    queryKey: ['mitre-tactics-threat-detail'],
    queryFn: () => apiGet<{ data: MITRETacticItem[] }>(API_ENDPOINTS.CYBER_MITRE_TACTICS),
    staleTime: 300000,
  });
  const techniquesQuery = useQuery({
    queryKey: ['mitre-techniques-threat-detail'],
    queryFn: () => apiGet<{ data: MITRETechniqueItem[] }>(API_ENDPOINTS.CYBER_MITRE_TECHNIQUES),
    staleTime: 300000,
  });

  const mappedTechniques = useMemo(() => {
    const byId = new Map((techniquesQuery.data?.data ?? []).map((item) => [item.id, item]));
    return threat.mitre_technique_ids
      .map((id) => byId.get(id))
      .filter((item): item is MITRETechniqueItem => Boolean(item));
  }, [techniquesQuery.data, threat.mitre_technique_ids]);

  if (tacticsQuery.isLoading || techniquesQuery.isLoading) {
    return <LoadingSkeleton variant="card" count={2} />;
  }
  if (tacticsQuery.error || techniquesQuery.error) {
    return <ErrorState message="Failed to load MITRE mapping" onRetry={() => {
      void tacticsQuery.refetch();
      void techniquesQuery.refetch();
    }} />;
  }

  const tactics = tacticsQuery.data?.data ?? [];
  const highlightedTactics = tactics.filter((item) => threat.mitre_tactic_ids.includes(item.id));

  if (highlightedTactics.length === 0 && mappedTechniques.length === 0) {
    return (
      <EmptyState
        icon={Shield}
        title="No MITRE mapping"
        description="This threat has not been mapped to ATT&CK tactics or techniques yet."
      />
    );
  }

  return (
    <div className="space-y-6">
      <section className="rounded-[24px] border bg-card p-5">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold">ATT&CK Matrix Slice</h3>
            <p className="text-sm text-muted-foreground">
              Tactics highlighted for this threat and the techniques currently mapped beneath them.
            </p>
          </div>
          <Badge variant="secondary">{mappedTechniques.length} techniques</Badge>
        </div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {highlightedTactics.map((tactic) => {
            const techniques = mappedTechniques.filter((technique) => technique.tactic_ids.includes(tactic.id));
            return (
              <div key={tactic.id} className="rounded-[20px] border border-emerald-200 bg-emerald-50/60 p-4">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.18em] text-emerald-800">{tactic.id}</p>
                    <h4 className="mt-1 font-semibold text-emerald-950">{tactic.name}</h4>
                  </div>
                  <Badge variant="outline" className="border-emerald-300 text-emerald-900">
                    {techniques.length}
                  </Badge>
                </div>
                <p className="mt-2 text-sm text-emerald-900/80">{tactic.description}</p>
                <div className="mt-4 flex flex-wrap gap-2">
                  {techniques.length > 0 ? techniques.map((technique) => (
                    <Badge key={technique.id} variant="outline" className="border-emerald-300 bg-white/80 font-mono">
                      {technique.id}
                    </Badge>
                  )) : (
                    <span className="text-xs text-emerald-900/70">No techniques mapped for this tactic yet.</span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </section>

      <section className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        {mappedTechniques.map((technique) => (
          <article key={technique.id} className="rounded-[24px] border bg-card p-5">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{technique.id}</p>
                <h4 className="mt-1 text-base font-semibold">{technique.name}</h4>
              </div>
              <div className="flex flex-wrap gap-2">
                {technique.tactic_ids.map((id) => (
                  <Badge key={id} variant="secondary">{id}</Badge>
                ))}
              </div>
            </div>
            <p className="mt-3 text-sm leading-6 text-muted-foreground">{technique.description}</p>
            {(technique.platforms?.length ?? 0) > 0 && (
              <div className="mt-4">
                <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Platforms</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {technique.platforms?.map((platform) => (
                    <Badge key={platform} variant="outline">{platform}</Badge>
                  ))}
                </div>
              </div>
            )}
          </article>
        ))}
      </section>
    </div>
  );
}
