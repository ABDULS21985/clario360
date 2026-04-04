'use client';

import Link from 'next/link';
import { ArrowRight, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { SectionCard } from '@/components/suites/section-card';
import { EmptyState } from '@/components/common/empty-state';
import { cn } from '@/lib/utils';
import type { ActaCommitteeCompliance } from '@/types/suites';

interface ActaComplianceBarsProps {
  items: ActaCommitteeCompliance[];
}

export function ActaComplianceBars({ items }: ActaComplianceBarsProps) {
  return (
    <SectionCard
      title="Compliance By Committee"
      description="Committee-level governance scorecards from the latest checks."
      actions={
        <Button variant="ghost" size="sm" asChild>
          <Link href="/acta/compliance">
            Full report
            <ArrowRight className="ml-1 h-3.5 w-3.5" />
          </Link>
        </Button>
      }
    >
      {items.length === 0 ? (
        <EmptyState
          icon={Shield}
          title="No compliance data"
          description="Run the Acta compliance engine to populate committee scorecards."
        />
      ) : (
        <div className="space-y-4">
          {items.map((item) => (
            <Link
              key={item.committee_id}
              href={`/acta/compliance?committee=${item.committee_id}`}
              className="block rounded-xl border px-4 py-3 transition hover:border-primary"
            >
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate font-medium">{item.committee_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {item.non_compliant} non-compliant • {item.warnings} warnings
                  </p>
                </div>
                <div className="text-sm font-semibold">{Math.round(item.score)}%</div>
              </div>
              <Progress
                value={item.score}
                indicatorClassName={cn(
                  item.score >= 85
                    ? 'bg-emerald-500'
                    : item.score >= 70
                      ? 'bg-amber-500'
                      : 'bg-rose-500',
                )}
                className="mt-3 h-2"
              />
            </Link>
          ))}
        </div>
      )}
    </SectionCard>
  );
}
